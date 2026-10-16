package glyphs

import (
	"os"
	"path/filepath"
)

var fallbackFiles = []string{
	"Apple Symbols.ttf",
	"Menlo.ttc",
	"Hiragino Sans GB.ttc",
	"PingFang.ttc",
	"AppleSDGothicNeo.ttc",
	"Arial Unicode.ttf",
	"Helvetica.ttc",
}

var emojiFiles []string

func fontDirs() []string {
	dirs := []string{"/System/Library/Fonts", "/Library/Fonts"}

	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Library", "Fonts"))
	}

	return dirs
}
