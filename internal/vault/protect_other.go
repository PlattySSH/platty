//go:build !windows

package vault

func protect(plain []byte) ([]byte, error) {
	return plain, nil
}

func unprotect(enc []byte) ([]byte, error) {
	return enc, nil
}
