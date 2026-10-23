package app

import (
	"image"
	"math"
	"strconv"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/widget"

	"github.com/plattyssh/platty/internal/session"
	"github.com/plattyssh/platty/internal/sshx"
	"github.com/plattyssh/platty/internal/ui/kit"
	"github.com/plattyssh/platty/internal/ui/sol"
)

type connecting struct {
	start     time.Time
	left      time.Time
	orbit     float64
	cancel    widget.Clickable
	cancelled bool
}

const (
	solPlaySize  = 96
	stageFadeIn  = 220 * time.Millisecond
	stageFadeOut = 320 * time.Millisecond
)

func easeOut(x float64) float32 {
	x = min(max(x, 0), 1)
	return float32(1 - math.Pow(1-x, 3))
}

func (a *App) paneOverlay(gtx C, p *Pane) {
	cv, f := &p.connecting, &p.failure
	now := gtx.Now

	switch {
	case p.isConnecting():
		if cv.start.IsZero() {
			cv.start, cv.left = now, time.Time{}

			if !f.start.IsZero() {
				cv.start = now.Add(-stageFadeIn)
			}
		}

		f.start = time.Time{}
		t := now.Sub(cv.start).Seconds()
		a.connectingView(gtx, p, t, easeOut(t/stageFadeIn.Seconds()))
	case p.failed():
		if f.start.IsZero() {
			f.start = now

			if !cv.start.IsZero() {
				f.orbit = now.Sub(cv.start).Seconds()
			}
		}

		cv.start = time.Time{}
		a.failedView(gtx, p)
	default:
		if !cv.start.IsZero() {
			cv.left, cv.orbit, cv.start = now, now.Sub(cv.start).Seconds(), time.Time{}
		}

		f.start = time.Time{}

		if d := now.Sub(cv.left); !cv.left.IsZero() && d < stageFadeOut {
			a.connectingView(gtx, p, cv.orbit+d.Seconds(), 1-easeOut(d.Seconds()/stageFadeOut.Seconds()))
		}

		if p.lost() {
			a.lostBanner(gtx, p)
		} else {
			f.shown = time.Time{}
		}
	}
}

func (a *App) stage(gtx C, opacity float32, scene, body layout.Widget) {
	th := a.th()
	size := gtx.Constraints.Max
	if opacity < 1 {
		defer paint.PushOpacity(gtx.Ops, max(opacity, 0.01)).Pop()
	}

	kit.Rect(gtx, image.Rectangle{Max: size}, th.Editor)

	box := gtx.Dp(solPlaySize) * 2
	top := max(size.Y/2-box/2-gtx.Dp(64), 0)

	off := layoutAt(gtx, image.Pt((size.X-box)/2, top))
	scene(gtx)
	off.Pop()

	bg := gtx
	bg.Constraints = layout.Constraints{Min: image.Pt(size.X, 0), Max: image.Pt(size.X, size.Y-top-box)}
	off = layoutAt(gtx, image.Pt(0, top+box))
	layout.N.Layout(bg, func(gtx C) D {
		gtx.Constraints.Min.X = 0
		gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(460))
		return body(gtx)
	})
	off.Pop()
}

func layoutAt(gtx C, at image.Point) op.TransformStack {
	return op.Offset(at).Push(gtx.Ops)
}

func (a *App) connectingView(gtx C, p *Pane, t float64, opacity float32) {
	k := a.kit
	th := a.th()
	cv := &p.connecting

	if cv.cancel.Clicked(gtx) {
		cv.cancelled = true
	}

	steps := p.sess.Steps()
	step := "Starting…"

	if len(steps) > 0 {
		step = steps[len(steps)-1].Text
	}

	who := sshx.User(p.sess.Target().Host) + "@" + p.sess.Target().Host.Address

	a.stage(gtx, opacity,
		func(gtx C) D { return sol.Playing(gtx, th, solPlaySize, t) },
		func(gtx C) D {
			return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(kit.Gap(2)),
				layout.Rigid(func(gtx C) D {
					return k.Label(gtx, kit.TextStyle{Size: th.TextSizeLarge}, "Connecting to "+who)
				}),
				layout.Rigid(kit.Gap(6)),
				layout.Rigid(func(gtx C) D {
					return k.Label(gtx, kit.TextStyle{Size: th.TextSizeSmall, Color: th.TextMuted}, step+"  ·  "+strconv.Itoa(int(t))+" s")
				}),
				layout.Rigid(kit.Gap(18)),
				layout.Rigid(func(gtx C) D { return k.Button(gtx, &cv.cancel, kit.Ghost, nil, "Cancel") }),
			)
		})
	gtx.Execute(op.InvalidateCmd{})
}

func (p *Pane) isConnecting() bool {
	st, _ := p.sess.State()
	return p.sess.Kind == session.SSH && st == session.Connecting
}
