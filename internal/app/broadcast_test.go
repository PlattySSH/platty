package app

import (
	"testing"
)

func TestBroadcast(t *testing.T) {
	h := newHarness(t)
	h.a.RunLocal("left", "cmd.exe", "/k", "echo left")
	h.a.Split(true, "right", "cmd.exe", "/k", "echo right")
	h.frames(10)

	tab := h.a.ws.tabs[0]
	panes := tab.root.panes(nil)
	h.a.toggleBroadcast(panes[0])
	h.frames(4)
	h.shot("broadcast")

	if !tab.broadcast || !panes[0].mirrored || !panes[1].mirrored {
		t.Fatal("broadcast didn't set up mirroring on every pane")
	}

	if extras := h.a.paneExtras(panes[0]); extras[1].Item.Label != "Stop broadcasting" {
		t.Errorf("menu says %q while broadcasting", extras[1].Item.Label)
	}

	h.a.toggleBroadcast(panes[1])

	if tab.broadcast || panes[0].mirrored {
		t.Error("stopping broadcast left mirroring on")
	}
}
