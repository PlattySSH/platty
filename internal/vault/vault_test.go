package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTripEncrypted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.dat")
	v, err := OpenAt(path)

	if err != nil {
		t.Fatal(err)
	}

	h, err := v.SaveHost(Host{Label: "web", Address: "10.0.0.1", Username: "root", Auth: AuthPassword, Password: "s3cret-pass"})

	if err != nil {
		t.Fatal(err)
	}

	raw, _ := os.ReadFile(path)

	if bytes.Contains(raw, []byte("s3cret-pass")) {
		t.Fatal("the password is stored in plain text in the file")
	}

	v2, err := OpenAt(path)

	if err != nil {
		t.Fatal(err)
	}

	got, ok := v2.Host(h.ID)

	if !ok || got.Password != "s3cret-pass" || got.Port != 22 {
		t.Fatalf("host: %+v ok=%v", got, ok)
	}
}

func TestDeleteHostCascades(t *testing.T) {
	v, _ := OpenAt(filepath.Join(t.TempDir(), "vault.dat"))
	jump, _ := v.SaveHost(Host{Address: "bastion"})
	h, _ := v.SaveHost(Host{Address: "db", JumpID: jump.ID})
	v.SaveForward(Forward{HostID: jump.ID, Kind: ForwardLocal, BindPort: 5432})

	v.DeleteHost(jump.ID)

	d := v.Snapshot()

	if len(d.Forwards) != 0 {
		t.Fatal("the deleted host's rule is still there")
	}

	got, _ := v.Host(h.ID)

	if got.JumpID != "" {
		t.Fatal("the deleted jump host is still referenced")
	}
}

func TestTarget(t *testing.T) {
	h := Host{Address: "example.com", Username: "ana", Port: 2222}

	if got := h.Target(); got != "ana@example.com:2222" {
		t.Fatal(got)
	}
}
