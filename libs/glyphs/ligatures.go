package glyphs

import (
	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
)

type ligKey struct {
	text  string
	style Style
}

type gidKey struct {
	gid   font.GID
	style Style
}

var english = language.NewLanguage("en")

func (c *Cache) styleFace(style Style) *face {
	if f := c.set.styles[style]; f != nil {
		return f
	}

	return c.set.styles[Regular]
}

func (c *Cache) Ligatures(text []rune, style Style) []font.GID {
	key := ligKey{string(text), style}

	c.ligMu.Lock()
	defer c.ligMu.Unlock()

	if gids, ok := c.ligs[key]; ok {
		return gids
	}

	gids := c.shape(text, style)
	c.ligs[key] = gids
	return gids
}

func (c *Cache) shape(text []rune, style Style) []font.GID {
	f := c.styleFace(style)
	out := c.shaper.Shape(shaping.Input{
		Text:      text,
		RunEnd:    len(text),
		Direction: di.DirectionLTR,
		Face:      f.Face,
		Size:      fixed.I(int(c.px)),
		Script:    language.Latin,
		Language:  english,
	})

	if len(out.Glyphs) != len(text) {
		return nil
	}

	gids := make([]font.GID, len(text))
	changed := false

	for _, g := range out.Glyphs {
		if g.RuneCount != 1 || g.GlyphCount != 1 || g.ClusterIndex >= len(text) {
			return nil
		}

		nominal, _ := f.NominalGlyph(text[g.ClusterIndex])

		if g.GlyphID != nominal {
			gids[g.ClusterIndex] = g.GlyphID
			changed = true
		}
	}

	if !changed {
		return nil
	}

	return gids
}

func (c *Cache) GetGlyph(gid font.GID, style Style) *Glyph {
	k := gidKey{gid, style}

	c.mu.Lock()
	g, ok := c.byGID[k]
	c.mu.Unlock()

	if ok {
		return g
	}

	g = c.renderFace(c.styleFace(style), gid, style, 1)

	c.mu.Lock()
	c.byGID[k] = g
	c.mu.Unlock()
	return g
}
