package pty

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

type conPTY struct {
	hpc     windows.Handle
	process windows.Handle
	in      *os.File
	out     *os.File

	closeOnce sync.Once
}

func Start(opts Options) (PTY, error) {
	cols, rows := max(opts.Cols, 1), max(opts.Rows, 1)

	cmdline, err := commandLine(opts.Command)

	if err != nil {
		return nil, err
	}

	var inR, inW, outR, outW windows.Handle

	if err := windows.CreatePipe(&inR, &inW, nil, 0); err != nil {
		return nil, fmt.Errorf("pty: input pipe: %w", err)
	}

	if err := windows.CreatePipe(&outR, &outW, nil, 0); err != nil {
		windows.CloseHandle(inR)
		windows.CloseHandle(inW)
		return nil, fmt.Errorf("pty: output pipe: %w", err)
	}

	var hpc windows.Handle
	size := windows.Coord{X: int16(cols), Y: int16(rows)}

	err = windows.CreatePseudoConsole(size, inR, outW, 0, &hpc)

	windows.CloseHandle(inR)
	windows.CloseHandle(outW)

	if err != nil {
		windows.CloseHandle(inW)
		windows.CloseHandle(outR)
		return nil, fmt.Errorf("pty: CreatePseudoConsole: %w", err)
	}

	process, err := spawn(hpc, cmdline, opts)

	if err != nil {
		windows.ClosePseudoConsole(hpc)
		windows.CloseHandle(inW)
		windows.CloseHandle(outR)
		return nil, err
	}

	return &conPTY{
		hpc:     hpc,
		process: process,
		in:      os.NewFile(uintptr(inW), "conpty-in"),
		out:     os.NewFile(uintptr(outR), "conpty-out"),
	}, nil
}

func spawn(hpc windows.Handle, cmdline string, opts Options) (windows.Handle, error) {
	attrs, err := windows.NewProcThreadAttributeList(1)

	if err != nil {
		return 0, fmt.Errorf("pty: attribute list: %w", err)
	}

	defer attrs.Delete()

	err = attrs.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, *(*unsafe.Pointer)(unsafe.Pointer(&hpc)), unsafe.Sizeof(hpc))

	if err != nil {
		return 0, fmt.Errorf("pty: attribute pseudoconsole: %w", err)
	}

	si := &windows.StartupInfoEx{ProcThreadAttributeList: attrs.List()}
	si.Cb = uint32(unsafe.Sizeof(*si))
	si.Flags = windows.STARTF_USESTDHANDLES

	cmd16, err := windows.UTF16PtrFromString(cmdline)

	if err != nil {
		return 0, err
	}

	var dir16 *uint16

	if opts.Dir != "" {
		dir16, err = windows.UTF16PtrFromString(opts.Dir)

		if err != nil {
			return 0, err
		}
	}

	env := envBlock(append(os.Environ(), opts.Env...))
	flags := uint32(windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_UNICODE_ENVIRONMENT)

	var pi windows.ProcessInformation
	err = windows.CreateProcess(nil, cmd16, nil, nil, false, flags, &env[0], dir16, &si.StartupInfo, &pi)

	if err != nil {
		return 0, fmt.Errorf("pty: CreateProcess %q: %w", cmdline, err)
	}

	windows.CloseHandle(pi.Thread)
	return pi.Process, nil
}

func (c *conPTY) Read(p []byte) (int, error) {
	return c.out.Read(p)
}

func (c *conPTY) Write(p []byte) (int, error) {
	return c.in.Write(p)
}

func (c *conPTY) Resize(cols, rows int) error {
	return windows.ResizePseudoConsole(c.hpc, windows.Coord{X: int16(max(cols, 1)), Y: int16(max(rows, 1))})
}

func (c *conPTY) Wait() (int, error) {
	_, err := windows.WaitForSingleObject(c.process, windows.INFINITE)

	if err != nil {
		return -1, err
	}

	var code uint32

	if err := windows.GetExitCodeProcess(c.process, &code); err != nil {
		return -1, err
	}

	return int(code), nil
}

func (c *conPTY) Close() error {
	c.closeOnce.Do(func() {
		c.in.Close()
		windows.ClosePseudoConsole(c.hpc)
		c.out.Close()
		windows.CloseHandle(c.process)
	})
	return nil
}

func commandLine(command []string) (string, error) {
	if len(command) == 0 {
		command = []string{defaultShell()}
	}

	exe, err := exec.LookPath(command[0])

	if err != nil {
		return "", fmt.Errorf("pty: %w", err)
	}

	args := make([]string, len(command))
	args[0] = windows.EscapeArg(exe)

	for i, a := range command[1:] {
		args[i+1] = windows.EscapeArg(a)
	}

	return strings.Join(args, " "), nil
}

func defaultShell() string {
	for _, sh := range []string{"pwsh.exe", "powershell.exe"} {
		if _, err := exec.LookPath(sh); err == nil {
			return sh
		}
	}

	if comspec := os.Getenv("COMSPEC"); comspec != "" {
		return comspec
	}

	return "cmd.exe"
}

func envBlock(env []string) []uint16 {
	seen := make(map[string]int, len(env))
	var kept []string

	for _, kv := range env {
		k, _, _ := strings.Cut(kv, "=")
		key := strings.ToUpper(k)

		if i, ok := seen[key]; ok {
			kept[i] = kv
			continue
		}

		seen[key] = len(kept)
		kept = append(kept, kv)
	}

	var block []uint16

	for _, kv := range kept {
		u, err := windows.UTF16FromString(kv)

		if err != nil {
			continue
		}

		block = append(block, u...)
	}

	return append(block, 0)
}
