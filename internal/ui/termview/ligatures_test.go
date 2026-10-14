package termview

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
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

func TestLigatureRendering(t *testing.T) {
	shaper, _ := fonts.Load()
	set, _ := fonts.Glyphs("")
	size := image.Pt(420, 60)
	win, err := headless.NewWindow(size.X, size.Y)

	if err != nil {
		t.Skip("no headless GPU:", err)
	}

	defer win.Release()

	render := func(ligs bool) *image.RGBA {
		term := vt.New(30, 2)
		v := New(&staticBackend{term: term}, kit.New(theme.New(theme.Carbon), shaper), set, Settings{TextSize: 16, Ligatures: ligs})
		gtx := layout.Context{Ops: new(op.Ops), Constraints: layout.Exact(size), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}}
		v.Layout(gtx)
		term.Write([]byte("a -> b != c => d <= e"))
		gtx.Ops.Reset()
		v.Layout(gtx)
		win.Frame(gtx.Ops)
		img := image.NewRGBA(image.Rectangle{Max: size})
		win.Screenshot(img)

		if dir := os.Getenv("PLATTY_SHOTS"); dir != "" {
			name := "ligatures-off.png"

			if ligs {
				name = "ligatures-on.png"
			}

			f, _ := os.Create(filepath.Join(dir, name))
			png.Encode(f, img)
			f.Close()
		}

		return img
	}

	if bytes.Equal(render(false).Pix, render(true).Pix) {
		t.Error("ligatures didn't change how the text is drawn")
	}
}
