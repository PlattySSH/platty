package app

import (
	"image"
	"image/color"
	"strconv"

	"gioui.org/f32"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/plattyssh/platty/internal/app/core"
	"github.com/plattyssh/platty/internal/app/settings"
	"github.com/plattyssh/platty/internal/platform/titlebar"
	"github.com/plattyssh/platty/internal/ui/icons"
	"github.com/plattyssh/platty/internal/ui/kit"
	"github.com/plattyssh/platty/internal/ui/metrics"
	"github.com/plattyssh/platty/internal/ui/sol"
)

const (
	sidebarWidth      = unit.Dp(260)
	sidebarFormWidth  = unit.Dp(330)
	sidebarTopHeight  = unit.Dp(68)
	headerHeight      = unit.Dp(52)
	cardMargin        = unit.Dp(8)
	windowButtonWidth = unit.Dp(38)
	navRowHeight      = unit.Dp(32)
)

var closeHover = color.NRGBA{R: 0xc4, G: 0x2b, B: 0x1c, A: 0xff}

type chrome struct {
	logo       sol.Sol
	sideToggle widget.Clickable
	home       kit.Row
	newTerm    kit.Row
	profiles   widget.Clickable
	settings   kit.Row
	upgrade    kit.Row

	search     widget.Clickable
	newTab     widget.Clickable
	moreTabs   widget.Clickable
	metrics    metrics.Bar
	tabMenu    kit.Menu
	hiddenTabs []int
	tabCtx     kit.Menu
	ctxTab     *Tab
	renaming   *Tab
	rename     kit.InlineEdit
	tabPos     map[*Tab][2]int
	splitR     widget.Clickable
	splitD     widget.Clickable
	toolPanel  widget.Clickable
	minimize   widget.Clickable
	maximize   widget.Clickable
	close      widget.Clickable
}

func (c *chrome) shell(a *App, gtx C) D {
	th := a.th()
	kit.Fill(gtx, th.Window)

	return layout.Flex{}.Layout(gtx,
		layout.Rigid(func(gtx C) D { return c.sidebar(a, gtx) }),
		layout.Flexed(1, func(gtx C) D { return c.main(a, gtx) }),
	)
}

func (c *chrome) sidebar(a *App, gtx C) D {
	k := a.kit

	a.ws.update(a, gtx)
	c.handleNav(a, gtx)

	if c.sideToggle.Clicked(gtx) {
		a.tools.toggleSide(a, core.Left)
	}

	full := gtx.Dp(sidebarWidth)

	if a.docked[core.Left] != nil {
		full = gtx.Dp(sidebarFormWidth)
	}

	target := float32(0)

	if a.tools.open[core.Left] >= 0 {
		target = float32(full)
	}

	width := int(k.Animated(gtx, kit.Key(&c.sideToggle, 0), target, kit.Slow))

	if target > 0 && width >= gtx.Dp(sidebarWidth) {
		full = width
	}

	if width <= 0 {
		return D{}
	}

	height := gtx.Constraints.Max.Y
	defer clip.Rect{Max: image.Pt(width, height)}.Push(gtx.Ops).Pop()
	defer op.Offset(image.Pt(width-full, 0)).Push(gtx.Ops).Pop()

	gtx.Constraints = layout.Exact(image.Pt(full, height))
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx C) D { return c.sidebarTop(a, gtx) }),
		layout.Rigid(func(gtx C) D {
			return layout.Inset{Left: 8, Right: 8}.Layout(gtx, func(gtx C) D {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx C) D {
						return layout.Inset{Left: 6, Right: 6, Bottom: 8}.Layout(gtx, func(gtx C) D { return c.searchBar(a, gtx) })
					}),
					layout.Rigid(func(gtx C) D {
						return c.navRow(a, gtx, &c.home, icons.HomeOutline, "Home", a.ws.current() == nil, nil)
					}),
					layout.Rigid(func(gtx C) D { return c.toolRow(a, gtx, a.tools.indexOf(a.tools.sftp)) }),
					layout.Rigid(func(gtx C) D { return c.newTermRow(a, gtx) }),
				)
			})
		}),
		layout.Flexed(1, func(gtx C) D {
			if a.docked[core.Left] != nil {
				return layout.Inset{Left: 8, Right: 8, Top: 12}.Layout(gtx, func(gtx C) D {
					a.layoutDocked(gtx, core.Left)
					return D{Size: gtx.Constraints.Max}
				})
			}

			return a.tools.hosts.Sidebar(gtx)
		}),
		layout.Rigid(func(gtx C) D {
			if a.docked[core.Left] != nil {
				return D{}
			}

			return c.sidebarTools(a, gtx)
		}),
	)

	return D{Size: image.Pt(width, height)}
}

func (c *chrome) sidebarTop(a *App, gtx C) D {
	c.logo.Still = true
	c.logo.Posed, c.logo.Tilt, c.logo.Look = true, 0.22, f32.Pt(1, -1)
	th := a.th()
	k := a.kit
	size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(sidebarTopHeight))
	gtx.Constraints = layout.Exact(size)
	moveArea(gtx, size)

	layout.Inset{Left: 10, Right: 10, Top: headerHeight - tabHeight}.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.Y = gtx.Dp(tabHeight)
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(trafficLightsGap),
			layout.Rigid(func(gtx C) D { return c.logo.Layout(gtx, a.kit, 46) }),
			layout.Flexed(1, func(gtx C) D { return D{Size: image.Pt(gtx.Constraints.Min.X, 0)} }),
			layout.Rigid(func(gtx C) D {
				return k.IconButtonSized(gtx, &c.sideToggle, icons.SidebarLeft, false, 30, 18, th.TextFaint)
			}),
		)
	})
	return D{Size: size}
}

func (c *chrome) handleNav(a *App, gtx C) {
	if c.search.Clicked(gtx) {
		a.palette.Show()
	}

	if ev := c.home.Update(gtx); ev.Click {
		a.ws.active = -1
	}

	if ev := c.newTerm.Update(gtx); ev.Click && !c.profiles.Hovered() {
		a.NewLocalTab(a.DefaultProfile())
	}

	if c.profiles.Clicked(gtx) {
		a.ws.menu.Width = 280
		a.ws.menu.Open(a.ws.newTabMenu(a), image.Pt(gtx.Dp(8), gtx.Dp(navRowHeight)))
	}

	if ev := c.upgrade.Update(gtx); ev.Click {
		switch a.Update().State {
		case core.UpdateAvailable, core.UpdateFailed:
			a.DownloadUpdate()
		case core.UpdateReady:
			a.RestartToUpdate()
		}
	}

	if ev := c.settings.Update(gtx); ev.Click {
		a.Open(settings.New(a))
	}

	for i := 1; i < len(a.tools.list); i++ {
		if ev := a.tools.rows[i].Update(gtx); ev.Click {
			a.docked[core.Right] = nil

			if a.tools.open[core.Right] == i {
				a.tools.open[core.Right] = -1
				a.ws.focusActive()
			} else {
				a.tools.open[core.Right] = i
			}
		}
	}
}

func (c *chrome) navRow(a *App, gtx C, r *kit.Row, ic *icons.Icon, label string, active bool, trailing layout.Widget) D {
	return c.navRowStyled(a, gtx, r, ic, label, active, active, trailing)
}

func (c *chrome) navRowStyled(a *App, gtx C, r *kit.Row, ic *icons.Icon, label string, active, lit bool, trailing layout.Widget) D {
	k := a.kit
	th := a.th()
	hv := k.Hover(gtx, kit.Key(r, 9), r.Hovered())
	fg := kit.Lerp(kit.Lerp(th.TextMuted, th.Text, hv), th.Text, kit.B2F(lit))

	return k.Row(gtx, r, kit.RowOpts{Selected: active, Focused: true}, func(gtx C) D {
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx C) D { return ic.Layout(gtx, 18, fg) }),
			layout.Rigid(kit.Gap(12)),
			layout.Flexed(1, func(gtx C) D { return k.Label(gtx, kit.TextStyle{Color: fg}, label) }),
			layout.Rigid(func(gtx C) D {
				if trailing == nil {
					return D{}
				}

				return trailing(gtx)
			}),
		)
	})
}

func (c *chrome) newTermRow(a *App, gtx C) D {
	k := a.kit
	th := a.th()

	dims := c.navRow(a, gtx, &c.newTerm, icons.PlusOutline, "New terminal", false, func(gtx C) D {
		return k.IconButtonSized(gtx, &c.profiles, icons.ExpandMore, a.ws.menu.IsOpen(), 24, 16, th.TextFaint)
	})
	k.Menu(gtx, &a.ws.menu)
	return dims
}

func (c *chrome) sidebarTools(a *App, gtx C) D {
	return layout.Inset{Left: 8, Right: 8, Top: 8, Bottom: 10}.Layout(gtx, func(gtx C) D {
		children := make([]layout.FlexChild, 0, len(a.tools.list)+1)

		for i := 1; i < len(a.tools.list); i++ {
			if a.tools.list[i] == core.Tool(a.tools.sftp) {
				continue
			}

			children = append(children, layout.Rigid(func(gtx C) D { return c.toolRow(a, gtx, i) }))
		}

		if label := upgradeLabel(a.Update()); label != "" {
			children = append(children, layout.Rigid(func(gtx C) D {
				return c.navRowStyled(a, gtx, &c.upgrade, icons.Download, label, false, true, nil)
			}))
		}

		children = append(children, layout.Rigid(func(gtx C) D {
			return c.navRow(a, gtx, &c.settings, icons.SlidersOutline, "Settings", false, nil)
		}))
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

func (c *chrome) card(a *App, gtx C) D {
	th := a.th()
	size := gtx.Constraints.Max
	r := image.Rectangle{Max: size}
	radius := gtx.Dp(th.RadiusLarge)

	kit.RRect(gtx, r, radius, th.Editor)
	content := clip.UniformRRect(r, radius).Push(gtx.Ops)
	gtx.Constraints = layout.Exact(size)
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Flexed(1, func(gtx C) D { return a.ws.layout(a, gtx) }),
		layout.Rigid(func(gtx C) D { return c.statusBar(a, gtx) }),
	)
	content.Pop()
	kit.Border(gtx, r, radius, 1, th.Border)
	return D{Size: size}
}

func (c *chrome) header(a *App, gtx C) D {
	th := a.th()
	k := a.kit
	c.handleHeader(a, gtx)

	size := image.Pt(gtx.Constraints.Max.X, gtx.Dp(headerHeight))
	gtx.Constraints = layout.Exact(size)
	moveArea(gtx, size)

	pane := a.ws.activePane()
	icon := func(clk *widget.Clickable, ic *icons.Icon, active bool) layout.FlexChild {
		return layout.Rigid(func(gtx C) D { return k.IconButtonSized(gtx, clk, ic, active, 32, 18, th.TextMuted) })
	}

	children := []layout.FlexChild{}

	if a.tools.open[core.Left] < 0 {
		children = append(children, layout.Rigid(trafficLightsGap), icon(&c.sideToggle, icons.SidebarLeft, false), layout.Rigid(kit.Gap(8)))
	}

	children = append(children,
		layout.Flexed(1, func(gtx C) D { return c.tabStrip(a, gtx) }),
		layout.Rigid(kit.Gap(8)),
	)

	if pane != nil {
		children = append(children, icon(&c.splitR, icons.SplitRight, false), icon(&c.splitD, icons.SplitDown, false))
	}

	children = append(children,
		icon(&c.toolPanel, icons.SidebarRight, a.tools.open[core.Right] >= 0),
		layout.Rigid(kit.Gap(8)),
		layout.Rigid(func(gtx C) D { return c.windowControls(a, gtx) }),
	)

	layout.Inset{Left: 12, Right: 8, Top: headerHeight - tabHeight}.Layout(gtx, func(gtx C) D {
		gtx.Constraints.Min.Y = gtx.Dp(tabHeight)
		return layout.Flex{Alignment: layout.Middle}.Layout(gtx, children...)
	})
	return D{Size: size}
}

func (c *chrome) handleHeader(a *App, gtx C) {
	if c.minimize.Clicked(gtx) {
		a.win.Perform(system.ActionMinimize)
	}

	if c.maximize.Clicked(gtx) {
		if a.maximized {
			a.win.Perform(system.ActionUnmaximize)
		} else {
			a.win.Perform(system.ActionMaximize)
		}
	}

	if c.close.Clicked(gtx) {
		a.win.Perform(system.ActionClose)
	}

	if c.splitR.Clicked(gtx) {
		a.ws.split(a, true)
	}

	if c.splitD.Clicked(gtx) {
		a.ws.split(a, false)
	}

	if c.toolPanel.Clicked(gtx) {
		a.tools.toggleSide(a, core.Right)
	}
}

func (c *chrome) searchBar(a *App, gtx C) D {
	k := a.kit
	th := a.th()

	return c.search.Layout(gtx, func(gtx C) D {
		w := gtx.Constraints.Max.X
		h := gtx.Dp(32)
		size := image.Pt(w, h)
		r := image.Rectangle{Max: size}
		radius := gtx.Dp(th.RadiusSmall)

		hv := k.Hover(gtx, kit.Key(&c.search, 0), c.search.Hovered())
		kit.RRect(gtx, r, radius, kit.Lerp(kit.Fade(th.Hover, 1.4), th.Selection, hv))

		gtx.Constraints = layout.Exact(size)
		layout.Inset{Left: 10, Right: 10}.Layout(gtx, func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D { return icons.Search.Layout(gtx, 16, th.TextMuted) }),
				layout.Rigid(kit.Gap(8)),
				layout.Flexed(1, func(gtx C) D {
					gtx.Constraints.Min.Y = h
					return layout.W.Layout(gtx, func(gtx C) D {
						return k.Label(gtx, kit.TextStyle{Color: th.TextMuted, MaxLines: 1}, "Search…")
					})
				}),
				layout.Rigid(func(gtx C) D {
					return k.Label(gtx, kit.TextStyle{Size: th.TextSizeSmall, Color: th.TextFaint}, "Ctrl+Shift+P")
				}),
			)
		})
		kit.Cursor(gtx, size, pointer.CursorPointer)
		return D{Size: size}
	})
}

func (c *chrome) windowButton(a *App, gtx C, clk *widget.Clickable, ic *icons.Icon, isClose bool) D {
	th := a.th()

	return clk.Layout(gtx, func(gtx C) D {
		size := image.Pt(gtx.Dp(windowButtonWidth), gtx.Dp(32))
		r := image.Rectangle{Max: size}
		radius := gtx.Dp(th.RadiusSmall)
		fg := th.TextMuted

		switch {
		case isClose && clk.Hovered():
			kit.RRect(gtx, r, radius, closeHover)
			fg = color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
		case clk.Hovered():
			kit.RRect(gtx, r, radius, th.Hover)
			fg = th.Text
		}

		gtx.Constraints = layout.Exact(size)
		layout.Center.Layout(gtx, func(gtx C) D { return ic.Layout(gtx, 16, fg) })
		return D{Size: size}
	})
}

func (c *chrome) toolRow(a *App, gtx C, i int) D {
	tw := a.tools.list[i]
	open := a.tools.open[core.Right] == i
	return c.navRowStyled(a, gtx, &a.tools.rows[i], tw.Icon(), tw.Title(), false, open, nil)
}

func trafficLightsGap(gtx C) D {
	return D{Size: image.Pt(gtx.Dp(titlebar.Width), 0)}
}

func (c *chrome) windowControls(a *App, gtx C) D {
	if titlebar.Native {
		return D{}
	}

	return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
		layout.Rigid(func(gtx C) D { return c.windowButton(a, gtx, &c.minimize, icons.Minimize, false) }),
		layout.Rigid(func(gtx C) D {
			ic := icons.Maximize

			if a.maximized {
				ic = icons.Restore
			}

			return c.windowButton(a, gtx, &c.maximize, ic, false)
		}),
		layout.Rigid(func(gtx C) D { return c.windowButton(a, gtx, &c.close, icons.WindowClose, true) }),
	)
}

func (c *chrome) main(a *App, gtx C) D {
	size := gtx.Constraints.Max
	hh := gtx.Dp(headerHeight)

	rec := op.Record(gtx.Ops)
	hg := gtx
	hg.Constraints = layout.Exact(image.Pt(size.X, hh))
	c.header(a, hg)
	header := rec.Stop()

	off := op.Offset(image.Pt(0, hh)).Push(gtx.Ops)
	bg := gtx
	bg.Constraints = layout.Exact(image.Pt(size.X, size.Y-hh))
	layout.Flex{}.Layout(bg,
		layout.Flexed(1, func(gtx C) D {
			in := layout.Inset{Bottom: cardMargin, Right: cardMargin}

			if a.tools.open[core.Left] < 0 {
				in.Left = cardMargin
			}

			if a.tools.open[core.Right] >= 0 {
				in.Right = 0
			}

			return in.Layout(gtx, func(gtx C) D { return c.card(a, gtx) })
		}),
		layout.Rigid(func(gtx C) D { return a.tools.panel(a, gtx, core.Right) }),
	)
	off.Pop()

	header.Add(gtx.Ops)
	return D{Size: size}
}

func upgradeLabel(info core.UpdateInfo) string {
	switch info.State {
	case core.UpdateAvailable:
		return "Update to " + info.Version
	case core.UpdateDownloading:
		return "Downloading update… " + strconv.Itoa(int(info.Progress*100)) + "%"
	case core.UpdateReady:
		return "Restart to update"
	case core.UpdateFailed:
		if info.Version != "" {
			return "Retry update"
		}
	}

	return ""
}
