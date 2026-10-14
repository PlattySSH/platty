package theme

import (
	"image/color"

	"gioui.org/unit"
	"github.com/plattyssh/vt"
)

const (
	Carbon   = "carbon"
	Graphite = "graphite"
	Midnight = "midnight"
	Paper    = "paper"
)

var Names = []string{Carbon, Graphite, Midnight, Paper}

var Labels = []string{"Carbon", "Graphite", "Midnight", "Paper"}

type Metrics struct {
	ToolbarHeight unit.Dp
	StatusHeight  unit.Dp
	TabHeight     unit.Dp
	RowHeight     unit.Dp
	FieldHeight   unit.Dp
	Radius        unit.Dp
	RadiusSmall   unit.Dp
	RadiusLarge   unit.Dp
	Gap           unit.Dp

	TextSize      unit.Sp
	TextSizeSmall unit.Sp
	TextSizeLarge unit.Sp
	TextSizeTitle unit.Sp
}

type Theme struct {
	Name string
	Dark bool
	Metrics

	Frame       color.NRGBA
	FrameText   color.NRGBA
	FrameMuted  color.NRGBA
	FramePanel  color.NRGBA
	FrameHover  color.NRGBA
	FrameBorder color.NRGBA

	Window    color.NRGBA
	Panel     color.NRGBA
	Editor    color.NRGBA
	Popup     color.NRGBA
	Field     color.NRGBA
	Border    color.NRGBA
	Divider   color.NRGBA
	FieldLine color.NRGBA
	Hover     color.NRGBA
	Pressed   color.NRGBA
	Selection color.NRGBA
	Inactive  color.NRGBA
	Scrim     color.NRGBA
	Dim       color.NRGBA

	Text      color.NRGBA
	TextMuted color.NRGBA
	TextFaint color.NRGBA
	OnAccent  color.NRGBA

	Accent      color.NRGBA
	AccentHover color.NRGBA
	AccentSoft  color.NRGBA
	Success     color.NRGBA
	Warning     color.NRGBA
	Danger      color.NRGBA

	Foreground  color.NRGBA
	Background  color.NRGBA
	Cursor      color.NRGBA
	CursorText  color.NRGBA
	TermSelect  color.NRGBA
	SearchMatch color.NRGBA
	ANSI        [16]color.NRGBA

	Tags []color.NRGBA

	palette [256]color.NRGBA
}

func hex(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}

func alpha(c color.NRGBA, a uint8) color.NRGBA {
	c.A = a
	return c
}

var metrics = Metrics{
	ToolbarHeight: 48,
	StatusHeight:  0,
	TabHeight:     34,
	RowHeight:     32,
	FieldHeight:   36,
	Radius:        8,
	RadiusSmall:   6,
	RadiusLarge:   12,
	Gap:           8,
	TextSize:      14,
	TextSizeSmall: 12,
	TextSizeLarge: 16,
	TextSizeTitle: 24,
}

type spec struct {
	name               string
	dark               bool
	canvas, surface    uint32
	raised, popup      uint32
	text, muted, faint uint32
	danger             uint32
	ansi               [16]uint32
}

var ansiDark = [16]uint32{
	0x1c1c1c, 0xe0625a, 0x93bf8f, 0xd6bb82,
	0x87a4c8, 0xb69ac8, 0x8cc2ba, 0xbdbdbd,
	0x525252, 0xe39598, 0xaad2a6, 0xe3cc9c,
	0xa1bad9, 0xcab3d9, 0xa6d4cd, 0xf2f2f2,
}

var ansiLight = [16]uint32{
	0x1f1f1f, 0xb5403f, 0x3f7a3a, 0x8a6a1f,
	0x3a5f94, 0x7a4f94, 0x2f7a72, 0x8a8a8a,
	0x5c5c5c, 0xc95553, 0x4f8f4a, 0xa07d2c,
	0x4d72a8, 0x8d61a8, 0x3c8e85, 0xa3a3a3,
}

var specs = map[string]spec{
	Carbon: {
		name: Carbon, dark: true,
		canvas: 0x0e0d0c, surface: 0x171615, raised: 0x25231f, popup: 0x1d1c1a,
		text: 0xece7e1, muted: 0x9b958d, faint: 0x67625b,
		danger: 0xe0625a, ansi: ansiDark,
	},
	Graphite: {
		name: Graphite, dark: true,
		canvas: 0x111111, surface: 0x1a1a1a, raised: 0x282828, popup: 0x202020,
		text: 0xf2f2f2, muted: 0x9a9a9a, faint: 0x666666,
		danger: 0xe0625a, ansi: ansiDark,
	},
	Midnight: {
		name: Midnight, dark: true,
		canvas: 0x0b0d12, surface: 0x12151c, raised: 0x1d212b, popup: 0x171a22,
		text: 0xe6e9ef, muted: 0x8b919c, faint: 0x575d68,
		danger: 0xe0625a, ansi: ansiDark,
	},
	Paper: {
		name: Paper, dark: false,
		canvas: 0xf1efeb, surface: 0xfdfcfa, raised: 0xe8e5df, popup: 0xffffff,
		text: 0x1c1a17, muted: 0x6f6a63, faint: 0xa39e96,
		danger: 0xc0392b, ansi: ansiLight,
	},
}

func New(name string) *Theme {
	sp, ok := specs[name]

	if !ok {
		sp = specs[Carbon]
	}

	surface, text := hex(sp.surface), hex(sp.text)
	fg := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

	if !sp.dark {
		fg = color.NRGBA{A: 0xff}
	}

	line := func(f float32) color.NRGBA { return Mix(surface, fg, f) }

	t := &Theme{
		Name:    sp.name,
		Dark:    sp.dark,
		Metrics: metrics,

		Window:    hex(sp.canvas),
		Panel:     surface,
		Editor:    surface,
		Popup:     hex(sp.popup),
		Field:     surface,
		Border:    line(0.11),
		Divider:   line(0.11),
		FieldLine: line(0.14),
		Hover:     line(0.04),
		Pressed:   line(0.08),
		Selection: hex(sp.raised),
		Inactive:  hex(sp.raised),
		Scrim:     alpha(hex(0x000000), 0x99),
		Dim:       alpha(surface, 0x66),

		Text:      text,
		TextMuted: hex(sp.muted),
		TextFaint: hex(sp.faint),
		OnAccent:  surface,

		Accent:      text,
		AccentHover: fg,
		AccentSoft:  alpha(fg, 0x14),
		Danger:      hex(sp.danger),

		Foreground:  text,
		Background:  surface,
		Cursor:      text,
		CursorText:  surface,
		TermSelect:  line(0.22),
		SearchMatch: alpha(fg, 0x33),
	}
	t.Success = t.Text
	t.Warning = t.TextMuted

	if !sp.dark {
		t.Scrim = alpha(hex(0x000000), 0x33)
		t.Dim = alpha(surface, 0x55)
	}

	for i, v := range sp.ansi {
		t.ANSI[i] = hex(v)
	}

	canvas := hex(sp.canvas)
	t.Frame, t.FrameText, t.FrameMuted = canvas, t.Text, t.TextMuted
	t.FramePanel, t.FrameHover, t.FrameBorder = Mix(canvas, fg, 0.06), Mix(canvas, fg, 0.05), Mix(canvas, fg, 0.07)

	t.buildPalette()
	return t
}

func (t *Theme) OnFrame() *Theme {
	return t
}

func (t *Theme) TagColor(i int) color.NRGBA {
	return t.TextMuted
}

func (t *Theme) buildPalette() {
	copy(t.palette[:16], t.ANSI[:])

	levels := [6]uint8{0, 95, 135, 175, 215, 255}

	for i := 0; i < 216; i++ {
		t.palette[16+i] = color.NRGBA{R: levels[i/36], G: levels[(i/6)%6], B: levels[i%6], A: 0xff}
	}

	for i := 0; i < 24; i++ {
		v := uint8(8 + i*10)
		t.palette[232+i] = color.NRGBA{R: v, G: v, B: v, A: 0xff}
	}
}

func (t *Theme) Color(c vt.Color, fg bool) color.NRGBA {
	switch c.Kind() {
	case vt.ColorIndexed:
		return t.palette[c.Index()]
	case vt.ColorRGB:
		r, g, b := c.RGB()
		return color.NRGBA{R: r, G: g, B: b, A: 0xff}
	}

	if fg {
		return t.Foreground
	}

	return t.Background
}

func Mix(a, b color.NRGBA, f float32) color.NRGBA {
	lerp := func(x, y uint8) uint8 { return uint8(float32(x) + (float32(y)-float32(x))*f) }
	return color.NRGBA{R: lerp(a.R, b.R), G: lerp(a.G, b.G), B: lerp(a.B, b.B), A: lerp(a.A, b.A)}
}

func WithAlpha(c color.NRGBA, a uint8) color.NRGBA {
	return alpha(c, a)
}
