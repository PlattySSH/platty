package open

import (
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

func File(path string) error {
	verb, err := windows.UTF16PtrFromString("open")

	if err != nil {
		return err
	}

	file, err := windows.UTF16PtrFromString(path)

	if err != nil {
		return err
	}

	return windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL)
}

var runnable = map[string]bool{
	".exe": true, ".com": true, ".bat": true, ".cmd": true, ".scr": true, ".pif": true, ".cpl": true, ".msc": true,
	".msi": true, ".msp": true, ".appx": true, ".msix": true, ".appref-ms": true, ".application": true,
	".lnk": true, ".url": true, ".scf": true, ".library-ms": true, ".search-ms": true, ".settingcontent-ms": true,
	".js": true, ".jse": true, ".vbs": true, ".vbe": true, ".wsf": true, ".wsh": true, ".ws": true, ".hta": true,
	".ps1": true, ".psm1": true, ".jar": true, ".reg": true, ".inf": true, ".chm": true, ".diagcab": true,
	".gadget": true, ".xll": true, ".iso": true, ".img": true, ".vhd": true, ".vhdx": true,
}

func Edit(path string) error {
	if runnable[strings.ToLower(filepath.Ext(path))] {
		return exec.Command("notepad.exe", path).Start()
	}

	return File(path)
}
