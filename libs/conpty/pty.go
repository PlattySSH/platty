package conpty

import (
	"errors"
	"io"
)

var ErrUnsupported = errors.New("conpty: unsupported platform")

type Options struct {
	Command    []string
	Dir        string
	Env        []string
	Cols, Rows int
}

type PTY interface {
	io.ReadWriteCloser
	Resize(cols, rows int) error
	Wait() (int, error)
}
