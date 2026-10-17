//go:build !(darwin && cgo)

package titlebar

const (
	Native = false
	Width  = 0
)

func Attach(view uintptr, x, center, height float32) {}
