//go:build !windows && !linux && !darwin

package pty

func Start(opts Options) (PTY, error) {
	return nil, ErrUnsupported
}
