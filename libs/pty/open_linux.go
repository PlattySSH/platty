package pty

import (
	"fmt"
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

func open() (*os.File, *os.File, error) {
	fd, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)

	if err != nil {
		return nil, nil, fmt.Errorf("pty: open /dev/ptmx: %w", err)
	}

	master := os.NewFile(uintptr(fd), "/dev/ptmx")

	if err := unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0); err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: unlock: %w", err)
	}

	n, err := unix.IoctlGetUint32(fd, unix.TIOCGPTN)

	if err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: ptsname: %w", err)
	}

	name := "/dev/pts/" + strconv.FormatUint(uint64(n), 10)
	slave, err := os.OpenFile(name, os.O_RDWR|unix.O_NOCTTY, 0)

	if err != nil {
		master.Close()
		return nil, nil, fmt.Errorf("pty: open %s: %w", name, err)
	}

	return master, slave, nil
}
