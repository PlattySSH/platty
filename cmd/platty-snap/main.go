package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/plattyssh/vt"

	"github.com/plattyssh/platty/internal/fonts"
	"github.com/plattyssh/platty/internal/ui/kit"
	"github.com/plattyssh/platty/internal/ui/termview"
	"github.com/plattyssh/platty/internal/ui/theme"
)

type backend struct {
	term *vt.Terminal
}

func (b *backend) Terminal() *vt.Terminal { return b.term }
func (b *backend) Send([]byte)            {}
func (b *backend) Resize(cols, rows int)  { b.term.Resize(cols, rows) }

func main() {
	out := flag.String("o", "snap.png", "output PNG file")
	width := flag.Int("w", 1100, "width in pixels")
	height := flag.Int("h", 600, "height in pixels")
	scale := flag.Float64("scale", 1, "scale (px per dp)")
	flag.Parse()

	if err := run(*out, *width, *height, float32(*scale), flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(out string, width, height int, scale float32, command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("usage: platty-snap [-o file.png] -- command [args...]")
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor", "PYTHONIOENCODING=utf-8")
	data, err := cmd.Output()

	if err != nil {
		return fmt.Errorf("running %v: %w", command, err)
	}

	term := vt.New(80, 24)
	shaper, _ := fonts.Load()
	set, _ := fonts.Glyphs("")
	view := termview.New(&backend{term: term}, kit.New(theme.New(theme.Carbon), shaper), set, termview.Settings{})

	win, err := headless.NewWindow(width, height)

	if err != nil {
		return err
	}

	defer win.Release()

	size := image.Pt(width, height)
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(size),
		Metric:      unit.Metric{PxPerDp: scale, PxPerSp: scale},
	}

	view.Layout(gtx)
	term.Write(toCRLF(data))

	gtx.Ops.Reset()
	view.Layout(gtx)

	if err := win.Frame(gtx.Ops); err != nil {
		return err
	}

	img := image.NewRGBA(image.Rectangle{Max: size})

	if err := win.Screenshot(img); err != nil {
		return err
	}

	f, err := os.Create(out)

	if err != nil {
		return err
	}

	defer f.Close()

	return png.Encode(f, img)
}

func toCRLF(p []byte) []byte {
	out := make([]byte, 0, len(p)+len(p)/16)

	for i, b := range p {
		if b == '\n' && (i == 0 || p[i-1] != '\r') {
			out = append(out, '\r')
		}

		out = append(out, b)
	}

	return out
}
