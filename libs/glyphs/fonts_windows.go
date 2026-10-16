package glyphs

import (
	"os"
	"path/filepath"
)

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

var emojiFiles = []string{"seguiemj.ttf"}

func fontDirs() []string {
	windir := os.Getenv("WINDIR")

	if windir == "" {
		windir = `C:\Windows`
	}

	return []string{filepath.Join(windir, "Fonts")}
}
