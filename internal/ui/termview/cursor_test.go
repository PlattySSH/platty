package termview

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/plattyssh/vt"

	"github.com/plattyssh/platty/internal/fonts"
	"github.com/plattyssh/platty/internal/ui/kit"
	"github.com/plattyssh/platty/internal/ui/theme"
)

func TestCursorExpandBlink(t *testing.T) {
	shaper, _ := fonts.Load()
	set, _ := fonts.Glyphs("")
	th := theme.New(theme.Carbon)
	term := vt.New(20, 3)
	v := New(&staticBackend{term: term}, kit.New(th, shaper), set, Settings{CursorBlink: true})

	size := image.Pt(240, 80)
	win, err := headless.NewWindow(size.X, size.Y)

	if err != nil {
		t.Skip("no headless GPU:", err)
	}

	defer win.Release()

	height := func(name string, p time.Duration) int {
		gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Exact(size), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}, Now: time.Now()}
		v.Layout(gtx)
		v.focused = true
		v.lastInput = time.Now()
		v.blinkStart = time.Now().Add(-p)
		gtx.Ops.Reset()
		v.Layout(gtx)

		if err := win.Frame(gtx.Ops); err != nil {
			t.Fatal(err)
		}

		img := image.NewRGBA(image.Rectangle{Max: size})

		if err := win.Screenshot(img); err != nil {
			t.Fatal(err)
		}

		if dir := os.Getenv("PLATTY_SHOTS"); dir != "" {
			f, _ := os.Create(filepath.Join(dir, "cursor-"+name+".png"))
			png.Encode(f, img)
			f.Close()
		}

		x := v.gridOff.X + v.m.w/2
		c := th.Cursor
		n := 0

		for y := 0; y < size.Y; y++ {
			px := img.RGBAAt(x, y)

			if px.R == c.R && px.G == c.G && px.B == c.B {
				n++
			}
		}

		return n
	}

	full := height("full", 0)
	half := height("half", blinkVisible+blinkFadeOut/2)
	hidden := height("hidden", blinkVisible+blinkFadeOut+blinkHidden/2)
	t.Logf("height: full %d, halfway %d, hidden %d (cell %d)", full, half, hidden, v.m.h)

	if full < v.m.h-1 {
		t.Errorf("the full cursor is %d tall, the cell %d", full, v.m.h)
	}

	if half <= 0 || half >= full*3/4 {
		t.Errorf("halfway through the blink it's %d tall (full %d)", half, full)
	}

	if hidden != 0 {
		t.Errorf("hidden but still %d tall", hidden)
	}
}
