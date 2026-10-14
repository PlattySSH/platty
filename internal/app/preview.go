package app

import (
	"time"

	gioapp "gioui.org/app"
	"gioui.org/x/explorer"

	"github.com/plattyssh/platty/internal/app/core"
	"github.com/plattyssh/platty/internal/app/hosts"
	"github.com/plattyssh/platty/internal/app/palette"
	"github.com/plattyssh/platty/internal/app/settings"
	"github.com/plattyssh/platty/internal/app/welcome"
	"github.com/plattyssh/platty/internal/config"
	"github.com/plattyssh/platty/internal/fonts"
	"github.com/plattyssh/platty/internal/platform/shell"
	"github.com/plattyssh/platty/internal/session"
	"github.com/plattyssh/platty/internal/sshx"
	"github.com/plattyssh/platty/internal/transfer"
	"github.com/plattyssh/platty/internal/ui/kit"
	"github.com/plattyssh/platty/internal/ui/theme"
	"github.com/plattyssh/platty/internal/vault"
)

type Preview struct {
	*App
}

func NewPreview(v *vault.Vault, themeName string) *Preview {
	s := config.Defaults()
	s.Theme = themeName

	a := &App{
		win:      new(gioapp.Window),
		settings: s,
		vault:    v,
		profiles: shell.Detect(),
	}
	a.shaper, a.face = fonts.Load()
	a.glyphs, _ = fonts.Glyphs(a.settings.TerminalFont)
	a.kit = kit.New(theme.New(themeName), a.shaper)
	a.explorer = explorer.NewExplorer(a.win)
	a.tunnels = a.newTunnels()
	a.transfers = transfer.New(a.Invalidate)
	a.tools.init(a)
	a.started = time.Now()
	a.clips = make(chan string, 4)
	a.restored = true
	a.welcome = welcome.New(a)
	a.palette = palette.New(a.kit, a.paletteEntries, a.paletteQuick, a.ws.focusActive)
	a.welcome.Art.OnRun(a.runTyped)
	a.toasts.OnChange(a.Invalidate)
	return &Preview{App: a}
}

func (p *Preview) Frame(gtx C) {
	p.frame(gtx)
}

func (p *Preview) OpenTool(i int) {
	p.tools.open[p.tools.list[i].Side()] = i
}

func (p *Preview) CloseTools() {
	p.tools.open = [2]int{-1, -1}
}

func (p *Preview) RunLocal(title string, command ...string) {
	s := session.New(session.Options{
		Kind:     session.Local,
		Command:  command,
		Cols:     120,
		Rows:     32,
		Title:    title,
		OnChange: func() {},
	})
	pane := p.newPane(s)
	p.ws.addTab(pane)
}

func (p *Preview) Split(vertical bool, title string, command ...string) {
	s := session.New(session.Options{Kind: session.Local, Command: command, Cols: 80, Rows: 24, Title: title, OnChange: func() {}})
	p.splitWith(p.newPane(s), vertical)
}

func (p *Preview) Wait(d time.Duration) {
	time.Sleep(d)
	p.ws.reap(p.App)
}

func (p *Preview) OpenPalette(q string) {
	p.palette.Show()
	p.palette.SetQuery(q)
}

func (p *Preview) ShowHostEditor(h vault.Host) {
	p.OpenDocked(core.Right, hosts.NewEditor(p.App, h))
}

func (p *Preview) ShowSettings(page int) {
	d := settings.New(p.App)
	d.SetPage(page)
	p.Open(d)
}

func (p *Preview) ConnectTrusted(h vault.Host, trust bool) {
	p.autoTrust = trust
	p.App.Connect(h)
}

func (p *Preview) Toast(title, msg string) {
	p.toasts.Success(title, msg)
}

func (p *Preview) Close() {
	p.ws.closeAll()
}

func (p *Preview) FakeStats() {
	pn := p.ws.activePane()

	if pn == nil {
		return
	}

	pn.stats.mu.Lock()
	pn.stats.running = true
	pn.stats.have = true
	pn.stats.st = sshx.Stats{CPU: 0.23, Cores: 4, MemUsed: 1_700_000_000, MemTotal: 3_900_000_000,
		DiskUsed: 21_000_000_000, DiskTotal: 40_000_000_000, DiskAvail: 19_000_000_000, Load1: 0.42, Load5: 0.30, Load15: 0.21,
		Uptime: 76 * time.Hour, RxRate: 12_400, TxRate: 3_100, RxTotal: 8_200_000_000, TxTotal: 1_900_000_000,
		SwapUsed: 120_000_000, SwapTotal: 2_000_000_000, OS: "Ubuntu 24.04.1 LTS", Kernel: "Linux 6.8.0-45-generic", Hostname: "osiris"}
	pn.stats.mu.Unlock()
}
