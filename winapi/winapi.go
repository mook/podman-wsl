package winapi

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type DWORD uint32

type KNOWNFOLDERID syscall.GUID

var (
	FOLDERID_System         = KNOWNFOLDERID{0x1AC14E77, 0x02E7, 0x4E5D, [8]byte{0xB7, 0x44, 0x2E, 0xB1, 0xAE, 0x51, 0x98, 0xB7}}
	FOLDERID_RoamingAppData = KNOWNFOLDERID{0x3EB685DB, 0x65F9, 0x4CF6, [8]byte{0xA0, 0x3A, 0xE3, 0xEF, 0x65, 0x72, 0x9F, 0x3D}}
)

const (
	KF_FLAG_DEFAULT                          = 0
	KF_FLAG_FORCE_APP_DATA_REDIRECTION       = 0x00080000
	KF_FLAG_RETURN_FILTER_REDIRECTION_TARGET = 0x00040000
	KF_FLAG_FORCE_PACKAGE_REDIRECTION        = 0x00020000
	KF_FLAG_NO_PACKAGE_REDIRECTION           = 0x00010000
	KF_FLAG_FORCE_APPCONTAINER_REDIRECTION   = 0x00020000 // Deprecated
	KF_FLAG_NO_APPCONTAINER_REDIRECTION      = 0x00010000 // Deprecated
	KF_FLAG_CREATE                           = 0x00008000
)

type dllInfo struct {
	dll   *windows.LazyDLL
	procs map[string]*windows.LazyProc
}

var procs struct {
	sync.Mutex
	fn map[string]dllInfo
}

func getProc(module, name string) (*windows.LazyProc, error) {
	procs.Lock()
	defer procs.Unlock()

	if procs.fn == nil {
		procs.fn = make(map[string]dllInfo)
	}
	dll, ok := procs.fn[module]
	if !ok {
		dll = dllInfo{
			dll:   windows.NewLazySystemDLL(module),
			procs: make(map[string]*windows.LazyProc),
		}
		procs.fn[module] = dll
		if err := dll.dll.Load(); err != nil {
			return nil, err
		}
	}
	fn, ok := dll.procs[name]
	if !ok {
		fn = dll.dll.NewProc(name)
		dll.procs[name] = fn
		if err := fn.Find(); err != nil {
			return nil, err
		}
	}
	return fn, nil
}

type HResult uint32

func (hr HResult) Ok() bool {
	return hr&0x8000_0000 == 0
}

func (hr HResult) Error() string {
	if err := hr.Unwrap(); err != nil {
		return err.Error()
	}
	return fmt.Sprintf("%08x", uint32(hr))
}

func (hr HResult) Unwrap() error {
	if hr.Ok() {
		return nil
	}
	if hr&0xFFFF_0000 == 0x8007_0000 {
		return syscall.Errno(hr & 0xFFFF)
	}
	return nil
}

func SHGetKnownFolderPath(rfid KNOWNFOLDERID, flags DWORD) (string, error) {
	SHGetKnownFolderPath, err := getProc("shell32.dll", "SHGetKnownFolderPath")
	if err != nil {
		return "", err
	}
	var path *uint16
	hr_, _, _ := SHGetKnownFolderPath.Call(
		uintptr(unsafe.Pointer(&rfid)),
		uintptr(flags),
		uintptr(unsafe.Pointer(nil)),
		uintptr(unsafe.Pointer(&path)),
	)
	if path != nil {
		windows.CoTaskMemFree(unsafe.Pointer(path))
	}
	hr := HResult(hr_)
	if !hr.Ok() {
		return "", hr
	}
	return windows.UTF16PtrToString(path), nil
}

// GetStdHandle returns a handle to one of stdin/stdout/stderr, which can be
// inherited by a child process.  The caller is responsible for closing the
// result.
func GetStdHandle(file *os.File) (syscall.Handle, error) {
	var stdhandle int
	switch file {
	case os.Stdin:
		stdhandle = syscall.STD_INPUT_HANDLE
	case os.Stdout:
		stdhandle = syscall.STD_OUTPUT_HANDLE
	case os.Stderr:
		stdhandle = syscall.STD_ERROR_HANDLE
	default:
		return syscall.InvalidHandle, fmt.Errorf("invalid input")
	}
	oldHandle, err := syscall.GetStdHandle(stdhandle)
	if err != nil {
		return syscall.InvalidHandle, err
	}
	hProc, err := syscall.GetCurrentProcess()
	if err != nil {
		return syscall.InvalidHandle, err
	}

	var target syscall.Handle
	err = syscall.DuplicateHandle(hProc, oldHandle, hProc, &target, 0, true, syscall.DUPLICATE_SAME_ACCESS)
	if err != nil {
		return syscall.InvalidHandle, err
	}
	return target, nil
}
