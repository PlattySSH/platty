package keyring

import "errors"

var (
	ErrUnsupported = errors.New("keyring: no system keyring is available")
	ErrNotFound    = errors.New("keyring: secret not found")
	ErrDismissed   = errors.New("keyring: the request was dismissed")
)
