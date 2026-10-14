//go:build !windows

package dragdrop

import "errors"

func Enable(hwnd uintptr, onDrop func(paths []string)) {}

func DragOut(paths []string) error {
	return errors.ErrUnsupported
}
