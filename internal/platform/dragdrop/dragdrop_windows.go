package dragdrop

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wmDropFiles    = 0x0233
	dropSubclassID = 0x504c4154
	dragQueryCount = 0xffffffff
)

var (
	shell32           = windows.NewLazySystemDLL("shell32.dll")
	comctl32          = windows.NewLazySystemDLL("comctl32.dll")
	procDragAccept    = shell32.NewProc("DragAcceptFiles")
	procDragQueryFile = shell32.NewProc("DragQueryFileW")
	procDragFinish    = shell32.NewProc("DragFinish")
	procSetSubclass   = comctl32.NewProc("SetWindowSubclass")
	procDefSubclass   = comctl32.NewProc("DefSubclassProc")

	dropCallback = syscall.NewCallback(dropProc)
	dropTarget   func(paths []string)
)

func Enable(hwnd uintptr, onDrop func(paths []string), onHover func(over bool)) {
	if hwnd == 0 || dropTarget != nil {
		return
	}

	dropTarget, hoverTarget = onDrop, onHover

	if registerTarget(hwnd) {
		return
	}

	procDragAccept.Call(hwnd, 1)
	procSetSubclass.Call(hwnd, dropCallback, dropSubclassID, 0)
}

func queryFiles(hdrop uintptr) []string {
	n, _, _ := procDragQueryFile.Call(hdrop, dragQueryCount, 0, 0)
	paths := make([]string, 0, n)

	for i := uintptr(0); i < n; i++ {
		size, _, _ := procDragQueryFile.Call(hdrop, i, 0, 0)
		buf := make([]uint16, size+1)
		procDragQueryFile.Call(hdrop, i, uintptr(unsafe.Pointer(&buf[0])), size+1)
		paths = append(paths, windows.UTF16ToString(buf))
	}

	return paths
}

func dropProc(hwnd, msg, wParam, lParam, id, ref uintptr) uintptr {
	if msg != wmDropFiles {
		r, _, _ := procDefSubclass.Call(hwnd, msg, wParam, lParam)
		return r
	}

	paths := queryFiles(wParam)
	procDragFinish.Call(wParam)

	if dropTarget != nil && len(paths) > 0 {
		go dropTarget(paths)
	}

	return 0
}
