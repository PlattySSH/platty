//go:build !linux && !(darwin && cgo)

package keyring

func Set(id, label string, secret []byte) error {
	return ErrUnsupported
}

func Get(id string) ([]byte, error) {
	return nil, ErrUnsupported
}

func Delete(id string) error {
	return ErrUnsupported
}
