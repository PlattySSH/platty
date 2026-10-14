package app

import (
	"image"
	"path/filepath"
	"testing"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/plattyssh/platty/internal/vault"
)

func testFrame(p *Preview) {
	gtx := layout.Context{
		Ops:         new(op.Ops),
		Constraints: layout.Exact(image.Pt(1280, 800)),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Now:         time.Now(),
	}
	p.Frame(gtx)
}

func TestCloseLastTabDoesNotPanic(t *testing.T) {
	v, err := vault.OpenAt(filepath.Join(t.TempDir(), "vault.dat"))

	if err != nil {
		t.Fatal(err)
	}

	p := NewPreview(v, "dark")
	defer p.Close()

	p.RunLocal("cmd", "cmd.exe", "/k")
	p.RunLocal("cmd", "cmd.exe", "/k")
	testFrame(p)

	for len(p.ws.tabs) > 0 {
		p.ws.tabs[len(p.ws.tabs)-1].close.Click()
		testFrame(p)
	}

	testFrame(p)

	if len(p.ws.tabs) != 0 {
		t.Fatalf("%d tabs left", len(p.ws.tabs))
	}
}
