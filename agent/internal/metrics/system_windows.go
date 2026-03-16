//go:build windows

package metrics

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

type SystemInfo struct {
	UptimeSec    float64
	Load1        float64
	Load5        float64
	Load15       float64
	ProcCount    float64
	ThreadCount  float64
	KThreadCount float64
	RunningProcs float64
	// Windows: load/kthread/running ficam 0
}

// Toolhelp snapshot structs
type processEntry32 struct {
	Size            uint32
	CntUsage        uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	CntThreads      uint32
	ParentProcessID uint32
	PcPriClassBase  int32
	Flags           uint32
	ExeFile         [windows.MAX_PATH]uint16
}

type threadEntry32 struct {
	Size           uint32
	CntUsage       uint32
	ThreadID       uint32
	OwnerProcessID uint32
	BasePri        int32
	DeltaPri       int32
	Flags          uint32
}

var (
	kernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	procCreateToolhelp32Snap = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW      = kernel32.NewProc("Process32FirstW")
	procProcess32NextW       = kernel32.NewProc("Process32NextW")
	procThread32First        = kernel32.NewProc("Thread32First")
	procThread32Next         = kernel32.NewProc("Thread32Next")

	// ✅ Uptime robusto: chamar GetTickCount64 direto do kernel32
	procGetTickCount64 = kernel32.NewProc("GetTickCount64")
)

const (
	TH32CS_SNAPPROCESS = 0x00000002
	TH32CS_SNAPTHREAD  = 0x00000004
)

func createSnapshot(flags uint32) (windows.Handle, error) {
	r1, _, e1 := procCreateToolhelp32Snap.Call(uintptr(flags), uintptr(0))
	h := windows.Handle(r1)
	if h == windows.InvalidHandle {
		if e1 != nil {
			return 0, e1
		}
		return 0, windows.GetLastError()
	}
	return h, nil
}

func getUptimeSec() float64 {
	r1, _, _ := procGetTickCount64.Call()
	// r1 = milliseconds since boot
	return float64(uint64(r1)) / 1000.0
}

func collectSystemInfo() (SystemInfo, bool) {
	var info SystemInfo

	// ✅ Uptime (sem depender de windows.GetTickCount64 do x/sys)
	info.UptimeSec = getUptimeSec()

	// Process count
	{
		h, err := createSnapshot(TH32CS_SNAPPROCESS)
		if err == nil {
			defer windows.CloseHandle(h)
			var pe processEntry32
			pe.Size = uint32(unsafe.Sizeof(pe))
			r1, _, _ := procProcess32FirstW.Call(uintptr(h), uintptr(unsafe.Pointer(&pe)))
			if r1 != 0 {
				count := 0
				for {
					count++
					r2, _, _ := procProcess32NextW.Call(uintptr(h), uintptr(unsafe.Pointer(&pe)))
					if r2 == 0 {
						break
					}
				}
				info.ProcCount = float64(count)
			}
		}
	}

	// Thread count
	{
		h, err := createSnapshot(TH32CS_SNAPTHREAD)
		if err == nil {
			defer windows.CloseHandle(h)
			var te threadEntry32
			te.Size = uint32(unsafe.Sizeof(te))
			r1, _, _ := procThread32First.Call(uintptr(h), uintptr(unsafe.Pointer(&te)))
			if r1 != 0 {
				count := 0
				for {
					count++
					r2, _, _ := procThread32Next.Call(uintptr(h), uintptr(unsafe.Pointer(&te)))
					if r2 == 0 {
						break
					}
				}
				info.ThreadCount = float64(count)
			}
		}
	}

	return info, true
}

// Export para o main.go
func GetSystemInfo() (SystemInfo, bool) {
	return collectSystemInfo()
}
