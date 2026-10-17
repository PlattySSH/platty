//go:build !windows && !darwin

package open

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
)

var errNoEditor = errors.New("no default text editor is set")

func File(path string) error {
	return exec.Command("xdg-open", path).Start()
}

var runnable = map[string]bool{
	".desktop": true, ".sh": true, ".bash": true, ".zsh": true, ".fish": true, ".run": true, ".bin": true,
	".appimage": true, ".flatpakref": true, ".flatpak": true, ".deb": true, ".rpm": true, ".snap": true,
	".py": true, ".pl": true, ".rb": true, ".jar": true, ".exe": true, ".msi": true, ".bat": true, ".cmd": true,
}

func Edit(path string) error {
	if !runnable[strings.ToLower(filepath.Ext(path))] {
		return File(path)
	}

	out, err := exec.Command("xdg-mime", "query", "default", "text/plain").Output()
	app := strings.TrimSpace(string(out))

	if err != nil || app == "" {
		return errNoEditor
	}

	return exec.Command("gtk-launch", app, path).Start()
}
