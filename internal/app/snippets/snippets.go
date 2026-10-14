package snippets

import (
	"image"
	"strings"

	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/widget"

	"github.com/plattyssh/platty/internal/app/core"
	"github.com/plattyssh/platty/internal/app/dialogs"
	"github.com/plattyssh/platty/internal/ui/icons"
	"github.com/plattyssh/platty/internal/ui/kit"
	"github.com/plattyssh/platty/internal/vault"
)

const (
	snipMenuRun = iota
	snipMenuPaste
	snipMenuSep1
	snipMenuEdit
	snipMenuSep2
	snipMenuDelete
)

type Panel struct {
	h core.Host

	add      widget.Clickable
	emptyAdd widget.Clickable
	list     kit.ScrollList
	menu     kit.Menu
	rows     map[string]*kit.Row
	run      map[string]*widget.Clickable
	snippets []vault.Snippet
	menuSnip vault.Snippet
}

func (p *Panel) Title() string     { return "Snippets" }
func (p *Panel) Icon() *icons.Icon { return icons.CodeOutline }
func (p *Panel) Side() core.Side   { return core.Right }
func (p *Panel) Shortcut() string  { return "Alt+4" }

func (p *Panel) Refresh() {
	a := p.h
	p.snippets = a.Vault().Snapshot().Snippets
}

func (p *Panel) Actions(gtx C) D {
	a := p.h

	if p.add.Clicked(gtx) {
		a.OpenIn(p, NewEditor(a, vault.Snippet{}))
	}

	return a.Kit().IconButtonSized(gtx, &p.add, icons.Add, false, 26, 18, a.Kit().Th.TextMuted)
}

func (p *Panel) Body(gtx C) D {
	a := p.h
	k := a.Kit()
	th := a.Kit().Th

	if p.rows == nil {
		p.rows = map[string]*kit.Row{}
		p.run = map[string]*widget.Clickable{}
	}

	p.handleMenu(a)

	if len(p.snippets) == 0 {
		if p.emptyAdd.Clicked(gtx) {
			a.OpenIn(p, NewEditor(a, vault.Snippet{}))
		}

		return a.Kit().EmptyState(gtx, icons.Snippet, "No snippets",
			"Save commands you run often and launch them in any session with one click.",
			func(gtx C) D { return k.Button(gtx, &p.emptyAdd, kit.Primary, icons.Add, "New snippet") },
		)
	}

	dims := k.List(gtx, &p.list, len(p.snippets), func(gtx C, i int) D {
		s := p.snippets[i]
		r := p.rows[s.ID]

		if r == nil {
			r = &kit.Row{}
			p.rows[s.ID] = r
			p.run[s.ID] = &widget.Clickable{}
		}

		run := p.run[s.ID]

		if run.Clicked(gtx) {
			a.RunSnippet(s, true)
		}

		ev := r.Update(gtx)

		if ev.DoubleClick {
			a.RunSnippet(s, true)
		}

		if ev.Context {
			p.menuSnip = s
			p.menu.Open([]kit.MenuItem{
				snipMenuRun:    {Label: "Run", Icon: icons.Play},
				snipMenuPaste:  {Label: "Paste without running", Icon: icons.Paste},
				snipMenuSep1:   kit.Separator(),
				snipMenuEdit:   {Label: "Edit…", Icon: icons.Edit},
				snipMenuSep2:   kit.Separator(),
				snipMenuDelete: {Label: "Delete", Icon: icons.Delete, Danger: true},
			}, ev.At.Add(image.Pt(0, gtx.Dp(4))))
		}

		return k.Row(gtx, r, kit.RowOpts{Height: 62}, func(gtx C) D {
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx C) D { return icons.Snippet.Layout(gtx, 18, th.TextMuted) }),
				layout.Rigid(kit.Gap(10)),
				layout.Flexed(1, func(gtx C) D {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx C) D { return k.Label(gtx, kit.TextStyle{Weight: font.Medium}, s.Label) }),
						layout.Rigid(kit.Gap(4)),
						layout.Rigid(func(gtx C) D {
							preview := strings.ReplaceAll(s.Command, "\n", " · ")
							return k.Label(gtx, kit.TextStyle{Size: th.TextSizeSmall - 1, Color: th.TextFaint, Mono: true}, preview)
						}),
					)
				}),
				layout.Rigid(func(gtx C) D {
					if !r.Hovered() && !run.Hovered() {
						return D{}
					}

					return k.IconButtonSized(gtx, run, icons.Play, false, 22, 16, th.TextMuted)
				}),
			)
		})
	})

	k.Menu(gtx, &p.menu)
	return dims
}

func (p *Panel) handleMenu(a core.Host) {
	i, ok := p.menu.Chosen()

	if !ok {
		return
	}

	s := p.menuSnip

	switch i {
	case snipMenuRun:
		a.RunSnippet(s, true)
	case snipMenuPaste:
		a.RunSnippet(s, false)
	case snipMenuEdit:
		a.OpenIn(p, NewEditor(a, s))
	case snipMenuDelete:
		a.Open(dialogs.NewConfirm(a.Kit(), "Delete snippet", "Delete “"+s.Label+"”?", "Delete", func() {
			a.Vault().DeleteSnippet(s.ID)
		}))
	}
}

func NewEditor(a core.Host, s vault.Snippet) *dialogs.Form {
	title := "New snippet"

	if s.ID != "" {
		title = "Edit snippet"
	}

	cmd := dialogs.NewField("Command", "sudo systemctl restart nginx", s.Command)
	cmd.Multi = true
	cmd.Mono = true

	f := dialogs.NewForm(a.Kit(), title, "Save", dialogs.NewField("Name", "Restart nginx", s.Label), cmd)
	f.Width = 520
	f.OnOK = func(v []string) bool {
		s.Label = strings.TrimSpace(v[0])
		s.Command = strings.TrimSpace(v[1])

		if s.Command == "" {
			return false
		}

		if s.Label == "" {
			s.Label = strings.SplitN(s.Command, "\n", 2)[0]
		}

		a.Vault().SaveSnippet(s)
		return true
	}
	return f
}

func New(h core.Host) *Panel {
	return &Panel{h: h}
}

type (
	C = core.C
	D = core.D
)
