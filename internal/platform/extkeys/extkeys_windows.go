package extkeys

import (
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

const (
	wmKeyDown    = 0x0100
	wmSysKeyDown = 0x0104
	vkF13        = 0x7C
	vkF24        = 0x87
	keysSubclass = 0x504c4b59
)

var (
	user32          = windows.NewLazySystemDLL("user32.dll")
	comctl32        = windows.NewLazySystemDLL("comctl32.dll")
	procGetKeyState = user32.NewProc("GetKeyState")
	procSetSubclass = comctl32.NewProc("SetWindowSubclass")
	procDefSubclass = comctl32.NewProc("DefSubclassProc")

	keysCallback = syscall.NewCallback(keysProc)
	onKey        func(name string)
)

func Enable(hwnd uintptr, fn func(name string)) {
	if hwnd == 0 || onKey != nil {
		return
	}

	onKey = fn
	procSetSubclass.Call(hwnd, keysCallback, keysSubclass, 0)
}

func down(vk uintptr) bool {
	r, _, _ := procGetKeyState.Call(vk)
	return int16(r) < 0
}

func keysProc(hwnd, msg, wParam, lParam, id, ref uintptr) uintptr {
	if (msg != wmKeyDown && msg != wmSysKeyDown) || wParam < vkF13 || wParam > vkF24 {
		r, _, _ := procDefSubclass.Call(hwnd, msg, wParam, lParam)
		return r
	}

	var parts []string

	if down(0x11) {
		parts = append(parts, "Ctrl")
	}

	if down(0x12) {
		parts = append(parts, "Alt")
	}

	if down(0x10) {
		parts = append(parts, "Shift")
	}

	if down(0x5B) || down(0x5C) {
		parts = append(parts, "Super")
	}

	name := strings.Join(append(parts, "F"+strconv.Itoa(int(wParam-vkF13)+13)), "+")

	if onKey != nil {
		onKey(name)
	}

	return 0
}
