package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/plattyssh/platty/internal/ui/mascot"
)

type avatar struct {
	name  string
	base  color.NRGBA
	blobs []blob
}

type blob struct {
	x, y, r float64
	c       color.NRGBA
	k       float64
}

var avatars = []avatar{
	{"black", hexc(0x050506), []blob{
		{0.62, 0.36, 0.5, hexc(0x2a2b30), 0.7},
	}},
	{"graphite", hexc(0x060607), []blob{
		{0.9, 0.1, 0.55, hexc(0x34363d), 0.75},
		{0.05, 0.95, 0.5, hexc(0x15161a), 0.8},
	}},
	{"pure", hexc(0x000000), nil},
}

func hexc(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 0xff}
}

func writeGitHub(out string) error {
	dir := filepath.Join(out, "github")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	for _, av := range avatars {
		img, err := renderAvatar(1024, av)

		if err != nil {
			return err
		}

		if err := writePNG(filepath.Join(dir, fmt.Sprintf("platty-github-%s.png", av.name)), img); err != nil {
			return err
		}
	}

	return nil
}

func renderAvatar(n int, av avatar) (*image.RGBA, error) {
	s := float32(n)

	size := s * 0.86
	origin := f32.Pt((s-size)/2, (s-size)/2+s*0.025)
	tr := f32.Affine2D{}.Offset(origin).Rotate(f32.Pt(s/2, s/2), 0.22)
	sprite, err := layerAlpha(n, func(ops *op.Ops) {
		st := op.Affine(tr).Push(ops)
		mascot.Draw(ops, size, mascot.State{T: 0.8, Look: f32.Pt(1, -1)}, mascot.DefaultColors)
		st.Pop()
	})

	if err != nil {
		return nil, err
	}

	clipToOutline(sprite, size, mascot.State{T: 0.8, Look: f32.Pt(1, -1)}, tr)
	sprite = addShadow(sprite, false)

	img := image.NewRGBA(image.Rect(0, 0, n, n))

	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			c := [3]float64{float64(av.base.R), float64(av.base.G), float64(av.base.B)}

			for _, b := range av.blobs {
				dx := float64(x)/float64(n) - b.x
				dy := float64(y)/float64(n) - b.y
				w := b.k * math.Exp(-(dx*dx+dy*dy)/(b.r*b.r))
				c[0] += (float64(b.c.R) - c[0]) * w
				c[1] += (float64(b.c.G) - c[1]) * w
				c[2] += (float64(b.c.B) - c[2]) * w
			}

			noise := float64(hash(uint32(x), uint32(y))>>24)/255 - 0.5
			sp := sprite.RGBAAt(x, y)
			a := float64(sp.A) / 255
			px := func(i int, s uint8) uint8 {
				v := c[i]*(1-a) + float64(s) + noise
				return uint8(max(0, min(255, math.Round(v))))
			}
			img.SetRGBA(x, y, color.RGBA{R: px(0, sp.R), G: px(1, sp.G), B: px(2, sp.B), A: 0xff})
		}
	}

	return img, nil
}

func layerAlpha(n int, draw func(ops *op.Ops)) (*image.RGBA, error) {
	shot := func(bg color.NRGBA) (*image.RGBA, error) {
		win, err := headless.NewWindow(n, n)

		if err != nil {
			return nil, err
		}

		defer win.Release()
		ops := new(op.Ops)
		paint.FillShape(ops, bg, clip.Rect{Max: image.Pt(n, n)}.Op())
		draw(ops)

		if err := win.Frame(ops); err != nil {
			return nil, err
		}

		img := image.NewRGBA(image.Rect(0, 0, n, n))
		return img, win.Screenshot(img)
	}
	black, err := shot(color.NRGBA{A: 0xff})

	if err != nil {
		return nil, err
	}

	white, err := shot(color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})

	if err != nil {
		return nil, err
	}

	out := image.NewRGBA(black.Bounds())

	for i := 0; i < len(out.Pix); i += 4 {
		d := (int(white.Pix[i]) - int(black.Pix[i]) + int(white.Pix[i+1]) - int(black.Pix[i+1]) + int(white.Pix[i+2]) - int(black.Pix[i+2])) / 3
		a := max(0, min(255, 255-d))
		out.Pix[i+0] = uint8(min(int(black.Pix[i+0]), a))
		out.Pix[i+1] = uint8(min(int(black.Pix[i+1]), a))
		out.Pix[i+2] = uint8(min(int(black.Pix[i+2]), a))
		out.Pix[i+3] = uint8(a)
	}

	return out, nil
}

func hash(x, y uint32) uint32 {
	h := x*0x9e3779b1 ^ y*0x85ebca77
	h ^= h >> 16
	h *= 0x7feb352d
	h ^= h >> 15
	h *= 0x846ca68b
	h ^= h >> 16
	return h
}
