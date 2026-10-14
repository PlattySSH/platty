package glyphs

import (
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/font/opentype/tables"
)

const (
	foregroundIndex = 0xffff
	maxPaintDepth   = 32
	emojiScale      = 0.95
)

type colorFn func(x, y float32) color.NRGBA

type painter struct {
	f     *face
	dst   *image.RGBA
	depth int
}

func (c *Cache) renderColor(f *face, gid font.GID, paint tables.PaintTable, span float32) *Glyph {
	ext, _ := f.FontHExtents()
	emH := float32(ext.Ascender - ext.Descender)

	if emH <= 0 {
		emH = f.upem
	}

	cellH := float32(c.m.CellH)
	scale := cellH * emojiScale / emH
	adv := f.HorizontalAdvance(gid) * scale

	if adv > span && adv > 0 {
		scale *= span / adv
		adv = span
	}

	top := (cellH - emH*scale) / 2
	baseline := top + float32(ext.Ascender)*scale

	m := identity().translate((span-adv)/2, baseline).mul(affine{a: scale, d: -scale})

	w := int(math.Ceil(float64(span))) + 2
	h := c.m.CellH + 2
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	p := &painter{f: f, dst: dst}
	p.paint(paint, m.translate(0, 0), rgba(0xff, 0xff, 0xff, 0xff))

	return &Glyph{Color: dst, Off: image.Point{}}
}

func (p *painter) paletteColor(idx uint16, alpha tables.Fixed214, fg color.NRGBA) color.NRGBA {
	c := fg

	if idx != foregroundIndex && len(p.f.CPAL) > 0 && int(idx) < len(p.f.CPAL[0]) {
		rec := p.f.CPAL[0][idx]
		c = rgba(rec.Red, rec.Green, rec.Blue, rec.Alpha)
	}

	c.A = uint8(float32(c.A) * f214(alpha))
	return c
}

func f214(v tables.Fixed214) float32 {
	return float32(v) / (1 << 14)
}

func (p *painter) paint(t tables.PaintTable, m affine, fg color.NRGBA) {
	if p.depth > maxPaintDepth || t == nil {
		return
	}

	p.depth++
	defer func() { p.depth-- }()

	switch v := t.(type) {
	case tables.PaintColrLayersResolved:
		for _, l := range v {
			col := p.paletteColor(l.PaletteIndex, 1<<14, fg)
			p.fillGlyph(font.GID(l.GlyphID), m, func(float32, float32) color.NRGBA { return col })
		}
	case tables.PaintColrLayers:
		layers, err := p.f.COLR.LayerList.Resolve(v)

		if err != nil {
			return
		}

		for _, l := range layers {
			p.paint(l, m, fg)
		}
	case tables.PaintGlyph:
		src := p.source(v.Paint, m, fg)
		p.fillGlyph(font.GID(v.GlyphID), m, src)
	case tables.PaintColrGlyph:
		if sub, ok := p.f.COLR.Search(tables.GlyphID(v.GlyphID)); ok {
			p.paint(sub, m, fg)
		}
	case tables.PaintComposite:
		p.paint(v.BackdropPaint, m, fg)
		p.paint(v.SourcePaint, m, fg)
	default:
		if child, n, ok := transformOf(t); ok {
			p.paint(child, m.mul(n), fg)
		}
	}
}

func (p *painter) source(t tables.PaintTable, m affine, fg color.NRGBA) colorFn {
	solid := func(c color.NRGBA) colorFn {
		return func(float32, float32) color.NRGBA { return c }
	}

	switch v := t.(type) {
	case tables.PaintSolid:
		return solid(p.paletteColor(v.PaletteIndex, v.Alpha, fg))
	case tables.PaintVarSolid:
		return solid(p.paletteColor(v.PaletteIndex, v.Alpha, fg))
	case tables.PaintLinearGradient:
		return p.linear(v.ColorLine, v.X0, v.Y0, v.X1, v.Y1, m, fg)
	case tables.PaintVarLinearGradient:
		return p.linear(varLine(v.ColorLine), v.X0, v.Y0, v.X1, v.Y1, m, fg)
	case tables.PaintRadialGradient:
		return p.radial(v.ColorLine, v.X1, v.Y1, v.Radius1, m, fg)
	case tables.PaintVarRadialGradient:
		return p.radial(varLine(v.ColorLine), v.X1, v.Y1, v.Radius1, m, fg)
	case tables.PaintSweepGradient:
		return p.stopsAt(v.ColorLine, fg, 0.5)
	}

	if child, n, ok := transformOf(t); ok {
		return p.source(child, m.mul(n), fg)
	}

	return solid(fg)
}

func varLine(l tables.VarColorLine) tables.ColorLine {
	out := tables.ColorLine{Extend: l.Extend, ColorStops: make([]tables.ColorStop, len(l.ColorStops))}

	for i, s := range l.ColorStops {
		out.ColorStops[i] = tables.ColorStop{StopOffset: s.StopOffset, PaletteIndex: s.PaletteIndex, Alpha: s.Alpha}
	}

	return out
}

func (p *painter) colorAt(l tables.ColorLine, t float32, fg color.NRGBA) color.NRGBA {
	stops := l.ColorStops

	if len(stops) == 0 {
		return fg
	}

	sort.SliceStable(stops, func(i, j int) bool { return stops[i].StopOffset < stops[j].StopOffset })
	t = min(max(t, 0), 1)

	first := stops[0]

	if t <= f214(first.StopOffset) {
		return p.paletteColor(first.PaletteIndex, first.Alpha, fg)
	}

	for i := 1; i < len(stops); i++ {
		a, b := stops[i-1], stops[i]
		ta, tb := f214(a.StopOffset), f214(b.StopOffset)

		if t > tb {
			continue
		}

		ca := p.paletteColor(a.PaletteIndex, a.Alpha, fg)
		cb := p.paletteColor(b.PaletteIndex, b.Alpha, fg)
		k := float32(0)

		if tb > ta {
			k = (t - ta) / (tb - ta)
		}

		return lerp(ca, cb, k)
	}

	last := stops[len(stops)-1]
	return p.paletteColor(last.PaletteIndex, last.Alpha, fg)
}

func (p *painter) stopsAt(l tables.ColorLine, fg color.NRGBA, t float32) colorFn {
	c := p.colorAt(l, t, fg)
	return func(float32, float32) color.NRGBA { return c }
}

func (p *painter) linear(l tables.ColorLine, x0, y0, x1, y1 int16, m affine, fg color.NRGBA) colorFn {
	inv := m.invert()
	ax, ay := float32(x0), float32(y0)
	dx, dy := float32(x1)-ax, float32(y1)-ay
	den := dx*dx + dy*dy

	if den == 0 {
		return p.stopsAt(l, fg, 0)
	}

	return func(px, py float32) color.NRGBA {
		x, y := inv.apply(px, py)
		t := ((x-ax)*dx + (y-ay)*dy) / den
		return p.colorAt(l, t, fg)
	}
}

func (p *painter) radial(l tables.ColorLine, cx, cy int16, r uint16, m affine, fg color.NRGBA) colorFn {
	inv := m.invert()

	if r == 0 {
		return p.stopsAt(l, fg, 1)
	}

	return func(px, py float32) color.NRGBA {
		x, y := inv.apply(px, py)
		dx, dy := x-float32(cx), y-float32(cy)
		t := float32(math.Sqrt(float64(dx*dx+dy*dy))) / float32(r)
		return p.colorAt(l, t, fg)
	}
}

func (p *painter) fillGlyph(gid font.GID, m affine, src colorFn) {
	outline, ok := p.f.GlyphDataOutline(gid)

	if !ok {
		return
	}

	mask, off := rasterize([]ot.Segment(outline.Segments), m)

	if mask == nil {
		return
	}

	b := p.dst.Bounds()
	mw, mh := mask.Rect.Dx(), mask.Rect.Dy()

	for y := 0; y < mh; y++ {
		dy := off.Y + y

		if dy < b.Min.Y || dy >= b.Max.Y {
			continue
		}

		for x := 0; x < mw; x++ {
			cov := mask.Pix[y*mask.Stride+x]

			if cov == 0 {
				continue
			}

			dx := off.X + x

			if dx < b.Min.X || dx >= b.Max.X {
				continue
			}

			c := src(float32(dx)+0.5, float32(dy)+0.5)
			a := uint32(c.A) * uint32(cov) / 255
			blendPremul(p.dst, dx, dy, c, a)
		}
	}
}

func blendPremul(dst *image.RGBA, x, y int, c color.NRGBA, a uint32) {
	if a == 0 {
		return
	}

	i := dst.PixOffset(x, y)
	pix := dst.Pix[i : i+4 : i+4]
	ia := 255 - a
	pix[0] = uint8((uint32(c.R)*a + uint32(pix[0])*ia) / 255)
	pix[1] = uint8((uint32(c.G)*a + uint32(pix[1])*ia) / 255)
	pix[2] = uint8((uint32(c.B)*a + uint32(pix[2])*ia) / 255)
	pix[3] = uint8(a + uint32(pix[3])*ia/255)
}

func lerp(a, b color.NRGBA, k float32) color.NRGBA {
	l := func(x, y uint8) uint8 { return uint8(float32(x) + (float32(y)-float32(x))*k) }
	return color.NRGBA{R: l(a.R, b.R), G: l(a.G, b.G), B: l(a.B, b.B), A: l(a.A, b.A)}
}

func transformOf(t tables.PaintTable) (tables.PaintTable, affine, bool) {
	around := func(cx, cy int16, n affine) affine {
		x, y := float32(cx), float32(cy)
		return identity().translate(x, y).mul(n).translate(-x, -y)
	}
	rot := func(angle tables.Fixed214) affine {
		r := float64(f214(angle)) * math.Pi
		s, c := float32(math.Sin(r)), float32(math.Cos(r))
		return affine{a: c, b: s, c: -s, d: c}
	}
	skew := func(ax, ay tables.Fixed214) affine {
		tx := float32(math.Tan(float64(f214(ax)) * math.Pi))
		ty := float32(math.Tan(float64(f214(ay)) * math.Pi))
		return affine{a: 1, b: ty, c: -tx, d: 1}
	}

	switch v := t.(type) {
	case tables.PaintTransform:
		tr := v.Transform
		return v.Paint, affine{a: tr.Xx, b: tr.Yx, c: tr.Xy, d: tr.Yy, e: tr.Dx, f: tr.Dy}, true
	case tables.PaintVarTransform:
		tr := v.Transform
		return v.Paint, affine{a: tr.Xx, b: tr.Yx, c: tr.Xy, d: tr.Yy, e: tr.Dx, f: tr.Dy}, true
	case tables.PaintTranslate:
		return v.Paint, identity().translate(float32(v.Dx), float32(v.Dy)), true
	case tables.PaintVarTranslate:
		return v.Paint, identity().translate(float32(v.Dx), float32(v.Dy)), true
	case tables.PaintScale:
		return v.Paint, affine{a: f214(v.ScaleX), d: f214(v.ScaleY)}, true
	case tables.PaintVarScale:
		return v.Paint, affine{a: f214(v.ScaleX), d: f214(v.ScaleY)}, true
	case tables.PaintScaleAroundCenter:
		return v.Paint, around(v.CenterX, v.CenterY, affine{a: f214(v.ScaleX), d: f214(v.ScaleY)}), true
	case tables.PaintVarScaleAroundCenter:
		return v.Paint, around(v.CenterX, v.CenterY, affine{a: f214(v.ScaleX), d: f214(v.ScaleY)}), true
	case tables.PaintScaleUniform:
		return v.Paint, affine{a: f214(v.Scale), d: f214(v.Scale)}, true
	case tables.PaintVarScaleUniform:
		return v.Paint, affine{a: f214(v.Scale), d: f214(v.Scale)}, true
	case tables.PaintScaleUniformAroundCenter:
		return v.Paint, around(v.CenterX, v.CenterY, affine{a: f214(v.Scale), d: f214(v.Scale)}), true
	case tables.PaintVarScaleUniformAroundCenter:
		return v.Paint, around(v.CenterX, v.CenterY, affine{a: f214(v.Scale), d: f214(v.Scale)}), true
	case tables.PaintRotate:
		return v.Paint, rot(v.Angle), true
	case tables.PaintVarRotate:
		return v.Paint, rot(v.Angle), true
	case tables.PaintRotateAroundCenter:
		return v.Paint, around(v.CenterX, v.CenterY, rot(v.Angle)), true
	case tables.PaintVarRotateAroundCenter:
		return v.Paint, around(v.CenterX, v.CenterY, rot(v.Angle)), true
	case tables.PaintSkew:
		return v.Paint, skew(v.XSkewAngle, v.YSkewAngle), true
	case tables.PaintVarSkew:
		return v.Paint, skew(v.XSkewAngle, v.YSkewAngle), true
	case tables.PaintSkewAroundCenter:
		return v.Paint, around(v.CenterX, v.CenterY, skew(v.XSkewAngle, v.YSkewAngle)), true
	case tables.PaintVarSkewAroundCenter:
		return v.Paint, around(v.CenterX, v.CenterY, skew(v.XSkewAngle, v.YSkewAngle)), true
	}

	return nil, affine{}, false
}
