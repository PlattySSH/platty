package app

import (
	"image"
	"strconv"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"

	"github.com/plattyssh/platty/internal/session"
	"github.com/plattyssh/platty/internal/sshx"
	"github.com/plattyssh/platty/internal/ui/kit"
	"github.com/plattyssh/platty/internal/ui/sol"
)

type connecting struct {
	start     time.Time
	cancel    widget.Clickable
	cancelled bool
}

const solPlaySize = 96

func (a *App) connectingView(gtx C, p *Pane) {
	k := a.kit
	th := a.th()
	cv := &p.connecting

	if cv.start.IsZero() {
		cv.start = gtx.Now
	}

	if cv.cancel.Clicked(gtx) {
		cv.cancelled = true
	}

	size := gtx.Constraints.Max
	kit.Rect(gtx, image.Rectangle{Max: size}, th.Editor)

	t := gtx.Now.Sub(cv.start).Seconds()
	steps := p.sess.Steps()
	step := "Starting…"

	if len(steps) > 0 {
		step = steps[len(steps)-1].Text
	}

	who := sshx.User(p.sess.Target().Host) + "@" + p.sess.Target().Host.Address

	gtx.Constraints = layout.Exact(size)
	layout.Center.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min = image.Point{}
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D { return sol.Playing(gtx, a.th(), solPlaySize, t) }),
			layout.Rigid(kit.Gap(2)),
			layout.Rigid(func(gtx C) D {
				return k.Label(gtx, kit.TextStyle{Size: th.TextSizeLarge}, "Connecting to "+who)
			}),
			layout.Rigid(kit.Gap(6)),
			layout.Rigid(func(gtx C) D {
				return k.Label(gtx, kit.TextStyle{Size: th.TextSizeSmall, Color: th.TextMuted}, step+"  ·  "+strconv.Itoa(int(t))+" s")
			}),
			layout.Rigid(kit.Gap(18)),
			layout.Rigid(func(gtx C) D { return k.Button(gtx, &cv.cancel, kit.Secondary, nil, "Cancel") }),
		)
	})
	gtx.Execute(op.InvalidateCmd{})
}

func (p *Pane) isConnecting() bool {
	st, _ := p.sess.State()
	return p.sess.Kind == session.SSH && st == session.Connecting
}
