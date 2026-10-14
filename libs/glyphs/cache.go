package glyphs

import (
	"image"
	"image/color"
	"math"
	"sync"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/vector"
)

type Glyph struct {
	Mask  *image.Alpha
	Color *image.RGBA
	Off   image.Point
}

func (g *Glyph) Empty() bool {
	return g == nil || (g.Mask == nil && g.Color == nil)
}

type Metrics struct {
	CellW, CellH int
	Baseline     int
}

type glyphKey struct {
	r     rune
	style Style
	cells uint8
}

type Cache struct {
	set  *Set
	px   float32
	line float32
	m    Metrics

	mu     sync.Mutex
	glyphs map[glyphKey]*Glyph
	byGID  map[gidKey]*Glyph

	ligMu  sync.Mutex
	ligs   map[ligKey][]font.GID
	shaper shaping.HarfbuzzShaper
}

func NewCache(set *Set, px, lineHeight float32) *Cache {
	c := &Cache{set: set, px: px, line: lineHeight, glyphs: map[glyphKey]*Glyph{}, byGID: map[gidKey]*Glyph{}, ligs: map[ligKey][]font.GID{}}

	primary := set.styles[Regular]
	scale := px / primary.upem
	ext, _ := primary.FontHExtents()
	asc := float32(ext.Ascender) * scale
	desc := -float32(ext.Descender) * scale

	gid, _ := primary.NominalGlyph('M')
	adv := primary.HorizontalAdvance(gid) * scale

	glyphH := asc + desc
	h := int(math.Round(float64(glyphH * lineHeight)))
	c.m = Metrics{
		CellW:    max(int(math.Round(float64(adv))), 1),
		CellH:    max(h, 1),
		Baseline: int(math.Round(float64((float32(h)-glyphH)/2 + asc))),
	}
	return c
}

func (c *Cache) Metrics() Metrics {
	return c.m
}

func (c *Cache) Get(r rune, style Style, cells int) *Glyph {
	k := glyphKey{r, style, uint8(cells)}

	c.mu.Lock()
	g, ok := c.glyphs[k]
	c.mu.Unlock()

	if ok {
		return g
	}

	g = c.render(r, style, cells)

	c.mu.Lock()
	c.glyphs[k] = g
	c.mu.Unlock()
	return g
}

func (c *Cache) render(r rune, style Style, cells int) *Glyph {
	f, gid, _ := c.set.find(r, style, cells == 2 && r >= 0x2190)
	return c.renderFace(f, gid, style, cells)
}

func (c *Cache) renderFace(f *face, gid font.GID, style Style, cells int) *Glyph {
	span := float32(cells * c.m.CellW)

	if paint, ok := f.GlyphDataColor(gid); ok && f.CPAL != nil {
		return c.renderColor(f, gid, paint.Paint, span)
	}

	outline, ok := f.GlyphDataOutline(gid)

	if !ok || len(outline.Segments) == 0 {
		return &Glyph{}
	}

	scale := c.px / f.upem
	adv := f.HorizontalAdvance(gid) * scale

	if adv > span*1.05 && adv > 0 {
		scale *= span / adv
		adv = span
	}

	m := identity()
	m = m.translate((span-adv)/2, float32(c.m.Baseline)).mul(affine{a: scale, d: -scale})

	mask, off := rasterize([]ot.Segment(outline.Segments), m)

	if mask == nil {
		return &Glyph{}
	}

	if (style == Bold || style == BoldItalic) && f != c.set.styles[Bold] && f != c.set.styles[BoldItalic] {
		embolden(mask)
	}

	return &Glyph{Mask: mask, Off: off}
}

func rasterize(segs []ot.Segment, m affine) (*image.Alpha, image.Point) {
	minX, minY := float32(math.MaxFloat32), float32(math.MaxFloat32)
	maxX, maxY := float32(-math.MaxFloat32), float32(-math.MaxFloat32)

	for _, s := range segs {
		for _, p := range s.ArgsSlice() {
			x, y := m.apply(p.X, p.Y)
			minX, minY = min(minX, x), min(minY, y)
			maxX, maxY = max(maxX, x), max(maxY, y)
		}
	}

	if minX > maxX {
		return nil, image.Point{}
	}

	x0, y0 := int(math.Floor(float64(minX)))-1, int(math.Floor(float64(minY)))-1
	x1, y1 := int(math.Ceil(float64(maxX)))+1, int(math.Ceil(float64(maxY)))+1
	w, h := x1-x0, y1-y0

	if w <= 0 || h <= 0 || w > 4096 || h > 4096 {
		return nil, image.Point{}
	}

	z := vector.NewRasterizer(w, h)
	fx, fy := float32(x0), float32(y0)
	pt := func(p ot.SegmentPoint) (float32, float32) {
		x, y := m.apply(p.X, p.Y)
		return x - fx, y - fy
	}

	for _, s := range segs {
		switch s.Op {
		case ot.SegmentOpMoveTo:
			z.ClosePath()
			z.MoveTo(pt(s.Args[0]))
		case ot.SegmentOpLineTo:
			z.LineTo(pt(s.Args[0]))
		case ot.SegmentOpQuadTo:
			ax, ay := pt(s.Args[0])
			bx, by := pt(s.Args[1])
			z.QuadTo(ax, ay, bx, by)
		case ot.SegmentOpCubeTo:
			ax, ay := pt(s.Args[0])
			bx, by := pt(s.Args[1])
			cx, cy := pt(s.Args[2])
			z.CubeTo(ax, ay, bx, by, cx, cy)
		}
	}

	z.ClosePath()

	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	z.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	return mask, image.Pt(x0, y0)
}

func embolden(m *image.Alpha) {
	w := m.Rect.Dx()

	for y := 0; y < m.Rect.Dy(); y++ {
		row := m.Pix[y*m.Stride : y*m.Stride+w]

		for x := w - 1; x > 0; x-- {
			row[x] = max(row[x], row[x-1])
		}
	}
}

type affine struct {
	a, b, c, d, e, f float32
}

func identity() affine {
	return affine{a: 1, d: 1}
}

func (m affine) apply(x, y float32) (float32, float32) {
	return m.a*x + m.c*y + m.e, m.b*x + m.d*y + m.f
}

func (m affine) mul(n affine) affine {
	return affine{
		a: m.a*n.a + m.c*n.b,
		b: m.b*n.a + m.d*n.b,
		c: m.a*n.c + m.c*n.d,
		d: m.b*n.c + m.d*n.d,
		e: m.a*n.e + m.c*n.f + m.e,
		f: m.b*n.e + m.d*n.f + m.f,
	}
}

func (m affine) translate(x, y float32) affine {
	return m.mul(affine{a: 1, d: 1, e: x, f: y})
}

func (m affine) invert() affine {
	det := m.a*m.d - m.b*m.c

	if det == 0 {
		return identity()
	}

	inv := 1 / det
	return affine{
		a: m.d * inv,
		b: -m.b * inv,
		c: -m.c * inv,
		d: m.a * inv,
		e: (m.c*m.f - m.d*m.e) * inv,
		f: (m.b*m.e - m.a*m.f) * inv,
	}
}

func rgba(r, g, b, a uint8) color.NRGBA {
	return color.NRGBA{R: r, G: g, B: b, A: a}
}
