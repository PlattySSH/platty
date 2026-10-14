package secure

import (
	"bytes"
	"testing"
)

func TestSealOpen(t *testing.T) {
	key := NewKey()
	sealed, err := Seal(key, []byte("secret"), []byte("aad"))

	if err != nil {
		t.Fatal(err)
	}

	plain, err := Open(key, sealed, []byte("aad"))

	if err != nil || string(plain) != "secret" {
		t.Fatalf("open: %q %v", plain, err)
	}

	if _, err := Open(key, sealed, []byte("other")); err != ErrDecrypt {
		t.Errorf("different aad: %v", err)
	}

	if _, err := Open(NewKey(), sealed, []byte("aad")); err != ErrDecrypt {
		t.Errorf("different key: %v", err)
	}

	sealed[len(sealed)-1] ^= 1

	if _, err := Open(key, sealed, []byte("aad")); err != ErrDecrypt {
		t.Errorf("tampered data: %v", err)
	}
}

func TestKDF(t *testing.T) {
	k := NewKDF(1, 8*1024)
	a, err := k.Key("correct horse")

	if err != nil {
		t.Fatal(err)
	}

	b, _ := k.Key("correct horse")
	c, _ := k.Key("wrong horse")

	if !bytes.Equal(a, b) || bytes.Equal(a, c) || len(a) != KeySize {
		t.Error("the derived keys don't behave")
	}

	k.Memory = 1 << 30

	if _, err := k.Key("x"); err != ErrBadParams {
		t.Errorf("huge memory accepted: %v", err)
	}
}
