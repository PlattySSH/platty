package app

import (
	"path/filepath"
	"testing"

	"gioui.org/io/key"
)

func TestTabExtras(t *testing.T) {
	h := newHarness(t)
	h.a.NewLocalTab(h.a.profiles[0])
	h.a.NewLocalTab(h.a.profiles[0])
	h.frames(10)

	first := h.a.ws.tabs[0]
	h.a.chrome.startRename(h.a.App, first)
	h.frames(3)
	h.a.chrome.rename.Editor().SetText("deploy box")
	h.key(key.NameReturn, 0)
	h.frames(3)
	h.shot("tab-renamed")

	if first.name != "deploy box" || h.a.chrome.renaming != nil {
		t.Fatalf("rename: name %q, still renaming %v", first.name, h.a.chrome.renaming != nil)
	}

	h.a.duplicateTab(first)
	h.frames(6)

	if n := len(h.a.ws.tabs); n != 3 {
		t.Fatalf("after duplicating there are %d tabs, want 3", n)
	}

	h.a.ws.active = 0
	h.a.ws.split(h.a.App, true)
	h.frames(6)

	path := filepath.Join(t.TempDir(), "tabs.json")
	h.a.saveTabsTo(path)

	h2 := newHarness(t)
	h2.a.restoreTabsFrom(path)
	h2.frames(6)
	h2.shot("tabs-restored")

	if n := len(h2.a.ws.tabs); n != 3 {
		t.Fatalf("restored %d tabs, want 3", n)
	}

	if h2.a.ws.tabs[0].name != "deploy box" {
		t.Errorf("the custom name came back as %q", h2.a.ws.tabs[0].name)
	}

	if n := len(h2.a.ws.tabs[0].root.panes(nil)); n != 2 {
		t.Errorf("the split tab came back with %d panes, want 2", n)
	}

	moved := h2.a.ws.tabs[2]
	h2.a.chrome.reorder(h2.a.App, moved, 1)

	if h2.a.ws.tabs[0] != moved {
		t.Error("dragging the last tab to the start didn't move it")
	}
}
