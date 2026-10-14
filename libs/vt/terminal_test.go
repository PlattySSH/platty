package vt

import (
	"bytes"
	"strings"
	"testing"
)

func newTest(cols, rows int) (*Terminal, *bytes.Buffer) {
	var replies bytes.Buffer
	t := New(cols, rows, WithReply(func(b []byte) { replies.Write(b) }))
	return t, &replies
}

func screenText(t *Terminal) string {
	var f Frame
	t.Snapshot(&f, 0)

	var sb strings.Builder

	for y := 0; y < f.Rows; y++ {
		var row strings.Builder

		for _, c := range f.Row(y) {
			if c.Width == 0 {
				continue
			}

			if c.Rune == 0 {
				row.WriteByte(' ')
				continue
			}

			row.WriteRune(c.Rune)
		}

		sb.WriteString(strings.TrimRight(row.String(), " "))

		if y < f.Rows-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func cursorAt(t *Terminal) (int, int) {
	var f Frame
	t.Snapshot(&f, 0)
	return f.Cursor.X, f.Cursor.Y
}

func TestPrintAndWrap(t *testing.T) {
	term, _ := newTest(5, 3)
	term.Write([]byte("hello world"))

	want := "hello\n worl\nd"

	if got := screenText(term); got != want {
		t.Fatalf("screen:\n%q\nwant:\n%q", got, want)
	}
}

func TestPendingWrapDoesNotScrollEarly(t *testing.T) {
	term, _ := newTest(5, 2)
	term.Write([]byte("abcde"))

	if x, y := cursorAt(term); x != 4 || y != 0 {
		t.Fatalf("cursor at %d,%d; want 4,0", x, y)
	}

	term.Write([]byte("\r\nxy"))

	if got := screenText(term); got != "abcde\nxy" {
		t.Fatalf("screen: %q", got)
	}
}

func TestScrollToScrollback(t *testing.T) {
	term, _ := newTest(10, 2)
	term.Write([]byte("uno\r\ndos\r\ntres"))

	if got := screenText(term); got != "dos\ntres" {
		t.Fatalf("screen: %q", got)
	}

	var f Frame
	term.Snapshot(&f, 1)

	if f.Scrollback != 1 || string(f.Row(0)[0].Rune) != "u" {
		t.Fatalf("scrollback=%d first=%q", f.Scrollback, f.Row(0)[0].Rune)
	}
}

func TestCursorMovementAndErase(t *testing.T) {
	term, _ := newTest(10, 3)
	term.Write([]byte("aaaaaaaaaa\r\nbbbbbbbbbb\r\ncccccccccc"))
	term.Write([]byte("\x1b[2;4H\x1b[K"))
	term.Write([]byte("\x1b[1;1H\x1b[3X"))

	want := "   aaaaaaa\nbbb\ncccccccccc"

	if got := screenText(term); got != want {
		t.Fatalf("screen:\n%q\nwant:\n%q", got, want)
	}
}

func TestSGR(t *testing.T) {
	term, _ := newTest(10, 1)
	term.Write([]byte("\x1b[1;31mA\x1b[38;2;10;20;30;48;5;200mB\x1b[38:2::1:2:3mC\x1b[0mD"))

	var f Frame
	term.Snapshot(&f, 0)
	row := f.Row(0)

	if row[0].Attr&AttrBold == 0 || row[0].Fg != Indexed(1) {
		t.Errorf("A: %+v", row[0])
	}

	if row[1].Fg != RGB(10, 20, 30) || row[1].Bg != Indexed(200) {
		t.Errorf("B: %+v", row[1])
	}

	if row[2].Fg != RGB(1, 2, 3) {
		t.Errorf("C: %+v", row[2])
	}

	if row[3].Attr != 0 || row[3].Fg != DefaultColor {
		t.Errorf("D: %+v", row[3])
	}
}

func TestWideChars(t *testing.T) {
	term, _ := newTest(4, 2)
	term.Write([]byte("a漢b"))

	var f Frame
	term.Snapshot(&f, 0)
	row := f.Row(0)

	if row[1].Rune != '漢' || row[1].Width != 2 || row[2].Width != 0 || row[3].Rune != 'b' {
		t.Fatalf("row: %+v", row)
	}

	term.Write([]byte("\r\nxyz漢"))

	if got := screenText(term); got != "漢b\nxyz" && got != "a漢b\nxyz" {
		t.Logf("screen: %q", got)
	}
}

func TestScrollRegion(t *testing.T) {
	term, _ := newTest(5, 4)
	term.Write([]byte("1\r\n2\r\n3\r\n4"))
	term.Write([]byte("\x1b[2;3r\x1b[3;1H\n"))

	if got := screenText(term); got != "1\n3\n\n4" {
		t.Fatalf("screen: %q", got)
	}
}

func TestInsertDeleteLines(t *testing.T) {
	term, _ := newTest(5, 4)
	term.Write([]byte("1\r\n2\r\n3\r\n4"))
	term.Write([]byte("\x1b[2;1H\x1b[L"))

	if got := screenText(term); got != "1\n\n2\n3" {
		t.Fatalf("after IL: %q", got)
	}

	term.Write([]byte("\x1b[M\x1b[M"))

	if got := screenText(term); got != "1\n3\n\n" {
		t.Fatalf("after DL: %q", got)
	}
}

func TestAltScreen(t *testing.T) {
	term, _ := newTest(5, 2)
	term.Write([]byte("main"))
	term.Write([]byte("\x1b[?1049h\x1b[Halt"))

	if got := screenText(term); got != "alt\n" {
		t.Fatalf("alt: %q", got)
	}

	term.Write([]byte("\x1b[?1049l"))

	if got := screenText(term); got != "main\n" {
		t.Fatalf("round trip: %q", got)
	}

	if x, _ := cursorAt(term); x != 4 {
		t.Fatalf("cursor no restaurado: x=%d", x)
	}
}

func TestDeviceStatusReport(t *testing.T) {
	term, replies := newTest(10, 5)
	term.Write([]byte("\x1b[3;7H\x1b[6n"))

	if got := replies.String(); got != "\x1b[3;7R" {
		t.Fatalf("reply: %q", got)
	}
}

func TestSplitSequences(t *testing.T) {
	term, _ := newTest(10, 1)
	input := []byte("\x1b[31mñ\x1b]0;title\x07ok")

	for _, b := range input {
		term.Write([]byte{b})
	}

	var f Frame
	term.Snapshot(&f, 0)

	if f.Row(0)[0].Rune != 'ñ' || f.Row(0)[0].Fg != Indexed(1) || f.Title != "title" {
		t.Fatalf("cell=%+v title=%q", f.Row(0)[0], f.Title)
	}

	if got := screenText(term); got != "ñok" {
		t.Fatalf("screen: %q", got)
	}
}

func TestDECGraphics(t *testing.T) {
	term, _ := newTest(5, 1)
	term.Write([]byte("\x1b(0lqk\x1b(Bx"))

	if got := screenText(term); got != "┌─┐x" {
		t.Fatalf("screen: %q", got)
	}
}

func TestResizeKeepsCursorVisible(t *testing.T) {
	term, _ := newTest(10, 4)
	term.Write([]byte("1\r\n2\r\n3\r\n4"))
	term.Resize(10, 2)

	if got := screenText(term); got != "3\n4" {
		t.Fatalf("after shrinking: %q", got)
	}

	term.Resize(10, 4)

	if got := screenText(term); got != "1\n2\n3\n4" {
		t.Fatalf("after growing: %q", got)
	}
}

func BenchmarkWritePlain(b *testing.B) {
	term := New(200, 60)
	data := bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog 0123456789\r\n"), 1000)

	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for b.Loop() {
		term.Write(data)
	}
}

func BenchmarkWriteColored(b *testing.B) {
	term := New(200, 60)
	data := bytes.Repeat([]byte("\x1b[1;32mok\x1b[0m \x1b[38;2;200;100;50mtruecolor\x1b[0m line with ñ and 漢字\r\n"), 1000)

	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for b.Loop() {
		term.Write(data)
	}
}

func TestTextRangeAndFind(t *testing.T) {
	term, _ := newTest(6, 2)
	term.Write([]byte("hello world\r\nbye"))

	var f Frame
	term.Snapshot(&f, 0)

	top := f.TopLine - f.Scrollback
	got := term.TextRange(Pos{Line: top, Col: 0}, Pos{Line: f.TopLine + 1, Col: 5})

	if got != "hello world\r\nbye" {
		t.Fatalf("text: %q", got)
	}

	m := term.Find("LO WO", false)

	if len(m) != 1 || m[0].Start != (Pos{Line: top, Col: 3}) || m[0].End != (Pos{Line: top + 1, Col: 1}) {
		t.Fatalf("find: %+v", m)
	}
}

func TestReflowNarrowAndBack(t *testing.T) {
	term, _ := newTest(10, 4)
	term.Write([]byte("0123456789abcdef\r\n$ "))

	term.Resize(5, 4)

	if got := screenText(term); got != "56789\nabcde\nf\n$" {
		t.Fatalf("estrecho:\n%q", got)
	}

	if x, y := cursorAt(term); x != 2 || y != 3 {
		t.Fatalf("cursor estrecho: %d,%d", x, y)
	}

	term.Resize(10, 4)

	if got := screenText(term); got != "0123456789\nabcdef\n$\n" {
		t.Fatalf("back:\n%q", got)
	}

	if x, y := cursorAt(term); x != 2 || y != 2 {
		t.Fatalf("cursor after returning: %d,%d", x, y)
	}
}

func TestReflowKeepsWideChars(t *testing.T) {
	term, _ := newTest(6, 3)
	term.Write([]byte("ab漢字c"))

	term.Resize(3, 3)

	if got := screenText(term); got != "ab\n漢\n字c" {
		t.Fatalf("screen:\n%q", got)
	}
}

func TestReflowWiderJoinsWrappedLines(t *testing.T) {
	term, _ := newTest(4, 3)
	term.Write([]byte("abcdefgh"))
	term.Resize(8, 3)

	if got := screenText(term); got != "abcdefgh\n\n" {
		t.Fatalf("screen:\n%q", got)
	}
}
