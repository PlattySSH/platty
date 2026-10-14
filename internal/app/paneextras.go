package app

import (
	"fmt"
	"image"
	"io"
	"strings"
	"time"

	"gioui.org/io/clipboard"

	"github.com/plattyssh/platty/internal/ui/icons"
	"github.com/plattyssh/platty/internal/ui/kit"
	"github.com/plattyssh/platty/internal/ui/termview"
)

const (
	actionLog       = "log"
	actionBroadcast = "broadcast"
)

func (a *App) paneExtras(p *Pane) []termview.Extra {
	log := termview.Extra{Action: actionLog, Item: kit.MenuItem{Label: "Log to file…", Icon: icons.File}}

	if p.sess.Logging() {
		log.Item.Label = "Stop logging"
	}

	t := a.ws.tabOf(p)
	cast := termview.Extra{Action: actionBroadcast, Item: kit.MenuItem{Label: "Broadcast input to this tab", Icon: icons.ArrowsOutline}}

	switch {
	case t != nil && t.broadcast:
		cast.Item.Label = "Stop broadcasting"
	case t == nil || len(t.root.panes(nil)) < 2:
		cast.Item.Disabled = true
	}

	return []termview.Extra{log, cast}
}

func (a *App) toggleLog(p *Pane) {
	if p.sess.Logging() {
		p.sess.StopLog()
		a.toasts.Success("Logging stopped", p.sess.Title())
		return
	}

	name := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`\/:*?"<>|`, r) {
			return '-'
		}

		return r
	}, p.sess.Title())

	go func() {
		w, err := a.explorer.CreateFile(fmt.Sprintf("%s %s.log", name, time.Now().Format("2006-01-02 1504")))

		if err != nil {
			return
		}

		p.sess.StartLog(w)
		a.toasts.Info("Logging "+p.sess.Title(), "Everything this session prints is saved as plain text.")
		a.Invalidate()
	}()
}

func (w *workspace) tabOf(p *Pane) *Tab {
	for _, t := range w.tabs {
		for _, q := range t.root.panes(nil) {
			if q == p {
				return t
			}
		}
	}

	return nil
}

func (a *App) toggleBroadcast(p *Pane) {
	t := a.ws.tabOf(p)

	if t == nil {
		return
	}

	t.broadcast = !t.broadcast

	if !t.broadcast {
		for _, q := range t.root.panes(nil) {
			q.sess.SetMirror(nil)
			q.mirrored = false
		}

		return
	}

	a.toasts.Info("Broadcasting", "What you type goes to every pane in this tab.")
	syncBroadcast(t)
}

func syncBroadcast(t *Tab) {
	if !t.broadcast {
		return
	}

	for _, p := range t.root.panes(nil) {
		if p.mirrored {
			continue
		}

		p.mirrored = true
		p.sess.SetMirror(func(b []byte) {
			for _, q := range t.root.panes(nil) {
				if q != p {
					q.sess.SendOnly(b)
				}
			}
		})
	}
}

func broadcastBorder(gtx C, a *App, size image.Point) {
	kit.Border(gtx, image.Rectangle{Max: size}.Inset(1), 0, 2, kit.Fade(a.th().Danger, 0.8))
}

const maxServerCopy = 1 << 20

func (a *App) serverCopy(text string) {
	if len(text) > maxServerCopy {
		return
	}

	select {
	case a.clips <- text:
		a.Invalidate()
	default:
	}
}

func (a *App) applyServerCopies(gtx C) {
	for {
		select {
		case text := <-a.clips:
			gtx.Execute(clipboard.WriteCmd{Type: "application/text", Data: io.NopCloser(strings.NewReader(text))})
			a.toasts.Info("Copied from the server", fmt.Sprintf("A program in the terminal copied %d characters to your clipboard.", len([]rune(text))))
		default:
			return
		}
	}
}
