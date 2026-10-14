package glyphs

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-text/typesetting/font"
)

type Style uint8

const (
	Regular Style = iota
	Bold
	Italic
	BoldItalic
)

const winFonts = `C:\Windows\Fonts`

var fallbackFiles = []string{
	"seguisym.ttf",
	"CascadiaMono.ttf",
	"consola.ttf",
	"msgothic.ttc",
	"msyh.ttc",
	"malgun.ttf",
	"seguihis.ttf",
	"arial.ttf",
	"segoeui.ttf",
}

const emojiFile = "seguiemj.ttf"

type face struct {
	*font.Face
	upem float32
}

type Set struct {
	styles [4]*face

	mu        sync.Mutex
	fallbacks []*face
	loaded    bool
	emoji     *face
	emojiOK   bool
	lookup    map[runeKey]lookupResult
}

type runeKey struct {
	r     rune
	style Style
	emoji bool
}

type lookupResult struct {
	f     *face
	gid   font.GID
	found bool
}

func NewSet(fsys fs.FS, files [4]string) (*Set, error) {
	s := &Set{lookup: map[runeKey]lookupResult{}}

	for i, name := range files {
		b, err := fs.ReadFile(fsys, name)

		if err != nil {
			return nil, err
		}

		f, err := parse(b)

		if err != nil {
			return nil, err
		}

		s.styles[i] = f
	}

	return s, nil
}

func parse(b []byte) (*face, error) {
	f, err := font.ParseTTF(bytes.NewReader(b))

	if err != nil {
		return nil, err
	}

	return &face{Face: f, upem: float32(f.Upem())}, nil
}

func loadFile(path string) []*face {
	b, err := os.ReadFile(path)

	if err != nil {
		return nil
	}

	if strings.HasSuffix(strings.ToLower(path), ".ttc") {
		fs, err := font.ParseTTC(bytes.NewReader(b))

		if err != nil || len(fs) == 0 {
			return nil
		}

		return []*face{{Face: fs[0], upem: float32(fs[0].Upem())}}
	}

	f, err := parse(b)

	if err != nil {
		return nil
	}

	return []*face{f}
}

func (s *Set) loadFallbacks() {
	if s.loaded {
		return
	}

	s.loaded = true

	for _, name := range fallbackFiles {
		s.fallbacks = append(s.fallbacks, loadFile(filepath.Join(winFonts, name))...)
	}
}

func (s *Set) emojiFace() *face {
	if !s.emojiOK {
		s.emojiOK = true

		if fs := loadFile(filepath.Join(winFonts, emojiFile)); len(fs) > 0 {
			s.emoji = fs[0]
		}
	}

	return s.emoji
}

func (s *Set) Primary() *font.Face {
	return s.styles[Regular].Face
}

func (s *Set) find(r rune, style Style, preferEmoji bool) (*face, font.GID, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := runeKey{r, style, preferEmoji}

	if res, ok := s.lookup[k]; ok {
		return res.f, res.gid, res.found
	}

	res := s.resolve(r, style, preferEmoji)
	s.lookup[k] = res
	return res.f, res.gid, res.found
}

func (s *Set) resolve(r rune, style Style, preferEmoji bool) lookupResult {
	try := func(f *face) (lookupResult, bool) {
		if f == nil {
			return lookupResult{}, false
		}

		if gid, ok := f.NominalGlyph(r); ok && gid != 0 {
			return lookupResult{f: f, gid: gid, found: true}, true
		}

		return lookupResult{}, false
	}

	if preferEmoji {
		if res, ok := try(s.emojiFace()); ok {
			return res
		}
	}

	if res, ok := try(s.styles[style]); ok {
		return res
	}

	if res, ok := try(s.styles[Regular]); ok {
		return res
	}

	s.loadFallbacks()

	for _, f := range s.fallbacks {
		if res, ok := try(f); ok {
			return res
		}
	}

	if res, ok := try(s.emojiFace()); ok {
		return res
	}

	return lookupResult{f: s.styles[Regular], gid: 0, found: false}
}
