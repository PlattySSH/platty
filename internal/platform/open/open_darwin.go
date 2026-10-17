package open

import (
	"os/exec"
	"path/filepath"
	"strings"
)

func File(path string) error {
	return exec.Command("open", path).Start()
}

var runnable = map[string]bool{
	".app": true, ".command": true, ".tool": true, ".terminal": true, ".pkg": true, ".mpkg": true, ".dmg": true,
	".sh": true, ".bash": true, ".zsh": true, ".fish": true, ".py": true, ".pl": true, ".rb": true, ".jar": true,
	".scpt": true, ".scptd": true, ".applescript": true, ".workflow": true, ".webloc": true, ".inetloc": true,
}

func Edit(path string) error {
	if runnable[strings.ToLower(filepath.Ext(path))] {
		return exec.Command("open", "-t", path).Start()
	}

	return File(path)
}
