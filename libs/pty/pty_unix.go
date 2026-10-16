//go:build linux || darwin

package pty

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

type unixPTY struct {
	master *os.File
	cmd    *exec.Cmd

	done chan struct{}
	code int
	err  error

	closeOnce sync.Once
}

func Start(opts Options) (PTY, error) {
	cols, rows := max(opts.Cols, 1), max(opts.Rows, 1)

	command := opts.Command

	if len(command) == 0 {
		command = []string{defaultShell()}
	}

	exe, err := exec.LookPath(command[0])

	if err != nil {
		return nil, fmt.Errorf("pty: %w", err)
	}

	master, slave, err := open()

	if err != nil {
		return nil, err
	}

	defer slave.Close()

	if err := setSize(master, cols, rows); err != nil {
		master.Close()
		return nil, err
	}

	cmd := &exec.Cmd{
		Path:   exe,
		Args:   command,
		Dir:    opts.Dir,
		Env:    append(os.Environ(), opts.Env...),
		Stdin:  slave,
		Stdout: slave,
		Stderr: slave,
		SysProcAttr: &syscall.SysProcAttr{
			Setsid:  true,
			Setctty: true,
		},
	}

	if err := cmd.Start(); err != nil {
		master.Close()
		return nil, fmt.Errorf("pty: start %q: %w", exe, err)
	}

	p := &unixPTY{master: master, cmd: cmd, done: make(chan struct{})}

	go p.reap()

	return p, nil
}

func (p *unixPTY) reap() {
	err := p.cmd.Wait()
	p.code = p.cmd.ProcessState.ExitCode()

	var exit *exec.ExitError

	if err != nil && !errors.As(err, &exit) {
		p.err = err
	}

	close(p.done)
}

func (p *unixPTY) Read(b []byte) (int, error) {
	n, err := p.master.Read(b)

	if errors.Is(err, syscall.EIO) {
		err = io.EOF
	}

	return n, err
}

func (p *unixPTY) Write(b []byte) (int, error) {
	return p.master.Write(b)
}

func (p *unixPTY) Resize(cols, rows int) error {
	return setSize(p.master, max(cols, 1), max(rows, 1))
}

func (p *unixPTY) Wait() (int, error) {
	<-p.done
	return p.code, p.err
}

func (p *unixPTY) Close() error {
	p.closeOnce.Do(func() {
		p.master.Close()

		select {
		case <-p.done:
		default:
			syscall.Kill(-p.cmd.Process.Pid, syscall.SIGHUP)
		}
	})
	return nil
}

func setSize(f *os.File, cols, rows int) error {
	ws := &unix.Winsize{Col: uint16(cols), Row: uint16(rows)}

	if err := unix.IoctlSetWinsize(int(f.Fd()), unix.TIOCSWINSZ, ws); err != nil {
		return fmt.Errorf("pty: resize: %w", err)
	}

	return nil
}

func defaultShell() string {
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh
	}

	return "/bin/sh"
}
