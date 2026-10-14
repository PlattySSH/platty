package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"golang.org/x/image/vector"

	"github.com/plattyssh/platty/internal/ui/mascot"
)

var canvas = color.NRGBA{R: 0x14, G: 0x14, B: 0x13, A: 0xff}

func main() {
	out := flag.String("o", "assets/mascot", "output directory")
	flag.Parse()

	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(out string) error {
	for _, d := range []string{out, filepath.Join(out, "frames")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	light := mascot.LightColors
	open := mascot.State{}

	for _, n := range []int{16, 24, 32, 48, 64, 128, 256, 512, 1024} {
		if err := save(filepath.Join(out, fmt.Sprintf("sol-%d.png", n)), n, open, mascot.DefaultColors, nil); err != nil {
			return err
		}
	}

	for _, n := range []int{256, 512} {
		if err := save(filepath.Join(out, fmt.Sprintf("sol-light-%d.png", n)), n, open, light, nil); err != nil {
			return err
		}

		happy := open
		happy.Happy = 1

		if err := save(filepath.Join(out, fmt.Sprintf("sol-happy-%d.png", n)), n, happy, mascot.DefaultColors, nil); err != nil {
			return err
		}

		if err := save(filepath.Join(out, fmt.Sprintf("sol-dark-%d.png", n)), n, open, mascot.DefaultColors, &canvas); err != nil {
			return err
		}
	}

	const fps = 25
	frames := int(math.Round(mascot.Period * fps))
	anim := &gif.GIF{LoopCount: 0}
	var pal color.Palette

	for i := 0; i < frames; i++ {
		t := float64(i) / fps
		st := mascot.State{
			T:     t,
			Blink: mascot.BlinkAt(t),
			Look:  f32.Pt(float32(math.Sin(t*2*math.Pi/mascot.Period)), float32(0.3*math.Sin(t*4*math.Pi/mascot.Period))),
		}
		img, err := render(256, st, mascot.DefaultColors, nil)

		if err != nil {
			return err
		}

		if err := writePNG(filepath.Join(out, "frames", fmt.Sprintf("sol-%03d.png", i)), img); err != nil {
			return err
		}

		bg, err := render(256, st, mascot.DefaultColors, &canvas)

		if err != nil {
			return err
		}

		if pal == nil {
			pal = paletteOf(bg)
		}

		p := image.NewPaletted(bg.Bounds(), pal)
		draw.FloydSteinberg.Draw(p, bg.Bounds(), bg, image.Point{})
		anim.Image = append(anim.Image, p)
		anim.Delay = append(anim.Delay, 100/fps)
	}

	f, err := os.Create(filepath.Join(out, "sol.gif"))

	if err != nil {
		return err
	}

	defer f.Close()

	if err := gif.EncodeAll(f, anim); err != nil {
		return err
	}

	if err := writeGitHub(out); err != nil {
		return err
	}

	return writeICO(filepath.Join(out, "platty.ico"), []int{16, 24, 32, 48, 64, 128, 256})
}

func render(n int, st mascot.State, c mascot.Colors, bg *color.NRGBA) (*image.RGBA, error) {
	img, err := renderAlpha(n, st, c)

	if err != nil {
		return nil, err
	}

	clipToOutline(img, float32(n), st, f32.Affine2D{})
	img = addShadow(img, c.Light)

	if bg != nil {
		over(img, *bg)
	}

	return img, nil
}

func renderRaw(n int, st mascot.State, c mascot.Colors, bg *color.NRGBA) (*image.RGBA, error) {
	scale := 2

	if n >= 256 {
		scale = 1
	}

	w := n * scale
	win, err := headless.NewWindow(w, w)

	if err != nil {
		return nil, err
	}

	defer win.Release()

	ops := new(op.Ops)

	if bg != nil {
		paint.FillShape(ops, *bg, clip.Rect{Max: image.Pt(w, w)}.Op())
	}

	mascot.Draw(ops, float32(w), st, c)

	if err := win.Frame(ops); err != nil {
		return nil, err
	}

	img := image.NewRGBA(image.Rect(0, 0, w, w))

	if err := win.Screenshot(img); err != nil {
		return nil, err
	}

	if scale == 1 {
		return img, nil
	}

	return downscale(img, scale), nil
}

func downscale(src *image.RGBA, f int) *image.RGBA {
	n := src.Bounds().Dx() / f
	dst := image.NewRGBA(image.Rect(0, 0, n, n))

	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			var r, g, b, a int

			for dy := 0; dy < f; dy++ {
				for dx := 0; dx < f; dx++ {
					c := src.RGBAAt(x*f+dx, y*f+dy)
					r, g, b, a = r+int(c.R), g+int(c.G), b+int(c.B), a+int(c.A)
				}
			}

			k := f * f
			dst.SetRGBA(x, y, color.RGBA{R: uint8(r / k), G: uint8(g / k), B: uint8(b / k), A: uint8(a / k)})
		}
	}

	return dst
}

func save(path string, n int, st mascot.State, c mascot.Colors, bg *color.NRGBA) error {
	img, err := render(n, st, c, bg)

	if err != nil {
		return err
	}

	return writePNG(path, img)
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)

	if err != nil {
		return err
	}

	defer f.Close()
	return png.Encode(f, img)
}

func paletteOf(img *image.RGBA) color.Palette {
	count := map[color.RGBA]int{}
	b := img.Bounds()

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			count[img.RGBAAt(x, y)]++
		}
	}

	cols := make([]color.RGBA, 0, len(count))

	for c := range count {
		cols = append(cols, c)
	}

	sort.Slice(cols, func(i, j int) bool { return count[cols[i]] > count[cols[j]] })
	pal := color.Palette{}

	for _, c := range cols {
		if len(pal) == 240 {
			break
		}

		pal = append(pal, c)
	}

	for _, c := range palette.WebSafe {
		if len(pal) == 256 {
			break
		}

		if r, g, b, _ := c.RGBA(); r == g && g == b {
			pal = append(pal, c)
		}
	}

	return pal
}

func writeICO(path string, sizes []int) error {
	var imgs [][]byte

	for _, n := range sizes {
		st := mascot.State{}
		img, err := render(n, st, mascot.DefaultColors, nil)

		if err != nil {
			return err
		}

		var buf bytes.Buffer

		if err := png.Encode(&buf, img); err != nil {
			return err
		}

		imgs = append(imgs, buf.Bytes())
	}

	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, [3]uint16{0, 1, uint16(len(sizes))})
	offset := 6 + 16*len(sizes)

	for i, n := range sizes {
		dim := uint8(n)

		if n >= 256 {
			dim = 0
		}

		binary.Write(&out, binary.LittleEndian, struct {
			W, H, Colors, Reserved uint8
			Planes, BPP            uint16
			Size, Offset           uint32
		}{dim, dim, 0, 0, 1, 32, uint32(len(imgs[i])), uint32(offset)})
		offset += len(imgs[i])
	}

	for _, b := range imgs {
		out.Write(b)
	}

	return os.WriteFile(path, out.Bytes(), 0o644)
}

func renderAlpha(n int, st mascot.State, c mascot.Colors) (*image.RGBA, error) {
	black, err := renderRaw(n, st, c, &color.NRGBA{A: 0xff})

	if err != nil {
		return nil, err
	}

	white, err := renderRaw(n, st, c, &color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})

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

func addShadow(img *image.RGBA, light bool) *image.RGBA {
	n := img.Bounds().Dx()
	alpha := make([]float64, n*n)

	for i := range alpha {
		alpha[i] = float64(img.Pix[i*4+3]) / 255
	}

	r := max(1, n/40)

	for pass := 0; pass < 3; pass++ {
		alpha = boxBlur(alpha, n, r)
	}

	dy := n * 35 / 1000
	strength := 0.42

	if light {
		strength = 0.2
	}

	out := image.NewRGBA(img.Bounds())
	copy(out.Pix, img.Pix)

	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			sy := y - dy

			if sy < 0 {
				continue
			}

			sh := alpha[sy*n+x] * strength
			i := (y*n + x) * 4
			a := float64(out.Pix[i+3]) / 255
			out.Pix[i+3] = uint8(math.Round(255 * (a + (1-a)*sh)))
		}
	}

	return out
}

func boxBlur(src []float64, n, r int) []float64 {
	tmp := make([]float64, len(src))
	dst := make([]float64, len(src))
	w := float64(2*r + 1)

	for y := 0; y < n; y++ {
		sum := 0.0

		for x := -r; x <= r; x++ {
			sum += src[y*n+min(max(x, 0), n-1)]
		}

		for x := 0; x < n; x++ {
			tmp[y*n+x] = sum / w
			sum += src[y*n+min(x+r+1, n-1)] - src[y*n+max(x-r, 0)]
		}
	}

	for x := 0; x < n; x++ {
		sum := 0.0

		for y := -r; y <= r; y++ {
			sum += tmp[min(max(y, 0), n-1)*n+x]
		}

		for y := 0; y < n; y++ {
			dst[y*n+x] = sum / w
			sum += tmp[min(y+r+1, n-1)*n+x] - tmp[max(y-r, 0)*n+x]
		}
	}

	return dst
}

func over(img *image.RGBA, bg color.NRGBA) {
	for i := 0; i < len(img.Pix); i += 4 {
		a := float64(img.Pix[i+3]) / 255
		img.Pix[i+0] = uint8(math.Round(float64(img.Pix[i+0]) + float64(bg.R)*(1-a)))
		img.Pix[i+1] = uint8(math.Round(float64(img.Pix[i+1]) + float64(bg.G)*(1-a)))
		img.Pix[i+2] = uint8(math.Round(float64(img.Pix[i+2]) + float64(bg.B)*(1-a)))
		img.Pix[i+3] = 0xff
	}
}

func clipToOutline(img *image.RGBA, size float32, st mascot.State, tr f32.Affine2D) {
	n := img.Bounds().Dx()
	cs := mascot.Outline(size, st)
	c := f32.Pt(size/2, size*0.48)
	grow := func(p f32.Point) f32.Point {
		d := p.Sub(c)
		return tr.Transform(c.Add(d.Mul(1.035)))
	}
	var r vector.Rasterizer
	r.Reset(n, n)
	p0 := grow(cs[0].P0)
	r.MoveTo(p0.X, p0.Y)

	for _, q := range cs {
		c1, c2, p3 := grow(q.C1), grow(q.C2), grow(q.P3)
		r.CubeTo(c1.X, c1.Y, c2.X, c2.Y, p3.X, p3.Y)
	}

	r.ClosePath()
	mask := image.NewAlpha(image.Rect(0, 0, n, n))
	r.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})

	for i := 0; i < n*n; i++ {
		m := float64(mask.Pix[i]) / 255

		if m >= 1 {
			continue
		}

		for j := 0; j < 4; j++ {
			img.Pix[i*4+j] = uint8(math.Round(float64(img.Pix[i*4+j]) * m))
		}
	}
}
