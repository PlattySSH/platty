package mascot

import (
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

type Colors struct {
	Glass    color.NRGBA
	Ink      color.NRGBA
	Shine    color.NRGBA
	Backdrop color.NRGBA
	Light    bool
}

var DefaultColors = Colors{
	Glass: color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	Ink:   color.NRGBA{R: 0x16, G: 0x16, B: 0x18, A: 0xff},
	Shine: color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
}

var LightColors = Colors{
	Glass: DefaultColors.Glass,
	Ink:   DefaultColors.Ink,
	Shine: DefaultColors.Shine,
	Light: true,
}

type State struct {
	T      float64
	Look   f32.Point
	Blink  float32
	Happy  float32
	Shadow bool
}

const Period = 4.2

const waves = 7

func BlinkAt(t float64) float32 {
	p := math.Mod(t, Period)
	blink := func(at float64) float32 {
		d := math.Abs(p - at)

		if d > 0.09 {
			return 0
		}

		return float32(1 - d/0.09)
	}
	b := blink(Period - 0.2)

	if int(t/Period)%3 == 1 {
		b = max(b, blink(Period-0.45))
	}

	return b
}

func Draw(ops *op.Ops, size float32, st State, c Colors) {
	DrawBody(ops, size, st, c)
	DrawEyes(ops, size, st, c)
}

func DrawBody(ops *op.Ops, size float32, st State, c Colors) {
	s := size
	center := f32.Pt(s/2, s*0.48)
	r := s * 0.41

	phase := st.T * 2 * math.Pi / Period / waves
	breath := math.Sin(st.T * 4 * math.Pi / Period)
	outer := func(grow, dy float32) clip.PathSpec {
		return blob(ops, center.Add(f32.Pt(0, dy)), r+grow, 0.06+0.012*breath, phase)
	}
	inner := func() clip.PathSpec {
		return blob(ops, center, r*0.83, 0.045-0.01*breath, -phase+math.Pi/waves)
	}

	shadowA := 0.3

	if c.Light {
		shadowA = 0.12
	}

	for i := 3; st.Shadow && i >= 0; i-- {
		a := shadowA * (1 - float64(i)/4) / 2.2
		rr := r*1.02 + s*0.012*float32(i)
		fillEllipse(ops, color.NRGBA{A: uint8(255 * a)}, center.X, center.Y+s*0.035, rr, rr)
	}

	ring := color.NRGBA{R: 0xd9, G: 0xda, B: 0xde, A: 0xff}
	top, bottom := alpha(ring, 0.78), alpha(ring, 0.5)

	if c.Backdrop.A != 0 {
		top, bottom = over(c.Backdrop, ring, 0.78), over(c.Backdrop, ring, 0.5)
	}

	fillGradient(ops, outer(0, 0), f32.Pt(0, center.Y-r), f32.Pt(0, center.Y+r), top, bottom)
	rimFrom, rimTo := alpha(c.Shine, 0.9), alpha(c.Shine, 0.08)

	if c.Light {
		rimFrom, rimTo = alpha(c.Ink, 0.12), alpha(c.Ink, 0.32)
	}

	strokeGradient(ops, outer(0, 0), max(s*0.01, 1), f32.Pt(center.X-r, center.Y-r), f32.Pt(center.X+r, center.Y+r), rimFrom, rimTo)

	fillGradient(ops, inner(), f32.Pt(0, center.Y-r*0.8), f32.Pt(0, center.Y+r*0.8),
		c.Glass, color.NRGBA{R: 0xe6, G: 0xe7, B: 0xeb, A: 0xff})
	strokeGradient(ops, inner(), max(s*0.006, 1), f32.Pt(center.X, center.Y+r), f32.Pt(center.X, center.Y-r*0.2),
		alpha(color.NRGBA{R: 0xb8, G: 0xba, B: 0xc2, A: 0xff}, 0.9), alpha(c.Shine, 0))

	glare := op.Affine(f32.Affine2D{}.Rotate(center, -0.5)).Push(ops)
	fillGradient(ops, ellipse(ops, center.X-r*0.04, center.Y-r*0.7, r*0.5, r*0.16),
		f32.Pt(0, center.Y-r*0.86), f32.Pt(0, center.Y-r*0.54),
		alpha(c.Shine, 0.9), alpha(c.Shine, 0))
	glare.Pop()
}

func DrawEyes(ops *op.Ops, size float32, st State, c Colors) {
	s := size
	center := f32.Pt(s/2, s*0.48)

	look := f32.Pt(clamp(st.Look.X), clamp(st.Look.Y)).Mul(s * 0.03)
	twinkle := float32(1 + 0.12*math.Sin(st.T*6*math.Pi/Period))

	for _, sx := range []float32{-1, 1} {
		eye := f32.Pt(center.X+sx*s*0.12, center.Y+s*0.01).Add(look)
		rx, ry := s*0.058, s*0.08
		open := 1 - max(st.Blink, st.Happy)

		if open < 0.15 {
			closedEye(ops, s, eye, rx, st.Happy > st.Blink, c.Ink)
			continue
		}

		ry *= open
		deep := color.NRGBA{R: 0x3a, G: 0x3d, B: 0x4a, A: 0xff}
		fillGradient(ops, ellipse(ops, eye.X, eye.Y, rx, ry), f32.Pt(0, eye.Y-ry), f32.Pt(0, eye.Y+ry), c.Ink, deep)
		fillGradient(ops, ellipse(ops, eye.X-rx*0.1, eye.Y+ry*0.5, rx*0.62, ry*0.3),
			f32.Pt(0, eye.Y+ry*0.85), f32.Pt(0, eye.Y+ry*0.25),
			alpha(c.Shine, 0.28), alpha(c.Shine, 0))

		if s >= 20 {
			hr := s * 0.024 * twinkle
			h := f32.Pt(eye.X+rx*0.36, eye.Y-ry*0.42)
			fillEllipse(ops, c.Shine, h.X, h.Y, hr, hr*min(1, open*1.4))
		}
	}
}

func strokeGradient(ops *op.Ops, path clip.PathSpec, width float32, a, b f32.Point, from, to color.NRGBA) {
	st := clip.Stroke{Path: path, Width: width}.Op().Push(ops)
	paint.LinearGradientOp{Stop1: a, Color1: from, Stop2: b, Color2: to}.Add(ops)
	paint.PaintOp{}.Add(ops)
	st.Pop()
}

type Cubic struct{ P0, C1, C2, P3 f32.Point }

func blobCubics(c f32.Point, r float32, amp, phase float64) []Cubic {
	const n = waves * 2
	pts := make([]f32.Point, n)

	for i := range pts {
		a := float64(i)/n*2*math.Pi + phase
		rr := float64(r) * (1 + amp*math.Cos(waves*(a-phase)))
		pts[i] = f32.Pt(c.X+float32(rr*math.Cos(a)), c.Y+float32(rr*math.Sin(a)))
	}

	at := func(i int) f32.Point { return pts[(i%n+n)%n] }
	out := make([]Cubic, n)

	for i := range out {
		p0, p1, p2, p3 := at(i-1), at(i), at(i+1), at(i+2)
		out[i] = Cubic{p1, p1.Add(p2.Sub(p0).Mul(1.0 / 6)), p2.Sub(p3.Sub(p1).Mul(1.0 / 6)), p2}
	}

	return out
}

func blob(ops *op.Ops, c f32.Point, r float32, amp, phase float64) clip.PathSpec {
	cs := blobCubics(c, r, amp, phase)
	var p clip.Path
	p.Begin(ops)
	p.MoveTo(cs[0].P0)

	for _, q := range cs {
		p.CubeTo(q.C1, q.C2, q.P3)
	}

	p.Close()
	return p.End()
}

func fillGradient(ops *op.Ops, path clip.PathSpec, a, b f32.Point, from, to color.NRGBA) {
	st := clip.Outline{Path: path}.Op().Push(ops)
	paint.LinearGradientOp{Stop1: a, Color1: from, Stop2: b, Color2: to}.Add(ops)
	paint.PaintOp{}.Add(ops)
	st.Pop()
}

func closedEye(ops *op.Ops, s float32, eye f32.Point, rx float32, happy bool, ink color.NRGBA) {
	bend := s * 0.035

	if happy {
		bend = -bend
	}

	var p clip.Path
	p.Begin(ops)
	p.MoveTo(f32.Pt(eye.X-rx, eye.Y))
	p.QuadTo(f32.Pt(eye.X, eye.Y+bend), f32.Pt(eye.X+rx, eye.Y))
	paint.FillShape(ops, ink, clip.Stroke{Path: p.End(), Width: max(s*0.028, 1.2)}.Op())
}

func alpha(c color.NRGBA, a float64) color.NRGBA {
	c.A = uint8(math.Round(float64(c.A) * a))
	return c
}

func clamp(v float32) float32 { return max(-1, min(1, v)) }

const kappa = 0.5522847

func ellipse(ops *op.Ops, cx, cy, rx, ry float32) clip.PathSpec {
	var p clip.Path
	p.Begin(ops)
	ox, oy := rx*kappa, ry*kappa
	p.MoveTo(f32.Pt(cx+rx, cy))
	p.CubeTo(f32.Pt(cx+rx, cy+oy), f32.Pt(cx+ox, cy+ry), f32.Pt(cx, cy+ry))
	p.CubeTo(f32.Pt(cx-ox, cy+ry), f32.Pt(cx-rx, cy+oy), f32.Pt(cx-rx, cy))
	p.CubeTo(f32.Pt(cx-rx, cy-oy), f32.Pt(cx-ox, cy-ry), f32.Pt(cx, cy-ry))
	p.CubeTo(f32.Pt(cx+ox, cy-ry), f32.Pt(cx+rx, cy-oy), f32.Pt(cx+rx, cy))
	p.Close()
	return p.End()
}

func fillEllipse(ops *op.Ops, c color.NRGBA, cx, cy, rx, ry float32) {
	paint.FillShape(ops, c, clip.Outline{Path: ellipse(ops, cx, cy, rx, ry)}.Op())
}

func over(bg, fg color.NRGBA, a float64) color.NRGBA {
	lin := func(v uint8) float64 {
		c := float64(v) / 255

		if c <= 0.04045 {
			return c / 12.92
		}

		return math.Pow((c+0.055)/1.055, 2.4)
	}

	srgb := func(l float64) uint8 {
		if l <= 0.0031308 {
			return uint8(math.Round(l * 12.92 * 255))
		}

		return uint8(math.Round((1.055*math.Pow(l, 1/2.4) - 0.055) * 255))
	}

	mix := func(b, f uint8) uint8 { return srgb(lin(b)*(1-a) + lin(f)*a) }
	return color.NRGBA{R: mix(bg.R, fg.R), G: mix(bg.G, fg.G), B: mix(bg.B, fg.B), A: 0xff}
}
