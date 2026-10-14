package app

import (
	"sync"
	"time"

	"github.com/plattyssh/platty/internal/session"
	"github.com/plattyssh/platty/internal/sshx"
)

const statsEvery = 3 * time.Second

type paneStats struct {
	mu      sync.Mutex
	st      sshx.Stats
	have    bool
	err     error
	running bool
}

func (s *paneStats) get() (sshx.Stats, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.st, s.have, s.err
}

func (a *App) watchStats(p *Pane) {
	ps := &p.stats
	ps.mu.Lock()

	if ps.running {
		ps.mu.Unlock()
		return
	}

	ps.running = true
	ps.mu.Unlock()

	go func() {
		var sm sshx.Sampler
		defer func() {
			ps.mu.Lock()
			ps.running = false
			ps.mu.Unlock()
		}()

		for {
			c := p.sess.Client()

			if c == nil {
				return
			}

			st, err := sm.Sample(c)
			ps.mu.Lock()

			if err == nil {
				ps.st, ps.have = st, true
			}

			ps.err = err
			ps.mu.Unlock()
			a.Invalidate()

			if err == sshx.ErrNoStats {
				return
			}

			time.Sleep(statsEvery)
		}
	}()
}

func (c *chrome) statusBar(a *App, gtx C) D {
	p := a.ws.activePane()

	if p == nil || p.sess.Kind != session.SSH {
		return D{}
	}

	if st, _ := p.sess.State(); st == session.Connected {
		a.watchStats(p)
	}

	s, have, err := p.stats.get()
	return c.metrics.Layout(gtx, a.kit, s, have, err)
}
