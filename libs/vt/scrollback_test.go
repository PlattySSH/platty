package vt

import (
	"fmt"
	"strings"
	"testing"
)

func TestScrollbackStoresCompactLines(t *testing.T) {
	term, _ := newTest(120, 4)

	for i := range 50 {
		fmt.Fprintf(term, "linea %d\r\n", i)
	}

	if n := term.sb.len(); n < 40 {
		t.Fatalf("scrollback has %d lines, want at least 40", n)
	}

	for i := 0; i < term.sb.len(); i++ {
		if c := cap(term.sb.at(i).cells); c > 12 {
			t.Fatalf("scrollback line %d keeps %d cells; it should be trimmed", i, c)
		}
	}

	var f Frame
	term.Snapshot(&f, term.sb.len())
	row := f.Row(0)

	if len(row) != 120 {
		t.Fatalf("the row has %d cells, want 120", len(row))
	}

	var b strings.Builder

	for _, c := range row {
		if c.Rune != 0 {
			b.WriteRune(c.Rune)
		}
	}

	if got := b.String(); got != "linea 0" {
		t.Fatalf("first scrollback line = %q", got)
	}
}
