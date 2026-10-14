//go:build !windows

package conpty

func Start(opts Options) (PTY, error) {
	return nil, ErrUnsupported
}
