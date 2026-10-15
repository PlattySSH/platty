package welcome

import (
	"image"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/plattyssh/platty/internal/app/core"
	"github.com/plattyssh/platty/internal/app/hosts"
	"github.com/plattyssh/platty/internal/ui/icons"
	"github.com/plattyssh/platty/internal/ui/kit"
	"github.com/plattyssh/platty/internal/ui/retro"
	"github.com/plattyssh/platty/internal/ui/sol"
	"github.com/plattyssh/platty/internal/vault"
)

type Screen struct {
	h      core.Host
	Hidden bool
	Art    retro.Computer

	mascot  sol.Sol
	newTerm widget.Clickable
	newHost widget.Clickable
	import_ widget.Clickable
	recent  map[string]*kit.Row

	hosts    []vault.Host
	hostsRev uint64
	hostsOK  bool
}

func (w *Screen) recentHosts(v *vault.Vault) []vault.Host {
	if rev := v.Revision(); !w.hostsOK || rev != w.hostsRev {
		w.hosts, w.hostsRev, w.hostsOK = RecentHosts(v, 5), rev, true
	}

	return w.hosts
}

func RecentHosts(v *vault.Vault, n int) []vault.Host {
	hosts := v.Snapshot().Hosts
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].LastUsed.After(hosts[j].LastUsed) })

	var out []vault.Host

	for _, h := range hosts {
		if h.LastUsed.IsZero() && len(out) >= 2 {
			continue
		}

		out = append(out, h)

		if len(out) == n {
			break
		}
	}

	return out
}

const welcomeArtMin = unit.Dp(860)

func (w *Screen) Layout(gtx C) D {
	a := w.h

	if w.newTerm.Clicked(gtx) {
		a.NewLocalTab(a.DefaultProfile())
	}

	if w.newHost.Clicked(gtx) {
		a.OpenDocked(core.Right, hosts.NewEditor(a, hosts.Defaults()))
	}

	if w.import_.Clicked(gtx) {
		a.ImportSSHConfig()
	}

	recent := w.recentHosts(a.Vault())
	command := "ssh user@server"

	if len(recent) > 0 {
		command = "ssh " + recent[0].Target()
	}

	side := gtx.Constraints.Max.X >= gtx.Dp(welcomeArtMin)
	return layout.Center.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min = image.Point{}
		children := []layout.FlexChild{
			layout.Rigid(func(gtx C) D {
				gtx.Constraints.Max.X = min(gtx.Constraints.Max.X, gtx.Dp(400))
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				return w.intro(a, gtx, recent)
			}),
		}

		if side {
			children = append(children,
				layout.Rigid(kit.Gap(56)),
				layout.Rigid(func(gtx C) D { return w.Art.Layout(gtx, a.Kit(), command) }),
			)
		}

		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
	})
}

func (w *Screen) intro(a core.Host, gtx C, recent []vault.Host) D {
	k := a.Kit()
	th := a.Kit().Th

	children := []layout.FlexChild{
		layout.Rigid(func(gtx C) D {
			w.mascot.Follow = true
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D { return w.mascot.Layout(gtx, a.Kit(), 68) }),
				layout.Rigid(kit.Gap(14)),
				layout.Flexed(1, func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx C) D {
							return k.Label(gtx, kit.TextStyle{Size: th.TextSizeTitle, Weight: font.SemiBold}, greeting(time.Now(), userName()))
						}),
						layout.Rigid(kit.Gap(4)),
						layout.Rigid(func(gtx C) D {
							return k.Label(gtx, kit.TextStyle{Color: th.TextMuted, MaxLines: 2}, summary(a))
						}),
					)
				}),
			)
		}),
		layout.Rigid(kit.Gap(22)),
		layout.Rigid(func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D { return k.Button(gtx, &w.newTerm, kit.Primary, icons.Terminal, "New terminal") }),
				layout.Rigid(kit.Gap(8)),
				layout.Rigid(func(gtx C) D { return k.Button(gtx, &w.newHost, kit.Secondary, icons.PlusOutline, "New host") }),
			)
		}),
	}

	if len(recent) == 0 {
		children = append(children,
			layout.Rigid(kit.Gap(14)),
			layout.Rigid(func(gtx C) D { return k.Link(gtx, &w.import_, "Import hosts from ~/.ssh/config") }),
		)
	} else {
		children = append(children,
			layout.Rigid(kit.Gap(34)),
			layout.Rigid(func(gtx C) D {
				return layout.Inset{Bottom: 6}.Layout(gtx, func(gtx C) D {
					return k.Label(gtx, kit.TextStyle{Size: th.TextSizeSmall, Color: th.TextFaint}, "Recent")
				})
			}),
		)

		for _, h := range recent {
			children = append(children, layout.Rigid(func(gtx C) D { return w.recentRow(a, gtx, h) }))
		}
	}

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
}

func (w *Screen) recentRow(a core.Host, gtx C, h vault.Host) D {
	k := a.Kit()
	th := a.Kit().Th

	if w.recent == nil {
		w.recent = map[string]*kit.Row{}
	}

	r := w.recent[h.ID]

	if r == nil {
		r = &kit.Row{}
		w.recent[h.ID] = r
	}

	if ev := r.Update(gtx); ev.Click {
		a.Connect(h)
	}

	return layout.Inset{Left: -12}.Layout(gtx, func(gtx C) D {
		hv := k.Hover(gtx, kit.Key(r, 9), r.Hovered())
		fg := kit.Lerp(th.TextMuted, th.Text, hv)
		return k.Row(gtx, r, kit.RowOpts{Height: 36}, func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D { return icons.Terminal.Layout(gtx, 16, fg) }),
				layout.Rigid(kit.Gap(10)),
				layout.Rigid(func(gtx C) D { return k.Label(gtx, kit.TextStyle{Color: kit.Lerp(fg, th.Text, 0.6)}, h.Title()) }),
				layout.Rigid(kit.Gap(10)),
				layout.Flexed(1, func(gtx C) D {
					return k.Label(gtx, kit.TextStyle{Size: th.TextSizeSmall, Color: th.TextFaint, MaxLines: 1}, h.Target())
				}),
				layout.Rigid(func(gtx C) D {
					if h.LastUsed.IsZero() {
						return D{}
					}

					return k.Label(gtx, kit.TextStyle{Size: th.TextSizeSmall, Color: th.TextFaint}, ago(time.Since(h.LastUsed)))
				}),
			)
		})
	})
}

func greeting(t time.Time, name string) string {
	var s string

	switch h := t.Hour(); {
	case h < 6:
		if name != "" {
			return "Up late, " + name + "?"
		}

		return "Up late?"
	case h < 13:
		s = "Good morning"
	case h < 21:
		s = "Good afternoon"
	default:
		s = "Good evening"
	}

	if name != "" {
		s += ", " + name
	}

	return s
}

func userName() string {
	n := strings.TrimSpace(os.Getenv("USERNAME"))

	if n == "" || strings.EqualFold(n, "user") || strings.EqualFold(n, "usuario") {
		return ""
	}

	r := []rune(n)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

func summary(a core.Host) string {
	d := a.Vault().Snapshot()
	hosts, folders := len(d.Hosts), len(d.AllFolders())

	if hosts == 0 {
		return "Add your first server or open a local terminal."
	}

	s := plural(hosts, "saved server", "saved servers")

	if folders > 0 {
		s += " in " + plural(folders, "folder", "folders")
	}

	if n := a.Tunnels().Active(); n > 0 {
		s += " · " + plural(n, "active tunnel", "active tunnels")
	}

	return s + "."
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}

	return strconv.Itoa(n) + " " + many
}

func ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + " min ago"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d.Hours())) + " h ago"
	case d < 48*time.Hour:
		return "yesterday"
	default:
		return strconv.Itoa(int(d.Hours()/24)) + " days ago"
	}
}

func New(h core.Host) *Screen {
	return &Screen{h: h}
}

type (
	C = core.C
	D = core.D
)
