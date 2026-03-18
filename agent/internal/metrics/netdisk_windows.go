//go:build windows

package metrics

import (
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
)

func pdhInit() {
	pdhOnce.Do(func() {
		pdh := syscall.NewLazyDLL("pdh.dll")
		openQuery := pdh.NewProc("PdhOpenQueryW")
		addEngCounter := pdh.NewProc("PdhAddEnglishCounterW")
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

		// Try _Total instance for aggregated counters
		if err := add(`\\Network Interface(_Total)\\Bytes Received/sec`, &pdhNetRx); err != nil {
			pdhInitErr = err
			return
		}
		if err := add(`\\Network Interface(_Total)\\Bytes Sent/sec`, &pdhNetTx); err != nil {
			pdhInitErr = err
			return
		}
		if err := add(`\\PhysicalDisk(_Total)\\Disk Read Bytes/sec`, &pdhDiskRead); err != nil {
			pdhInitErr = err
			return
		}
		if err := add(`\\PhysicalDisk(_Total)\\Disk Write Bytes/sec`, &pdhDiskWrite); err != nil {
			pdhInitErr = err
			return
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

	rx, _ := pdhGet(pdhNetRx)
	tx, _ := pdhGet(pdhNetTx)
	dr, _ := pdhGet(pdhDiskRead)
	dw, _ := pdhGet(pdhDiskWrite)
	return rx, tx, dr, dw
}

