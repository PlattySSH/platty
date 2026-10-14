package sol

import (
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/plattyssh/platty/internal/ui/kit"
	"github.com/plattyssh/platty/internal/ui/mascot"
	"github.com/plattyssh/platty/internal/ui/theme"
)

func Playing(gtx layout.Context, th *theme.Theme, size unit.Dp, t float64) layout.Dimensions {
	px := gtx.Dp(size)
	box := px * 2
	center := f32.Pt(float32(box)/2, float32(box)/2)

	ang := t * 2 * math.Pi / 2.4
	rx, ry := float64(px)*1.0, float64(px)*0.38
	tilt := -0.35
	ox, oy := rx*math.Cos(ang), ry*math.Sin(ang)
	star := f32.Pt(
		center.X+float32(ox*math.Cos(tilt)-oy*math.Sin(tilt)),
		center.Y+float32(ox*math.Sin(tilt)+oy*math.Cos(tilt)),
	)
	behind := math.Sin(ang) < 0
	starSize := float32(px) * 0.17 * float32(1+0.15*math.Sin(t*9))

	colors := mascot.DefaultColors

	if !th.Dark {
		colors = mascot.LightColors
	}

	rock := float32(0.18 * math.Sin(t*2*math.Pi/2.4))
	bob := float32(math.Abs(math.Sin(t*2*math.Pi/1.2))) * float32(px) * 0.05
	look := star.Sub(center)
	st := mascot.State{T: t, Blink: mascot.BlinkAt(t), Look: f32.Pt(look.X/float32(rx), look.Y/float32(ry))}

	starCol := kit.Lerp(th.Text, th.Editor, 0.05)

	if behind {
		Star(gtx, star, starSize*0.8, kit.Fade(starCol, 0.55))
	}

	o := f32.Pt((float32(box)-float32(px))/2, (float32(box)-float32(px))/2-bob)
	tr := f32.Affine2D{}.Rotate(f32.Pt(float32(px)/2, float32(px)/2), rock).Offset(o)
	s := op.Affine(tr).Push(gtx.Ops)
	mascot.Draw(gtx.Ops, float32(px), st, colors)
	s.Pop()

	if !behind {
		Star(gtx, star, starSize, starCol)
	}

	return layout.Dimensions{Size: image.Pt(box, box)}
}

func Star(gtx layout.Context, c f32.Point, r float32, col color.NRGBA) {
	starShape(gtx, c, r*1.25, color.NRGBA{A: 0x50})
	starShape(gtx, c, r, col)
}

func starShape(gtx layout.Context, c f32.Point, r float32, col color.NRGBA) {
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(c.X, c.Y-r))
	p.QuadTo(c, f32.Pt(c.X+r, c.Y))
	p.QuadTo(c, f32.Pt(c.X, c.Y+r))
	p.QuadTo(c, f32.Pt(c.X-r, c.Y))
	p.QuadTo(c, f32.Pt(c.X, c.Y-r))
	p.Close()
	paint.FillShape(gtx.Ops, col, clip.Outline{Path: p.End()}.Op())
}
