package app

import (
	"image"
	"path/filepath"
	"testing"
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/plattyssh/platty/internal/vault"
)

func routedFrame(p *Preview, r *input.Router) {
	ops := new(op.Ops)
	gtx := layout.Context{
		Ops:         ops,
		Constraints: layout.Exact(image.Pt(1280, 800)),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Now:         time.Now(),
		Source:      r.Source(),
	}
	p.Frame(gtx)
	r.Frame(ops)
}

func TestDragHostIntoFolder(t *testing.T) {
	v, err := vault.OpenAt(filepath.Join(t.TempDir(), "vault.dat"))

	if err != nil {
		t.Fatal(err)
	}

	v.CreateFolder("Prod")
	h, _ := v.SaveHost(vault.Host{Label: "web", Address: "10.0.0.1"})

	p := NewPreview(v, "carbon")
	defer p.Close()
	p.tools.refresh(p.App)

	var r input.Router
	routedFrame(p, &r)
	routedFrame(p, &r)

	hostY := folderY + float32(navRowHeight)
	x := float32(120)

	r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(x, hostY)},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(x, hostY)},
	)
	routedFrame(p, &r)

	for y := hostY; y >= folderY; y -= 6 {
		r.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: f32.Pt(x, y)})
		routedFrame(p, &r)
	}

	r.Queue(pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: f32.Pt(x, folderY)})

	for i := 0; i < 4; i++ {
		routedFrame(p, &r)
	}

	got, _ := v.Host(h.ID)

	if got.Group != "Prod" {
		t.Fatalf("the host should end up in \"Prod\", it's in %q", got.Group)
	}
}

func TestClickFolderCollapses(t *testing.T) {
	v, err := vault.OpenAt(filepath.Join(t.TempDir(), "vault.dat"))

	if err != nil {
		t.Fatal(err)
	}

	v.SaveHost(vault.Host{Label: "web", Address: "10.0.0.1", Group: "Prod"})

	p := NewPreview(v, "carbon")
	defer p.Close()
	p.tools.refresh(p.App)

	var r input.Router
	routedFrame(p, &r)

	pos := f32.Pt(120, folderY)
	r.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: pos},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Buttons: pointer.ButtonPrimary, Position: pos},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: pos},
	)
	routedFrame(p, &r)
	routedFrame(p, &r)

	if !p.tools.hosts.Collapsed("Prod") {
		t.Fatal("the click didn't collapse the folder")
	}
}

var folderY = float32(sidebarTopHeight+40+3*navRowHeight+48) + float32(navRowHeight)/2
