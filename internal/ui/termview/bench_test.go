package termview

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/plattyssh/vt"

	"github.com/plattyssh/platty/internal/fonts"
	"github.com/plattyssh/platty/internal/ui/kit"
	"github.com/plattyssh/platty/internal/ui/theme"
)

type staticBackend struct {
	term *vt.Terminal
}

func (b *staticBackend) Terminal() *vt.Terminal { return b.term }
func (b *staticBackend) Send([]byte)            {}
func (b *staticBackend) Resize(cols, rows int)  { b.term.Resize(cols, rows) }

func benchFrames(b *testing.B, line func(i int) string) {
	const w, h = 1920, 1080

	win, err := headless.NewWindow(w, h)

	if err != nil {
		b.Skip("no headless GPU:", err)
	}

	defer win.Release()

	term := vt.New(80, 24)
	shaper, _ := fonts.Load()
	set, _ := fonts.Glyphs("")
	view := New(&staticBackend{term: term}, kit.New(theme.New(theme.Carbon), shaper), set, Settings{})

	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(w, h)),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
	view.Layout(gtx)
	cols, rows := term.Size()
	b.Logf("grid %d×%d", cols, rows)

	b.ResetTimer()

	for i := 0; b.Loop(); i++ {
		var sb strings.Builder

		for r := range rows {
			sb.WriteString("\r\n")
			sb.WriteString(line(i*rows + r))
		}

		term.Write([]byte(sb.String()))

		gtx.Ops.Reset()
		view.Layout(gtx)

		if err := win.Frame(gtx.Ops); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFramePlainText(b *testing.B) {
	benchFrames(b, func(i int) string {
		return fmt.Sprintf("%06d The quick brown fox jumps over the lazy dog. Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore", i)
	})
}

func BenchmarkFrameColored(b *testing.B) {
	benchFrames(b, func(i int) string {
		return fmt.Sprintf("\x1b[38;5;%dm%06d\x1b[0m \x1b[1;32mINFO\x1b[0m \x1b[38;2;122;162;255mserver.go:%d\x1b[0m request handled in \x1b[33m%dms\x1b[0m path=/api/v1/users/%d status=\x1b[42;30m 200 \x1b[0m │ ok", i%256, i, i%500, i%97, i)
	})
}

func BenchmarkFrameStatic(b *testing.B) {
	const w, h = 1920, 1080

	win, err := headless.NewWindow(w, h)

	if err != nil {
		b.Skip("no headless GPU:", err)
	}

	defer win.Release()

	term := vt.New(80, 24)
	shaper, _ := fonts.Load()
	set, _ := fonts.Glyphs("")
	view := New(&staticBackend{term: term}, kit.New(theme.New(theme.Carbon), shaper), set, Settings{})

	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(w, h)),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
	}
	view.Layout(gtx)
	_, rows := term.Size()

	for i := range rows {
		term.Write(fmt.Appendf(nil, "\r\nline %d with \x1b[32mcolored\x1b[0m text and more content to fill the screen", i))
	}

	b.ResetTimer()

	for b.Loop() {
		gtx.Ops.Reset()
		view.Layout(gtx)

		if err := win.Frame(gtx.Ops); err != nil {
			b.Fatal(err)
		}
	}
}
