package app

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/gpu/headless"
	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/plattyssh/platty/internal/app/core"
	"github.com/plattyssh/platty/internal/sshx"
	"github.com/plattyssh/platty/internal/vault"
)

type harness struct {
	t    *testing.T
	a    *Preview
	r    input.Router
	win  *headless.Window
	size image.Point
	ops  op.Ops
	now  time.Time
	pos  f32.Point
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	dir := t.TempDir()
	v, err := vault.OpenAt(filepath.Join(dir, "vault.dat"))

	if err != nil {
		t.Fatal(err)
	}

	for _, h := range []vault.Host{
		{Label: "web-01", Address: "10.0.1.11", Username: "deploy", Group: "Production/Web", LastUsed: time.Now().Add(-4 * time.Minute)},
		{Label: "web-02", Address: "10.0.1.12", Username: "deploy", Group: "Production/Web"},
		{Label: "postgres-main", Address: "10.0.2.5", Username: "postgres", Group: "Production"},
		{Label: "raspberry", Address: "192.168.1.40", Username: "pi", Group: "Cabin", LastUsed: time.Now().Add(-3 * time.Hour)},
		{Label: "nas", Address: "192.168.1.10", Username: "admin", Group: "Cabin"},
		{Label: "vps", Address: "203.0.113.7", Username: "root", LastUsed: time.Now().Add(-30 * time.Hour)},
	} {
		v.SaveHost(h)
	}

	v.CreateFolder("Clients/Acme")

	h := &harness{t: t, size: image.Pt(1440, 900), now: time.Now()}
	h.a = NewPreview(v, "carbon")
	h.a.tools.open[core.Left] = 0
	win, err := headless.NewWindow(h.size.X, h.size.Y)

	if err != nil {
		t.Skip("no headless GPU:", err)
	}

	h.win = win
	t.Cleanup(func() { h.a.Close(); win.Release() })
	h.frames(3)
	return h
}

func (h *harness) frame() {
	h.now = h.now.Add(50 * time.Millisecond)
	h.ops.Reset()
	gtx := layout.Context{
		Ops:         &h.ops,
		Constraints: layout.Exact(h.size),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Now:         h.now,
		Source:      h.r.Source(),
	}
	h.a.frame(gtx)
	h.r.Frame(&h.ops)

	if err := h.win.Frame(&h.ops); err != nil {
		h.t.Fatal(err)
	}
}

func (h *harness) frames(n int) {
	for i := 0; i < n; i++ {
		h.frame()
	}
}

func (h *harness) ptr(kind pointer.Kind, x, y float32, b pointer.Buttons) {
	h.pos = f32.Pt(x, y)
	h.r.Queue(pointer.Event{Kind: kind, Source: pointer.Mouse, Position: h.pos, Buttons: b, Time: time.Duration(h.now.UnixNano())})
	h.frame()
}

func (h *harness) move(x, y float32) { h.ptr(pointer.Move, x, y, 0) }

func (h *harness) click(x, y float32) {
	h.move(x, y)
	h.ptr(pointer.Press, x, y, pointer.ButtonPrimary)
	h.ptr(pointer.Release, x, y, 0)
	h.frames(4)
}

func (h *harness) rclick(x, y float32) {
	h.move(x, y)
	h.ptr(pointer.Press, x, y, pointer.ButtonSecondary)
	h.ptr(pointer.Release, x, y, 0)
	h.frames(4)
}

func (h *harness) drag(x0, y0, x1, y1 float32, hold bool) {
	h.move(x0, y0)
	h.ptr(pointer.Press, x0, y0, pointer.ButtonPrimary)

	for i := 1; i <= 10; i++ {
		t := float32(i) / 10
		h.ptr(pointer.Move, x0+(x1-x0)*t, y0+(y1-y0)*t, pointer.ButtonPrimary)
	}

	if !hold {
		h.ptr(pointer.Release, x1, y1, 0)
	}

	h.frames(4)
}

func (h *harness) key(name key.Name, mods key.Modifiers) {
	h.r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Press})
	h.frame()
	h.r.Queue(key.Event{Name: name, Modifiers: mods, State: key.Release})
	h.frames(2)
}

func (h *harness) text(s string) {
	h.r.Queue(key.EditEvent{Text: s})
	h.frames(2)
}

func (h *harness) shot(name string) {
	dir := os.Getenv("PLATTY_SHOTS")

	if dir == "" {
		return
	}

	h.frames(6)
	img := image.NewRGBA(image.Rectangle{Max: h.size})

	if err := h.win.Screenshot(img); err != nil {
		h.t.Fatal(err)
	}

	f, err := os.Create(filepath.Join(dir, name+".png"))

	if err != nil {
		h.t.Fatal(err)
	}

	defer f.Close()
	png.Encode(f, img)
}

func sshxGenerate(label string) (vault.Key, error) { return sshx.GenerateEd25519(label, "") }
