package input

import (
	"testing"

	"github.com/plattyssh/vt"
)

func TestEncodeKey(t *testing.T) {
	cases := []struct {
		key  Key
		mod  Mod
		app  bool
		want string
	}{
		{KeyUp, 0, false, "\x1b[A"},
		{KeyUp, 0, true, "\x1bOA"},
		{KeyLeft, ModCtrl, false, "\x1b[1;5D"},
		{KeyDelete, 0, false, "\x1b[3~"},
		{KeyPageUp, ModShift, false, "\x1b[5;2~"},
		{KeyF1, 0, false, "\x1bOP"},
		{KeyF5, 0, false, "\x1b[15~"},
		{KeyTab, ModShift, false, "\x1b[Z"},
		{KeyBackspace, 0, false, "\x7f"},
	}

	for _, c := range cases {
		got := string(EncodeKey(c.key, c.mod, vt.Modes{CursorKeysApp: c.app}))

		if got != c.want {
			t.Errorf("key %d mod %d: %q, want %q", c.key, c.mod, got, c.want)
		}
	}

	if got := string(EncodeRune('c', ModCtrl)); got != "\x03" {
		t.Errorf("ctrl+c: %q", got)
	}
}

func TestEncodeMouseSGR(t *testing.T) {
	md := vt.Modes{Mouse: vt.MouseClick, MouseSGR: true}

	if got := string(EncodeMouse(MouseLeft, MousePress, 4, 9, 0, md)); got != "\x1b[<0;5;10M" {
		t.Fatalf("press: %q", got)
	}

	if got := string(EncodeMouse(MouseLeft, MouseUp, 4, 9, 0, md)); got != "\x1b[<0;5;10m" {
		t.Fatalf("release: %q", got)
	}

	if EncodeMouse(MouseLeft, MouseMove, 1, 1, 0, md) != nil {
		t.Fatal("click mode must not report motion")
	}
}
