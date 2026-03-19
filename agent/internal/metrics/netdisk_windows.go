//go:build windows

package metrics

import (
	"math"
	"sync"
	"syscall"
	"unsafe"
)

// Minimal PDH bindings (English counters to avoid localization issues)
// Uses:
//   \\Network Interface(_Total)\\Bytes Received/sec
//   \\Network Interface(_Total)\\Bytes Sent/sec
//   \\PhysicalDisk(_Total)\\Disk Read Bytes/sec
//   \\PhysicalDisk(_Total)\\Disk Write Bytes/sec

const pdhFmtDouble = 0x00000200

type pdhStatus uint32

type pdhCounterValue struct {
	CStatus     uint32
	DoubleValue float64
}

var (
	pdhOnce    sync.Once
	pdhInitErr error

	pdhQuery      syscall.Handle
	pdhNetRx      syscall.Handle
	pdhNetTx      syscall.Handle
	pdhDiskRead   syscall.Handle
	pdhDiskWrite  syscall.Handle

	pdhNetRxList    []syscall.Handle
	pdhNetTxList    []syscall.Handle
	pdhDiskReadList []syscall.Handle
	pdhDiskWriteList []syscall.Handle
)

func pdhInit() {
	pdhOnce.Do(func() {
		pdh := syscall.NewLazyDLL("pdh.dll")
		openQuery := pdh.NewProc("PdhOpenQueryW")
		addEngCounter := pdh.NewProc("PdhAddEnglishCounterW")
		addCounter := pdh.NewProc("PdhAddCounterW")
		expandWild := pdh.NewProc("PdhExpandWildCardPathW")
		collect := pdh.NewProc("PdhCollectQueryData")

		// open query
		r1, _, err := openQuery.Call(0, 0, uintptr(unsafe.Pointer(&pdhQuery)))
		if r1 != 0 {
			pdhInitErr = err
			return
		}

		add := func(path string, out *syscall.Handle) error {
			p, _ := syscall.UTF16PtrFromString(path)
			r, _, e := addEngCounter.Call(uintptr(pdhQuery), uintptr(unsafe.Pointer(p)), 0, uintptr(unsafe.Pointer(out)))
			if r != 0 {
				return e
			}
			return nil
		}

		// Try English _Total instance for aggregated counters
		_ = add(`\\Network Interface(_Total)\\Bytes Received/sec`, &pdhNetRx)
		_ = add(`\\Network Interface(_Total)\\Bytes Sent/sec`, &pdhNetTx)
		_ = add(`\\PhysicalDisk(_Total)\\Disk Read Bytes/sec`, &pdhDiskRead)
		_ = add(`\\PhysicalDisk(_Total)\\Disk Write Bytes/sec`, &pdhDiskWrite)

		// If English counters not available, use localized names (PT-BR) with wildcards
		if pdhNetRx == 0 || pdhNetTx == 0 {
			pdhNetRxList = addLocalizedCounters(expandWild, addCounter, `\\Interface de rede(*)\\Bytes recebidos/s`)
			pdhNetTxList = addLocalizedCounters(expandWild, addCounter, `\\Interface de rede(*)\\Bytes enviados/s`)
		}
		if pdhDiskRead == 0 || pdhDiskWrite == 0 {
			pdhDiskReadList = addLocalizedCounters(expandWild, addCounter, `\\PhysicalDisk(*)\\Bytes de leitura de disco/s`)
			pdhDiskWriteList = addLocalizedCounters(expandWild, addCounter, `\\PhysicalDisk(*)\\Bytes de gravação de disco/s`)
		}

		// first collect
		r2, _, err2 := collect.Call(uintptr(pdhQuery))
		if r2 != 0 {
			pdhInitErr = err2
			return
		}
	})
}

func pdhGet(counter syscall.Handle) (float64, error) {
	pdh := syscall.NewLazyDLL("pdh.dll")
	getFmt := pdh.NewProc("PdhGetFormattedCounterValue")

	var val pdhCounterValue
	r, _, err := getFmt.Call(
		uintptr(counter),
		uintptr(pdhFmtDouble),
		0,
		uintptr(unsafe.Pointer(&val)),
	)
	if r != 0 {
		return 0, err
	}
	return val.DoubleValue, nil
}

func collectNetDiskBps() (float64, float64, float64, float64) {
	pdhInit()
	if pdhInitErr != nil {
		return 0, 0, 0, 0
	}

	pdh := syscall.NewLazyDLL("pdh.dll")
	collect := pdh.NewProc("PdhCollectQueryData")
	collect.Call(uintptr(pdhQuery))

	rx := sumCounters(pdhNetRx, pdhNetRxList)
	tx := sumCounters(pdhNetTx, pdhNetTxList)
	dr := sumCounters(pdhDiskRead, pdhDiskReadList)
	dw := sumCounters(pdhDiskWrite, pdhDiskWriteList)
	return rx, tx, dr, dw
}

func addLocalizedCounters(expandWild, addCounter *syscall.LazyProc, wildcardPath string) []syscall.Handle {
	p, _ := syscall.UTF16PtrFromString(wildcardPath)

	// first call to get buffer size
	var bufSize uint32
	r1, _, _ := expandWild.Call(0, uintptr(unsafe.Pointer(p)), 0, 0, uintptr(unsafe.Pointer(&bufSize)))
	_ = r1
	if bufSize == 0 {
		return nil
	}

	buf := make([]uint16, bufSize)
	r2, _, _ := expandWild.Call(0, uintptr(unsafe.Pointer(p)), 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bufSize)))
	if r2 != 0 {
		return nil
	}

	paths := splitMultiSz(buf)
	var handles []syscall.Handle
	for _, path := range paths {
		if path == "" {
			continue
		}
		pth, _ := syscall.UTF16PtrFromString(path)
		var h syscall.Handle
		r, _, _ := addCounter.Call(uintptr(pdhQuery), uintptr(unsafe.Pointer(pth)), 0, uintptr(unsafe.Pointer(&h)))
		if r == 0 && h != 0 {
			handles = append(handles, h)
		}
	}
	return handles
}

func splitMultiSz(buf []uint16) []string {
	var out []string
	start := 0
	for i := 0; i < len(buf); i++ {
		if buf[i] == 0 {
			if i > start {
				out = append(out, syscall.UTF16ToString(buf[start:i]))
			} else {
				break
			}
			start = i + 1
		}
	}
	return out
}

func sumCounters(single syscall.Handle, list []syscall.Handle) float64 {
	if single != 0 {
		v, _ := pdhGet(single)
		return v
	}
	var sum float64
	for _, h := range list {
		if h == 0 {
			continue
		}
		v, _ := pdhGet(h)
		sum += v
	}
	if math.IsNaN(sum) {
		return 0
	}
	return sum
}
