package app

import (
	"testing"

	"github.com/plattyssh/platty/internal/app/onboarding"
	"github.com/plattyssh/platty/internal/ui/theme"
)

func TestOnboarding(t *testing.T) {
	h := newHarness(t)
	intro := onboarding.New(h.a.App)
	h.a.Open(intro)
	h.frames(50)
	h.shot("intro-0")

	intro.ClickNext()
	h.frames(7)
	h.shot("intro-jump")
	h.frames(10)
	intro.ClickTheme(3)
	h.frames(44)
	h.shot("intro-1")

	if h.a.settings.Theme != theme.Names[3] {
		t.Errorf("picking a theme didn't apply it (theme %q)", h.a.settings.Theme)
	}

	intro.ClickTheme(0)
	h.frames(6)

	for step := 2; step <= 3; step++ {
		intro.ClickNext()
		h.frames(50)
		h.shot("intro-" + string(rune('0'+step)))
	}

	intro.SetPasswords("long enough password", "something else")
	intro.ClickNext()
	h.frames(4)

	if intro.Step() != 3 || intro.PasswordError() == "" {
		t.Fatalf("mismatched passwords moved on (step %d)", intro.Step())
	}

	intro.SetPasswords("", "")
	intro.ClickNext()
	h.frames(50)
	h.shot("intro-4")

	if h.a.vault.HasPassword() {
		t.Error("an empty password was set as master password")
	}

	intro.ClickNext()
	h.frames(60)
	h.shot("intro-5")
	intro.ClickNext()
	h.frames(4)

	if h.a.modal != nil || !h.a.settings.Onboarded {
		t.Error("finishing didn't close the intro or mark it as done")
	}
}
