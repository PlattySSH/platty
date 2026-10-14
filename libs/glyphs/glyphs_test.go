package glyphs_test

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"testing"

	"github.com/plattyssh/glyphs"
)

func TestRenderSample(t *testing.T) {
	set, err := glyphs.NewSet(os.DirFS("testdata"), [4]string{"JetBrainsMono-Regular.ttf", "JetBrainsMono-Bold.ttf", "JetBrainsMono-Regular.ttf", "JetBrainsMono-Bold.ttf"})

	if err != nil {
		t.Fatal(err)
	}

	c := glyphs.NewCache(set, 28, 1.2)
	m := c.Metrics()

	if m.CellW <= 0 || m.CellH <= 0 {
		t.Fatalf("metrics: %+v", m)
	}

	type cell struct {
		r     rune
		style glyphs.Style
		cells int
	}
	var line []cell

	for _, r := range "Ab " {
		line = append(line, cell{r, glyphs.Regular, 1})
	}

	for _, r := range "Bold " {
		line = append(line, cell{r, glyphs.Bold, 1})
	}

	for _, r := range "✔⚠→★ " {
		line = append(line, cell{r, glyphs.Regular, 1})
	}

	for _, r := range "漢字🚀👍🐹" {
		line = append(line, cell{r, glyphs.Regular, 2})
	}

	w := 0

	for _, cl := range line {
		w += cl.cells * m.CellW
	}

	img := image.NewRGBA(image.Rect(0, 0, w+20, m.CellH+10))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{30, 31, 34, 255}), image.Point{}, draw.Src)

	x := 10
	empty := 0

	for _, cl := range line {
		g := c.Get(cl.r, cl.style, cl.cells)

		if g.Empty() && cl.r != ' ' {
			empty++
		}

		org := image.Pt(x, 5)

		if g.Mask != nil {
			r := g.Mask.Bounds().Add(org.Add(g.Off))
			draw.DrawMask(img, r, image.NewUniform(color.RGBA{223, 225, 229, 255}), image.Point{}, g.Mask, image.Point{}, draw.Over)
		}

		if g.Color != nil {
			r := g.Color.Bounds().Add(org.Add(g.Off))
			draw.Draw(img, r, g.Color, image.Point{}, draw.Over)
		}

		x += cl.cells * m.CellW
	}

	if empty > 0 {
		t.Errorf("%d empty glyphs", empty)
	}

	if p := os.Getenv("PLATTY_GLYPH_PNG"); p != "" {
		f, _ := os.Create(p)
		png.Encode(f, img)
		f.Close()
	}
}

func TestLigatures(t *testing.T) {
	set, err := glyphs.NewSet(os.DirFS("testdata"), [4]string{"JetBrainsMono-Regular.ttf", "JetBrainsMono-Bold.ttf", "JetBrainsMono-Regular.ttf", "JetBrainsMono-Bold.ttf"})

	if err != nil {
		t.Fatal(err)
	}

	c := glyphs.NewCache(set, 28, 1.2)

	for _, s := range []string{"->", "!=", "=>", "<=>"} {
		gids := c.Ligatures([]rune(s), glyphs.Regular)

		if gids == nil {
			t.Errorf("%q has no ligature", s)
			continue
		}

		drawn := false

		for _, g := range gids {
			drawn = drawn || g != 0 && !c.GetGlyph(g, glyphs.Regular).Empty()
		}

		if !drawn {
			t.Errorf("%q: the ligature draws nothing", s)
		}
	}

	if gids := c.Ligatures([]rune("ab"), glyphs.Regular); gids != nil {
		t.Errorf("plain letters were changed: %v", gids)
	}
}
