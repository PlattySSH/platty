package vault

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateV1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.dat")
	plain, _ := json.Marshal(Data{Version: 1, Hosts: []Host{{ID: "a", Address: "10.0.0.1", Password: "hunter2"}}})
	old, err := protect(plain)

	if err != nil {
		t.Fatal(err)
	}

	os.WriteFile(path, old, 0o600)
	v, err := OpenAt(path)

	if err != nil {
		t.Fatal(err)
	}

	if h, ok := v.Host("a"); !ok || h.Password != "hunter2" {
		t.Fatalf("migrated host: %+v %v", h, ok)
	}

	raw, _ := os.ReadFile(path)

	if !bytes.Contains(raw, []byte(`"format":"platty-vault"`)) || bytes.Contains(raw, []byte("hunter2")) {
		t.Fatal("the vault wasn't rewritten in the new format")
	}

	again, err := OpenAt(path)

	if err != nil || len(again.Snapshot().Hosts) != 1 {
		t.Fatalf("reopen after migration: %v", err)
	}
}

func TestMasterPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.dat")
	v, _ := OpenAt(path)
	v.SaveHost(Host{Address: "10.0.0.2", Password: "s3cret"})

	if err := v.SetOSUnlock(false); err != ErrNoPassword {
		t.Errorf("turning off OS unlock without a password: %v", err)
	}

	if err := v.SetPassword("", "correct horse battery"); err != nil {
		t.Fatal(err)
	}

	if err := v.SetOSUnlock(false); err != nil {
		t.Fatal(err)
	}

	v, err := OpenAt(path)

	if err != nil {
		t.Fatal(err)
	}

	if !v.Locked() || len(v.Snapshot().Hosts) != 0 {
		t.Fatal("without OS unlock the vault should open locked and empty")
	}

	if _, err := v.SaveHost(Host{Address: "x"}); err != ErrLocked {
		t.Errorf("saving while locked: %v", err)
	}

	if err := v.Unlock("wrong horse"); err != ErrWrongPassword {
		t.Errorf("wrong password: %v", err)
	}

	if err := v.Unlock("correct horse battery"); err != nil || len(v.Snapshot().Hosts) != 1 {
		t.Fatalf("unlock: %v", err)
	}

	if err := v.SetPassword("nope", "new one"); err != ErrWrongPassword {
		t.Errorf("change with a wrong current password: %v", err)
	}

	if err := v.SetPassword("correct horse battery", "new password here"); err != nil {
		t.Fatal(err)
	}

	v.Lock()

	if len(v.Snapshot().Hosts) != 0 {
		t.Error("Lock left data in memory")
	}

	if err := v.Unlock("new password here"); err != nil {
		t.Fatalf("unlock with the new password: %v", err)
	}

	if err := v.SetPassword("new password here", ""); err != nil {
		t.Fatal(err)
	}

	if !v.OSUnlock() || v.HasPassword() {
		t.Error("removing the password must turn OS unlock back on")
	}

	if v, _ := OpenAt(path); v.Locked() {
		t.Error("without a password the vault should open by itself")
	}
}
