package vt

type Color uint32

type ColorKind uint8

const (
	ColorDefault ColorKind = iota
	ColorIndexed
	ColorRGB
)

const (
	colorKindShift = 24
	colorValueMask = 0x00ffffff
)

const DefaultColor Color = 0

func Indexed(i uint8) Color {
	return Color(uint32(ColorIndexed)<<colorKindShift | uint32(i))
}

func RGB(r, g, b uint8) Color {
	return Color(uint32(ColorRGB)<<colorKindShift | uint32(r)<<16 | uint32(g)<<8 | uint32(b))
}

func (c Color) Kind() ColorKind {
	return ColorKind(c >> colorKindShift)
}

func (c Color) Index() uint8 {
	return uint8(c & colorValueMask)
}

func (c Color) RGB() (r, g, b uint8) {
	return uint8(c >> 16), uint8(c >> 8), uint8(c)
}

type Attr uint16

const (
	AttrBold Attr = 1 << iota
	AttrFaint
	AttrItalic
	AttrUnderline
	AttrBlink
	AttrInverse
	AttrInvisible
	AttrStrike
	AttrDoubleUnderline
	AttrCurlyUnderline
)

type Cell struct {
	Rune  rune
	Width uint8
	Attr  Attr
	Fg    Color
	Bg    Color
	Link  uint16
}

func blank(bg Color) Cell {
	return Cell{Width: 1, Bg: bg}
}

type CursorStyle uint8

const (
	CursorBlock CursorStyle = iota
	CursorUnderline
	CursorBar
)

type Cursor struct {
	X, Y    int
	Visible bool
	Style   CursorStyle
	Blink   bool
	Custom  bool
}
