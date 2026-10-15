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
	Rune rune
	Fg   Color
	Bg   Color
	Attr Attr
	Link uint16
}

const (
	attrWidthShift      = 14
	attrWidthMask  Attr = 3 << attrWidthShift
)

func (c Cell) Width() uint8 {
	return uint8(c.Attr >> attrWidthShift)
}

func (c *Cell) SetWidth(w uint8) {
	c.Attr = c.Attr&^attrWidthMask | Attr(w)<<attrWidthShift
}

func (c Cell) Style() Attr {
	return c.Attr &^ attrWidthMask
}

func blank(bg Color) Cell {
	return Cell{Bg: bg, Attr: 1 << attrWidthShift}
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
