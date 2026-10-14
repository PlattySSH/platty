package vt

import "testing"

func TestOSCLinkCwdClipboard(t *testing.T) {
	var copied string
	term := New(40, 3, WithClipboard(func(s string) { copied = s }))
	term.Write([]byte("go \x1b]8;;https://example.com\x1b\x5chere\x1b]8;;\x1b\x5c now"))
	term.Write([]byte("\x1b]7;file://box/home/deploy/app\x07"))
	term.Write([]byte("\x1b]52;c;aGVsbG8=\x07"))
	term.Write([]byte("\x1b]52;c;?\x07"))

	var f Frame
	term.Snapshot(&f, 0)
	row := f.Row(0)

	if row[3].Link == 0 || row[6].Link != row[3].Link || row[7].Link != 0 || row[0].Link != 0 {
		t.Fatalf("link ids: %d %d %d %d", row[0].Link, row[3].Link, row[6].Link, row[7].Link)
	}

	if got := term.Link(row[3].Link); got != "https://example.com" {
		t.Errorf("link %q", got)
	}

	if host, path := term.Cwd(); host != "box" || path != "/home/deploy/app" {
		t.Errorf("cwd %q %q", host, path)
	}

	if copied != "hello" {
		t.Errorf("clipboard %q", copied)
	}
}
