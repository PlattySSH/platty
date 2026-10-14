package dragdrop

import (
	"errors"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	dropEffectCopy        = 1
	mkLButton             = 0x0001
	sOK                   = 0
	eNoInterface          = 0x80004002
	dragDropSDrop         = 0x00040100
	dragDropSCancel       = 0x00040101
	dragDropSUseDefCursor = 0x00040102
)

var (
	ole32                  = windows.NewLazySystemDLL("ole32.dll")
	procOleInitialize      = ole32.NewProc("OleInitialize")
	procDoDragDrop         = ole32.NewProc("DoDragDrop")
	procCoTaskMemFree      = ole32.NewProc("CoTaskMemFree")
	procSHParseDisplayName = shell32.NewProc("SHParseDisplayName")
	procSHCreateDataObject = shell32.NewProc("SHCreateDataObject")
	procILFindLastID       = shell32.NewProc("ILFindLastID")

	iidUnknown    = windows.GUID{Data1: 0x00000000, Data4: [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidDataObject = windows.GUID{Data1: 0x0000010e, Data4: [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidDropSource = windows.GUID{Data1: 0x00000121, Data4: [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}

	errDragFailed = errors.New("dragdrop: the shell refused the drag")
)

type dropSourceVtbl struct {
	queryInterface    uintptr
	addRef            uintptr
	release           uintptr
	queryContinueDrag uintptr
	giveFeedback      uintptr
}

type dropSource struct {
	vtbl *dropSourceVtbl
}

var (
	sourceVtbl = &dropSourceVtbl{
		queryInterface: syscall.NewCallback(func(this uintptr, riid *windows.GUID, ppv *uintptr) uintptr {
			if *riid != iidUnknown && *riid != iidDropSource {
				*ppv = 0
				return eNoInterface
			}

			*ppv = this
			return sOK
		}),
		addRef:  syscall.NewCallback(func(this uintptr) uintptr { return 1 }),
		release: syscall.NewCallback(func(this uintptr) uintptr { return 1 }),
		queryContinueDrag: syscall.NewCallback(func(this, escape, keys uintptr) uintptr {
			switch {
			case int32(escape) != 0:
				return dragDropSCancel
			case keys&mkLButton == 0:
				return dragDropSDrop
			}

			return sOK
		}),
		giveFeedback: syscall.NewCallback(func(this, effect uintptr) uintptr { return dragDropSUseDefCursor }),
	}

	source = &dropSource{vtbl: sourceVtbl}
)

func pidl(path string) (uintptr, error) {
	p, err := windows.UTF16PtrFromString(path)

	if err != nil {
		return 0, err
	}

	var id uintptr
	hr, _, _ := procSHParseDisplayName.Call(uintptr(unsafe.Pointer(p)), 0, uintptr(unsafe.Pointer(&id)), 0, 0)

	if hr != sOK {
		return 0, windows.Errno(hr)
	}

	return id, nil
}

func DragOut(paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	procOleInitialize.Call(0)

	folder, err := pidl(filepath.Dir(paths[0]))

	if err != nil {
		return err
	}

	defer procCoTaskMemFree.Call(folder)

	children := make([]uintptr, 0, len(paths))

	for _, p := range paths {
		abs, err := pidl(p)

		if err != nil {
			return err
		}

		defer procCoTaskMemFree.Call(abs)

		last, _, _ := procILFindLastID.Call(abs)
		children = append(children, last)
	}

	var data unsafe.Pointer
	hr, _, _ := procSHCreateDataObject.Call(folder, uintptr(len(children)), uintptr(unsafe.Pointer(&children[0])), 0, uintptr(unsafe.Pointer(&iidDataObject)), uintptr(unsafe.Pointer(&data)))

	if hr != sOK || data == nil {
		return errDragFailed
	}

	defer func() {
		vtbl := *(**[3]uintptr)(data)
		syscall.SyscallN(vtbl[2], uintptr(data))
	}()

	var effect uint32
	hr, _, _ = procDoDragDrop.Call(uintptr(data), uintptr(unsafe.Pointer(source)), dropEffectCopy, uintptr(unsafe.Pointer(&effect)))

	if hr != dragDropSDrop && hr != dragDropSCancel {
		return windows.Errno(hr)
	}

	return nil
}
