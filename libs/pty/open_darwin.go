package pty

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func open() (*os.File, *os.File, error) {
	fd, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)

	if err != nil {
		return nil, nil, fmt.Errorf("pty: open /dev/ptmx: %w", err)
	}

	master := os.NewFile(uintptr(fd), "/dev/ptmx")

	if err := unix.IoctlSetInt(fd, unix.TIOCPTYGRANT, 0); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: grant: %w", err)
	}

	if err := unix.IoctlSetInt(fd, unix.TIOCPTYUNLK, 0); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: unlock: %w", err)
	}

	buf := make([]byte, 128)

	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(unix.TIOCPTYGNAME), uintptr(unsafe.Pointer(&buf[0]))); errno != 0 {
		master.Close()
		return nil, nil, fmt.Errorf("pty: ptsname: %w", errno)
	}

	name := unix.ByteSliceToString(buf)
	slave, err := os.OpenFile(name, os.O_RDWR|unix.O_NOCTTY, 0)

	if err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: open %s: %w", name, err)
	}

	return master, slave, nil
}
