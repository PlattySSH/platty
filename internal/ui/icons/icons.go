package icons

import (
	"image"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"

	"github.com/plattyssh/platty/internal/ui/shape"
)

type Icon struct {
	draw func(ops *op.Ops, size float32, c color.NRGBA)
}

func custom(fn func(ops *op.Ops, size float32, c color.NRGBA)) *Icon {
	return &Icon{draw: fn}
}

func (i *Icon) Layout(gtx layout.Context, size unit.Dp, c color.NRGBA) layout.Dimensions {
	px := gtx.Dp(size)
	gtx.Constraints = layout.Exact(image.Pt(px, px))

	i.draw(gtx.Ops, float32(px), c)
	return layout.Dimensions{Size: image.Pt(px, px)}
}

var (
	SidebarLeft  = sidebar(false)
	SidebarRight = sidebar(true)
)

func sidebar(right bool) *Icon {
	return custom(func(ops *op.Ops, s float32, c color.NRGBA) {
		stroke := max(s/14, 1.2)
		r := image.Rect(int(s*0.12), int(s*0.2), int(s*0.88), int(s*0.8))
		paint.FillShape(ops, c, clip.Outline{Path: shape.Ring(ops, r, float32(int(s*0.1)), stroke)}.Op())

		col := image.Rect(r.Min.X, r.Min.Y, r.Min.X+int(s*0.28), r.Max.Y)

		if right {
			col = image.Rect(r.Max.X-int(s*0.28), r.Min.Y, r.Max.X, r.Max.Y)
		}

		paint.FillShape(ops, c, clip.UniformRRect(col, int(s*0.1)).Op(ops))
	})
}

func Logo(gtx layout.Context, size unit.Dp, bg, fg color.NRGBA) layout.Dimensions {
	px := gtx.Dp(size)
	s := float32(px)
	rr := clip.UniformRRect(image.Rect(0, 0, px, px), px/4)
	paint.FillShape(gtx.Ops, bg, rr.Op(gtx.Ops))

	stroke := max(s/9, 1.5)
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(s*0.26, s*0.30))
	p.LineTo(f32.Pt(s*0.46, s*0.50))
	p.LineTo(f32.Pt(s*0.26, s*0.70))
	paint.FillShape(gtx.Ops, fg, clip.Stroke{Path: p.End(), Width: stroke}.Op())
	paint.FillShape(gtx.Ops, fg, clip.Rect{
		Min: image.Pt(int(s*0.52), int(s*0.66)),
		Max: image.Pt(int(s*0.76), int(s*0.66+stroke)),
	}.Op())

	return layout.Dimensions{Size: image.Pt(px, px)}
}

func LogoGradient(gtx layout.Context, size unit.Dp, from, to, fg color.NRGBA) layout.Dimensions {
	px := gtx.Dp(size)
	s := float32(px)

	rr := clip.UniformRRect(image.Rect(0, 0, px, px), px*3/10).Push(gtx.Ops)
	paint.LinearGradientOp{Stop1: f32.Pt(0, 0), Color1: from, Stop2: f32.Pt(s, s), Color2: to}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	rr.Pop()

	stroke := max(s/9, 1.5)
	var p clip.Path
	p.Begin(gtx.Ops)
	p.MoveTo(f32.Pt(s*0.27, s*0.31))
	p.LineTo(f32.Pt(s*0.46, s*0.50))
	p.LineTo(f32.Pt(s*0.27, s*0.69))
	paint.FillShape(gtx.Ops, fg, clip.Stroke{Path: p.End(), Width: stroke}.Op())
	paint.FillShape(gtx.Ops, fg, clip.UniformRRect(image.Rect(int(s*0.52), int(s*0.64), int(s*0.75), int(s*0.64+stroke)), int(stroke/2)).Op(gtx.Ops))

	return layout.Dimensions{Size: image.Pt(px, px)}
}

var Minimize = custom(func(ops *op.Ops, s float32, c color.NRGBA) {
	y := int(s / 2)
	paint.FillShape(ops, c, clip.Rect{Min: image.Pt(int(s*0.3), y), Max: image.Pt(int(s*0.7), y+1)}.Op())
})

var Maximize = custom(func(ops *op.Ops, s float32, c color.NRGBA) {
	r := image.Rect(int(s*0.3), int(s*0.3), int(s*0.7), int(s*0.7))
	strokeRect(ops, r, c)
})

var Restore = custom(func(ops *op.Ops, s float32, c color.NRGBA) {
	back := image.Rect(int(s*0.38), int(s*0.26), int(s*0.74), int(s*0.62))
	front := image.Rect(int(s*0.26), int(s*0.38), int(s*0.62), int(s*0.74))
	paint.FillShape(ops, c, clip.Rect{Min: back.Min, Max: image.Pt(back.Max.X, back.Min.Y+1)}.Op())
	paint.FillShape(ops, c, clip.Rect{Min: image.Pt(back.Max.X-1, back.Min.Y), Max: image.Pt(back.Max.X, back.Max.Y)}.Op())
	strokeRect(ops, front, c)
})

var WindowClose = custom(func(ops *op.Ops, s float32, c color.NRGBA) {
	var p clip.Path
	p.Begin(ops)
	p.MoveTo(f32.Pt(s*0.3, s*0.3))
	p.LineTo(f32.Pt(s*0.7, s*0.7))
	p.MoveTo(f32.Pt(s*0.7, s*0.3))
	p.LineTo(f32.Pt(s*0.3, s*0.7))
	paint.FillShape(ops, c, clip.Stroke{Path: p.End(), Width: 1}.Op())
})

func strokeRect(ops *op.Ops, r image.Rectangle, c color.NRGBA) {
	paint.FillShape(ops, c, clip.Rect{Min: r.Min, Max: image.Pt(r.Max.X, r.Min.Y+1)}.Op())
	paint.FillShape(ops, c, clip.Rect{Min: image.Pt(r.Min.X, r.Max.Y-1), Max: r.Max}.Op())
	paint.FillShape(ops, c, clip.Rect{Min: r.Min, Max: image.Pt(r.Min.X+1, r.Max.Y)}.Op())
	paint.FillShape(ops, c, clip.Rect{Min: image.Pt(r.Max.X-1, r.Min.Y), Max: r.Max}.Op())
}

var MoreH = custom(func(ops *op.Ops, s float32, c color.NRGBA) {
	d := max(int(s*0.16+0.5), 3)
	gap := max(int(s*0.12+0.5), 2)
	x := (int(s) - 3*d - 2*gap) / 2
	y := (int(s) - d) / 2

	for i := 0; i < 3; i++ {
		r := image.Rect(x, y, x+d, y+d)
		paint.FillShape(ops, c, clip.Ellipse(r).Op(ops))
		x += d + gap
	}
})
