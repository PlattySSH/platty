package app

import (
	"testing"
	"time"

	"gioui.org/io/key"

	"github.com/plattyssh/platty/internal/app/export"
	"github.com/plattyssh/platty/internal/app/settings"
	"github.com/plattyssh/platty/internal/sshx"
)

func waitFor(h *harness, cond func() bool) bool {
	for range 100 {
		h.frames(1)

		if cond() {
			return true
		}

		time.Sleep(20 * time.Millisecond)
	}

	return false
}

func TestLockAndUnlock(t *testing.T) {
	h := newHarness(t)
	h.a.LockVault()

	if h.a.vault.Locked() {
		t.Fatal("locked without a master password")
	}

	if err := h.a.vault.SetPassword("", "correct horse battery"); err != nil {
		t.Fatal(err)
	}

	h.a.LockVault()
	h.frames(4)
	h.shot("locked")

	if !h.a.vault.Locked() || len(h.a.vault.Snapshot().Hosts) != 0 {
		t.Fatal("LockVault didn't lock the vault")
	}

	h.text("wrong password")
	h.key(key.NameReturn, 0)

	if !waitFor(h, func() bool { return h.a.lock.err != "" }) {
		t.Fatal("a wrong password showed no error")
	}

	h.shot("locked-wrong")

	if !h.a.vault.Locked() {
		t.Fatal("a wrong password unlocked the vault")
	}

	h.a.lock.password.SetText("")
	h.text("correct horse battery")
	h.key(key.NameReturn, 0)

	if !waitFor(h, func() bool { return !h.a.vault.Locked() }) {
		t.Fatal("the right password didn't unlock the vault")
	}

	h.frames(4)

	if len(h.a.tools.hosts.Hosts()) == 0 {
		t.Error("the sidebar didn't reload the hosts after unlocking")
	}
}

func TestIdleLock(t *testing.T) {
	h := newHarness(t)
	h.a.vault.SetPassword("", "correct horse battery")
	h.a.settings.LockAfter = 1
	h.a.started = time.Now().Add(-2 * time.Minute)
	h.frames(2)

	if !h.a.vault.Locked() {
		t.Error("the vault didn't lock after being idle")
	}
}

func TestKeyPassphrasePrompt(t *testing.T) {
	h := newHarness(t)
	k, err := sshx.GenerateEd25519("laptop", "secret phrase")

	if err != nil {
		t.Fatal(err)
	}

	k.Passphrase = ""
	saved, _ := h.a.vault.SaveKey(k)
	got := make(chan bool, 1)

	go func() {
		_, ok := h.a.keyPassphrase(saved)
		got <- ok
	}()

	if !waitFor(h, func() bool { return h.a.modal != nil }) {
		t.Fatal("no passphrase prompt")
	}

	h.shot("key-passphrase")
	h.text("secret phrase")
	h.key(key.NameReturn, 0)

	select {
	case ok := <-got:
		if !ok {
			t.Fatal("the right passphrase was rejected")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the prompt never answered")
	}

	if pass, ok := h.a.passes.get(saved.ID); !ok || pass != "secret phrase" {
		t.Error("the passphrase wasn't kept for the session")
	}

	if k, _ := h.a.vault.Key(saved.ID); k.Passphrase != "" {
		t.Error("the passphrase was saved without asking to remember it")
	}
}

func TestSecuritySettingsShots(t *testing.T) {
	h := newHarness(t)
	h.a.vault.SetPassword("", "correct horse battery")

	for page, name := range map[int]string{3: "settings-security", 4: "settings-data", 6: "settings-about"} {
		d := settings.New(h.a.App)
		d.SetPage(page)
		h.a.Open(d)
		h.frames(10)
		h.shot(name)
	}

	h.a.Open(export.New(h.a.App))
	h.frames(10)
	h.shot("export")
}
