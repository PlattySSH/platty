package mascot

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"golang.org/x/image/vector"
)

type linear struct {
	a, b     f32.Point
	from, to [4]float64
}

func newLinear(a, b f32.Point, from, to color.NRGBA) linear {
	return linear{a: a, b: b, from: toLinear(from), to: toLinear(to)}
}

func toLinear(c color.NRGBA) [4]float64 {
	ch := func(v uint8) float64 {
		x := float64(v) / 255

		if x <= 0.04045 {
			return x / 12.92
		}

		return math.Pow((x+0.055)/1.055, 2.4)
	}

	return [4]float64{ch(c.R), ch(c.G), ch(c.B), float64(c.A) / 255}
}

func toSRGB(x float64) uint8 {
	x = min(max(x, 0), 1)

	if x <= 0.0031308 {
		return uint8(math.Round(x * 12.92 * 255))
	}

	return uint8(math.Round((1.055*math.Pow(x, 1/2.4) - 0.055) * 255))
}

func (g linear) ColorModel() color.Model { return color.NRGBAModel }

func (g linear) Bounds() image.Rectangle { return image.Rect(-1<<20, -1<<20, 1<<20, 1<<20) }

func (g linear) At(x, y int) color.Color {
	d := g.b.Sub(g.a)
	l := d.X*d.X + d.Y*d.Y
	t := float64(0)

	if l > 0 {
		p := f32.Pt(float32(x)+0.5, float32(y)+0.5).Sub(g.a)
		t = min(max(float64((p.X*d.X+p.Y*d.Y)/l), 0), 1)
	}

	var c [4]float64

	for i := range c {
		c[i] = g.from[i] + (g.to[i]-g.from[i])*t
	}

	return color.NRGBA{R: toSRGB(c[0]), G: toSRGB(c[1]), B: toSRGB(c[2]), A: uint8(math.Round(c[3] * 255))}
}

type shape func(z *vector.Rasterizer)

func blobShape(c f32.Point, r float32, amp, phase float64, reverse bool) shape {
	return func(z *vector.Rasterizer) {
		cs := blobCubics(c, r, amp, phase)

		if !reverse {
			z.MoveTo(cs[0].P0.X, cs[0].P0.Y)

			for _, q := range cs {
				z.CubeTo(q.C1.X, q.C1.Y, q.C2.X, q.C2.Y, q.P3.X, q.P3.Y)
			}

			z.ClosePath()
			return
		}

		last := cs[len(cs)-1]
		z.MoveTo(last.P3.X, last.P3.Y)

		for i := len(cs) - 1; i >= 0; i-- {
			q := cs[i]
			z.CubeTo(q.C2.X, q.C2.Y, q.C1.X, q.C1.Y, q.P0.X, q.P0.Y)
		}

		z.ClosePath()
	}
}

func blobRing(c f32.Point, r float32, amp, phase float64, width float32) shape {
	h := width / 2
	outer := blobShape(c, r+h, amp*float64(r/(r+h)), phase, false)
	inner := blobShape(c, r-h, amp*float64(r/(r-h)), phase, true)

	return func(z *vector.Rasterizer) {
		outer(z)
		inner(z)
	}
}

func ellipseShape(c f32.Point, rx, ry float32, rot float32, pivot f32.Point) shape {
	tr := f32.Affine2D{}.Rotate(pivot, rot)

	return func(z *vector.Rasterizer) {
		pt := func(x, y float32) f32.Point { return tr.Transform(f32.Pt(c.X+x, c.Y+y)) }
		ox, oy := rx*kappa, ry*kappa
		start := pt(rx, 0)
		z.MoveTo(start.X, start.Y)

		segs := [][3]f32.Point{
			{pt(rx, oy), pt(ox, ry), pt(0, ry)},
			{pt(-ox, ry), pt(-rx, oy), pt(-rx, 0)},
			{pt(-rx, -oy), pt(-ox, -ry), pt(0, -ry)},
			{pt(ox, -ry), pt(rx, -oy), pt(rx, 0)},
		}

		for _, s := range segs {
			z.CubeTo(s[0].X, s[0].Y, s[1].X, s[1].Y, s[2].X, s[2].Y)
		}

		z.ClosePath()
	}
}

func fill(dst *image.RGBA, s shape, src image.Image) {
	b := dst.Bounds()
	z := vector.NewRasterizer(b.Dx(), b.Dy())
	s(z)
	z.Draw(dst, b, src, image.Point{})
}

func RenderBody(px int, st State, c Colors) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, px, px))
	s := float32(px)
	center := f32.Pt(s/2, s*0.48)
	r := s * 0.41

	phase := st.T * 2 * math.Pi / Period / waves
	breath := math.Sin(st.T * 4 * math.Pi / Period)
	outerAmp := 0.06 + 0.012*breath
	innerAmp, innerPhase := 0.045-0.01*breath, -phase+math.Pi/waves

	ring := color.NRGBA{R: 0xd9, G: 0xda, B: 0xde, A: 0xff}
	top, bottom := alpha(ring, 0.78), alpha(ring, 0.5)

	if c.Backdrop.A != 0 {
		top, bottom = over(c.Backdrop, ring, 0.78), over(c.Backdrop, ring, 0.5)
	}

	fill(img, blobShape(center, r, outerAmp, phase, false), newLinear(f32.Pt(0, center.Y-r), f32.Pt(0, center.Y+r), top, bottom))

	rimFrom, rimTo := alpha(c.Shine, 0.9), alpha(c.Shine, 0.08)

	if c.Light {
		rimFrom, rimTo = alpha(c.Ink, 0.12), alpha(c.Ink, 0.32)
	}

	fill(img, blobRing(center, r, outerAmp, phase, max(s*0.01, 1)), newLinear(f32.Pt(center.X-r, center.Y-r), f32.Pt(center.X+r, center.Y+r), rimFrom, rimTo))

	fill(img, blobShape(center, r*0.83, innerAmp, innerPhase, false),
		newLinear(f32.Pt(0, center.Y-r*0.8), f32.Pt(0, center.Y+r*0.8), c.Glass, color.NRGBA{R: 0xe6, G: 0xe7, B: 0xeb, A: 0xff}))

	fill(img, blobRing(center, r*0.83, innerAmp, innerPhase, max(s*0.006, 1)),
		newLinear(f32.Pt(center.X, center.Y+r), f32.Pt(center.X, center.Y-r*0.2), alpha(color.NRGBA{R: 0xb8, G: 0xba, B: 0xc2, A: 0xff}, 0.9), alpha(c.Shine, 0)))

	rot := f32.Affine2D{}.Rotate(center, -0.5)
	fill(img, ellipseShape(f32.Pt(center.X-r*0.04, center.Y-r*0.7), r*0.5, r*0.16, -0.5, center),
		newLinear(rot.Transform(f32.Pt(center.X, center.Y-r*0.86)), rot.Transform(f32.Pt(center.X, center.Y-r*0.54)), alpha(c.Shine, 0.9), alpha(c.Shine, 0)))

	return img
}
