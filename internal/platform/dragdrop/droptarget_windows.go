package dragdrop

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	cfHDrop         = 15
	dvAspectContent = 1
	tymedHGlobal    = 1
	dropEffectNone  = 0
)

var (
	procRegisterDragDrop = ole32.NewProc("RegisterDragDrop")
	procReleaseStgMedium = ole32.NewProc("ReleaseStgMedium")

	iidDropTarget = windows.GUID{Data1: 0x00000122, Data4: [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}
)

type formatEtc struct {
	format uint16
	ptd    uintptr
	aspect uint32
	index  int32
	tymed  uint32
}

type stgMedium struct {
	tymed   uint32
	hGlobal uintptr
	release uintptr
}

type dropTargetVtbl struct {
	queryInterface uintptr
	addRef         uintptr
	release        uintptr
	dragEnter      uintptr
	dragOver       uintptr
	dragLeave      uintptr
	drop           uintptr
}

type dropTargetObj struct {
	vtbl *dropTargetVtbl
}

var (
	hoverTarget func(over bool)
	accepting   bool

	targetVtbl = &dropTargetVtbl{
		queryInterface: syscall.NewCallback(func(this uintptr, riid *windows.GUID, ppv *uintptr) uintptr {
			if *riid != iidUnknown && *riid != iidDropTarget {
				*ppv = 0
				return eNoInterface
			}

			*ppv = this
			return sOK
		}),
		addRef:  syscall.NewCallback(func(this uintptr) uintptr { return 1 }),
		release: syscall.NewCallback(func(this uintptr) uintptr { return 1 }),
		dragEnter: syscall.NewCallback(func(this uintptr, data unsafe.Pointer, keys, pt uintptr, effect *uint32) uintptr {
			accepting = hasFiles(data)
			setEffect(effect)

			if accepting && hoverTarget != nil {
				hoverTarget(true)
			}

			return sOK
		}),
		dragOver: syscall.NewCallback(func(this, keys, pt uintptr, effect *uint32) uintptr {
			setEffect(effect)
			return sOK
		}),
		dragLeave: syscall.NewCallback(func(this uintptr) uintptr {
			leave()
			return sOK
		}),
		drop: syscall.NewCallback(func(this uintptr, data unsafe.Pointer, keys, pt uintptr, effect *uint32) uintptr {
			setEffect(effect)
			leave()
			paths := filesOf(data)

			if dropTarget != nil && len(paths) > 0 {
				go dropTarget(paths)
			}

			return sOK
		}),
	}

	target = &dropTargetObj{vtbl: targetVtbl}
)

func setEffect(effect *uint32) {
	*effect = dropEffectNone

	if accepting {
		*effect = dropEffectCopy
	}
}

func leave() {
	if accepting && hoverTarget != nil {
		hoverTarget(false)
	}

	accepting = false
}

func hdropFormat() formatEtc {
	return formatEtc{format: cfHDrop, aspect: dvAspectContent, index: -1, tymed: tymedHGlobal}
}

func hasFiles(data unsafe.Pointer) bool {
	vtbl := *(**[16]uintptr)(data)
	f := hdropFormat()
	hr, _, _ := syscall.SyscallN(vtbl[5], uintptr(data), uintptr(unsafe.Pointer(&f)))
	return hr == sOK
}

func filesOf(data unsafe.Pointer) []string {
	vtbl := *(**[16]uintptr)(data)
	f := hdropFormat()
	var m stgMedium
	hr, _, _ := syscall.SyscallN(vtbl[3], uintptr(data), uintptr(unsafe.Pointer(&f)), uintptr(unsafe.Pointer(&m)))

	if hr != sOK {
		return nil
	}

	defer procReleaseStgMedium.Call(uintptr(unsafe.Pointer(&m)))
	return queryFiles(m.hGlobal)
}

func registerTarget(hwnd uintptr) bool {
	procOleInitialize.Call(0)
	hr, _, _ := procRegisterDragDrop.Call(hwnd, uintptr(unsafe.Pointer(target)))
	return hr == sOK
}
