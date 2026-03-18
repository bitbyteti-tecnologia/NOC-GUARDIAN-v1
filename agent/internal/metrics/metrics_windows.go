//go:build windows

package metrics

import (
	"errors"
	"syscall"
	"time"
	"unsafe"
)

type Snapshot struct {
	TS          time.Time
	CPUPercent  float64
	MemUsedPct  float64
	DiskUsedPct float64
	DiskPath    string

	MemTotalBytes float64
	MemUsedBytes  float64

	Services map[string]string

	HasSys bool
	System SystemInfo
}

type memStatusEx struct {
	cbSize                  uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

func getMemStatusEx() (memStatusEx, error) {
	// GlobalMemoryStatusEx
	k32 := syscall.NewLazyDLL("kernel32.dll")
	proc := k32.NewProc("GlobalMemoryStatusEx")

	var st memStatusEx
	st.cbSize = uint32(unsafe.Sizeof(st))

	r1, _, err := proc.Call(uintptr(unsafe.Pointer(&st)))
	if r1 == 0 {
		return memStatusEx{}, err
	}
	return st, nil
}

func memUsedPercent() (float64, error) {
	st, err := getMemStatusEx()
	if err != nil {
		return 0, err
	}
	// dwMemoryLoad já é percent usado (0-100)
	used := float64(st.dwMemoryLoad)
	if used < 0 {
		used = 0
	}
	if used > 100 {
		used = 100
	}
	return used, nil
}

func diskUsedPercent(path string) (float64, error) {
	if path == "" {
		path = `C:\`
	}

	k32 := syscall.NewLazyDLL("kernel32.dll")
	proc := k32.NewProc("GetDiskFreeSpaceExW")

	var freeBytesAvail uint64
	var totalBytes uint64
	var totalFreeBytes uint64

	// UTF-16 pointer
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	r1, _, err := proc.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&freeBytesAvail)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if r1 == 0 {
		return 0, err
	}
	if totalBytes == 0 {
		return 0, errors.New("disk total invalid")
	}

	used := (float64(totalBytes-totalFreeBytes) / float64(totalBytes)) * 100.0
	if used < 0 {
		used = 0
	}
	if used > 100 {
		used = 100
	}
	return used, nil
}

func Collect(diskPath string) (Snapshot, error) {
	memPct, totalBytes, usedBytes, err := memInfo()
	if err != nil {
		return Snapshot{}, err
	}
	if diskPath == "" {
		diskPath = `C:\`
	}
	disk, err := diskUsedPercent(diskPath)
	if err != nil {
		return Snapshot{}, err
	}

	sys, hasSys := collectSystemInfo()

	return Snapshot{
		TS:            time.Now().UTC(),
		CPUPercent:    0, // MVP: implementar depois (PDH/WMI)
		MemUsedPct:    memPct,
		MemTotalBytes: totalBytes,
		MemUsedBytes:  usedBytes,
		DiskUsedPct:   disk,
		DiskPath:      diskPath,
		Services:      map[string]string{},
		HasSys:        hasSys,
		System:        sys,
	}, nil
}

func memInfo() (pct float64, total float64, used float64, err error) {
	st, err := getMemStatusEx()
	if err != nil {
		return 0, 0, 0, err
	}
	total = float64(st.ullTotalPhys)
	free := float64(st.ullAvailPhys)
	used = total - free
	pct = float64(st.dwMemoryLoad)
	return pct, total, used, nil
}
