//go:build !windows

package open

import "os/exec"

func File(path string) error {
	return exec.Command("xdg-open", path).Start()
}
