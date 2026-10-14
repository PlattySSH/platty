package app

import (
	"strconv"
	"testing"
	"time"

	"gioui.org/io/key"
	"gioui.org/io/pointer"

	"github.com/plattyssh/platty/internal/app/core"
	"github.com/plattyssh/platty/internal/app/hosts"
	"github.com/plattyssh/platty/internal/app/keychain"
	"github.com/plattyssh/platty/internal/app/snippets"
	"github.com/plattyssh/platty/internal/privacy"
	"github.com/plattyssh/platty/internal/session"
	"github.com/plattyssh/platty/internal/vault"
)

const (
	yNewTerm = 188
	xChevron = 228
	yFolder  = 268
)

func TestChevronDoesNotOpenTerminal(t *testing.T) {
	h := newHarness(t)
	h.click(xChevron, yNewTerm)
	h.shot("chevron")

	if n := len(h.a.ws.tabs); n != 0 {
		t.Errorf("the New terminal dropdown opened %d tabs", n)
	}

	if !h.a.ws.menu.IsOpen() {
		t.Error("the dropdown didn't open")
	}
}

func TestFolderClickCollapses(t *testing.T) {
	h := newHarness(t)
	h.click(84, yFolder)
	h.shot("folder-click")

	if !h.a.tools.hosts.Collapsed("Cabin") {
		t.Error("a click didn't collapse the folder")
	}

	time.Sleep(450 * time.Millisecond)
	h.click(84, yFolder)

	if h.a.tools.hosts.Collapsed("Cabin") {
		t.Error("the second click didn't expand it")
	}
}

func TestDragHostIntoNestedFolder(t *testing.T) {
	h := newHarness(t)
	h.drag(110, 333, 110, 429, true)
	h.shot("dragging")
	h.ptr(pointer.Release, 110, 429, 0)
	h.frames(4)
	h.a.tools.hosts.Refresh()

	for _, x := range h.a.tools.hosts.Hosts() {
		if x.Label == "raspberry" && x.Group != "Production" {
			t.Errorf("raspberry ended up in %q", x.Group)
		}
	}
}

func TestPaletteArrows(t *testing.T) {
	h := newHarness(t)
	h.key("P", key.ModCtrl|key.ModShift)
	h.shot("palette-0")
	before := h.a.palette.Selected()
	h.key(key.NameDownArrow, 0)
	h.key(key.NameDownArrow, 0)
	h.shot("palette-2")

	if h.a.palette.Selected() == before {
		t.Errorf("the arrows didn't move the selection (%d)", before)
	}
}

func TestHomeShot(t *testing.T) {
	h := newHarness(t)
	h.shot("home")
	h.move(235, 235)
	h.shot("hover-more")
	h.a.tools.open[core.Right] = 1
	h.frames(10)
	h.shot("keychain")
}

func TestTabs(t *testing.T) {
	h := newHarness(t)
	h.a.RunLocal("PowerShell", "cmd.exe", "/k", "echo uno")
	h.a.RunLocal("cmd", "cmd.exe", "/k", "echo dos")
	h.a.RunLocal("WSL", "cmd.exe", "/k", "echo tres")
	h.frames(20)
	h.shot("tabs")

	if h.a.ws.active != 2 {
		t.Fatalf("active %d, want the last one", h.a.ws.active)
	}

	h.click(260+10+40, 42)

	if h.a.ws.active != 0 {
		t.Errorf("clicking the first tab left %d active", h.a.ws.active)
	}

	if h.a.tools.hosts.ItemCount() == 0 {
		t.Error("the sidebar lost its hosts")
	}

	h.a.ws.closeTab(1)
	h.frames(20)
	h.shot("tabs-closed")

	if n := len(h.a.ws.tabs); n != 2 {
		t.Errorf("%d tabs left, want 2", n)
	}
}

func TestHomeHoverTyping(t *testing.T) {
	h := newHarness(t)
	h.frames(40)
	h.move(1010, 470)
	h.frames(5)
	h.shot("home-typing")

	if n := len(h.a.ws.tabs); n != 0 {
		t.Fatalf("hovering opened %d tabs", n)
	}
}

func TestFolderClickWithJitter(t *testing.T) {
	h := newHarness(t)
	h.move(84, yFolder)
	h.ptr(pointer.Press, 84, yFolder, pointer.ButtonPrimary)
	h.ptr(pointer.Move, 85, yFolder+1, pointer.ButtonPrimary)
	h.ptr(pointer.Move, 86, yFolder+2, pointer.ButtonPrimary)
	h.ptr(pointer.Release, 86, yFolder+2, 0)
	h.frames(4)

	if !h.a.tools.hosts.Collapsed("Cabin") {
		t.Error("a click with 2 px of jitter didn't collapse the folder")
	}
}

func TestFolderDoubleClickTogglesOnce(t *testing.T) {
	h := newHarness(t)
	h.move(84, yFolder)

	for i := 0; i < 2; i++ {
		h.ptr(pointer.Press, 84, yFolder, pointer.ButtonPrimary)
		h.ptr(pointer.Release, 84, yFolder, 0)
	}

	h.frames(4)

	if !h.a.tools.hosts.Collapsed("Cabin") {
		t.Error("the double click left the folder unchanged")
	}
}

func TestTabCloseButton(t *testing.T) {
	h := newHarness(t)
	h.a.RunLocal("PowerShell", "cmd.exe", "/k", "echo uno")
	h.a.RunLocal("cmd", "cmd.exe", "/k", "echo dos")
	h.a.ws.active = 0
	h.frames(20)
	h.move(330, 42)
	h.shot("tab-hover")
	h.click(448, 42)
	h.frames(20)

	if n := len(h.a.ws.tabs); n != 1 {
		t.Fatalf("%d tabs left, want 1", n)
	}

	if h.a.ws.active != 0 || h.a.ws.tabs[0].closing {
		t.Errorf("active is %d (closing: %v)", h.a.ws.active, h.a.ws.tabs[0].closing)
	}
}

func TestMoreMenu(t *testing.T) {
	h := newHarness(t)
	h.click(235, 235)
	h.shot("more-menu")

	if !h.a.tools.hosts.MoreMenuOpen() {
		t.Fatal("the ⋯ menu didn't open")
	}
}

func TestNewFolderInline(t *testing.T) {
	h := newHarness(t)
	h.click(183, 235)
	h.shot("folder-new")

	if !h.a.tools.hosts.EditingFolder() {
		t.Fatal("the row to type the name didn't show up")
	}

	if h.a.modal != nil {
		t.Fatal("a modal opened")
	}

	h.text("Backups")
	h.key(key.NameReturn, 0)
	h.frames(4)
	found := false

	for _, f := range h.a.vault.Snapshot().AllFolders() {
		found = found || f == "Backups"
	}

	if !found {
		t.Error("folder \"Backups\" wasn't created")
	}
}

func TestHostEditorDocked(t *testing.T) {
	h := newHarness(t)
	h.click(209, 235)
	h.frames(10)
	h.shot("host-docked")

	if h.a.modal != nil || h.a.docked[core.Right] == nil {
		t.Fatal("the host editor didn't dock in the side panel")
	}

	h.key(key.NameEscape, 0)

	if h.a.docked[core.Right] != nil {
		t.Error("Escape didn't close the form")
	}
}

func TestSnippetDocked(t *testing.T) {
	h := newHarness(t)
	h.a.OpenIn(h.a.tools.snippets, snippets.NewEditor(h.a.App, vault.Snippet{}))
	h.frames(10)
	h.shot("snippet-docked")

	if h.a.docked[core.Right] == nil {
		t.Fatal("the snippet editor didn't dock")
	}
}

func keyPos(t *testing.T, h *harness, label string) (float32, float32) {
	p, ok := h.a.welcome.Art.KeyPos(label)

	if !ok {
		t.Fatalf("no key %q", label)
	}

	return p.X, p.Y
}

func TestKeyboardTypesAndRuns(t *testing.T) {
	h := newHarness(t)
	h.frames(60)
	h.move(1100, 470)
	h.move(1101, 471)

	for _, l := range []string{"L", "S"} {
		x, y := keyPos(t, h, l)
		h.click(x, y)
	}

	h.shot("keyboard-typed")

	if got := h.a.welcome.Art.Typed(); got != "ls" {
		t.Fatalf("screen shows %q, want \"ls\"", got)
	}

	if n := len(h.a.ws.tabs); n != 0 {
		t.Fatalf("pressing keys opened %d tabs", n)
	}

	x, y := keyPos(t, h, "enter")
	h.click(x, y)

	if n := len(h.a.ws.tabs); n != 0 {
		t.Fatalf("Enter opened %d tabs; only the screen should open one", n)
	}

	if got := h.a.welcome.Art.Lines(); len(got) != 1 || got[0] != "$ ls" {
		t.Fatalf("after Enter the screen has %q", got)
	}

	r := &h.a.welcome.Art
	box := r.CasePos()
	h.click(box.X, box.Y)

	if n := len(h.a.ws.tabs); n != 0 {
		t.Fatalf("clicking the case opened %d tabs", n)
	}

	scr := r.ScreenPos()
	h.click(scr.X, scr.Y)

	if n := len(h.a.ws.tabs); n != 1 {
		t.Errorf("clicking the screen opened %d tabs, want 1", n)
	}
}

func TestNewFolderCancel(t *testing.T) {
	h := newHarness(t)
	h.click(183, 235)

	if !h.a.tools.hosts.EditingFolder() {
		t.Fatal("the row didn't show up")
	}

	h.key(key.NameEscape, 0)

	if h.a.tools.hosts.EditingFolder() {
		t.Error("Escape didn't cancel the new folder")
	}

	h.click(183, 235)
	h.click(700, 700)

	if h.a.tools.hosts.EditingFolder() {
		t.Error("clicking outside didn't cancel the empty new folder")
	}
}

func TestBrowserTabsSpinner(t *testing.T) {
	h := newHarness(t)
	h.a.RunLocal("PowerShell", "cmd.exe", "/k", "echo uno")
	h.a.RunLocal("cmd", "cmd.exe", "/k", "echo dos")
	h.a.Connect(vault.Host{Label: "slow", Address: "10.255.255.1", Port: 22, Username: "root"})
	h.frames(20)
	h.shot("tabs-browser")
	st, _ := h.a.ws.activePane().sess.State()

	if st != session.Connecting {
		t.Skipf("the session isn't connecting (%v)", st)
	}

	h.frames(8)
	h.shot("connecting")
	h.frames(16)
	h.shot("connecting2")

	if h.a.ws.activePane().connecting.start.IsZero() {
		t.Error("the connecting screen wasn't drawn")
	}
}

func TestKeyRename(t *testing.T) {
	h := newHarness(t)
	k, err := sshxGenerate("my-key")

	if err != nil {
		t.Fatal(err)
	}

	k, _ = h.a.vault.SaveKey(k)
	h.a.tools.open[core.Right] = 1
	h.frames(12)
	h.shot("keys-list")
	h.rclick(1250, 120)
	h.shot("keys-menu")

	if !h.a.tools.keys.MenuOpen() {
		t.Fatal("right click didn't open the key menu")
	}

	h.click(1280, 173)
	h.frames(6)
	h.shot("keys-rename")

	if h.a.tools.keys.Renaming() != k.ID {
		t.Fatal("Rename didn't start editing the name")
	}

	if sel := h.a.tools.keys.RenameEditor().SelectedText(); sel != "my-key" {
		t.Fatalf("selected %q, want the whole name", sel)
	}

	h.a.tools.keys.RenameEditor().Insert("laptop")
	h.frames(2)
	h.key(key.NameReturn, 0)
	h.frames(6)
	got, _ := h.a.vault.Key(k.ID)

	if got.Label != "laptop" {
		t.Errorf("the key is named %q", got.Label)
	}
}

func TestKeyGeneratedView(t *testing.T) {
	h := newHarness(t)
	h.a.OpenIn(h.a.tools.keys, keychain.NewGenerateKey(h.a.App))
	h.frames(8)
	h.shot("keygen-form")
	h.text("laptop")
	h.key(key.NameReturn, 0)
	h.frames(10)
	h.shot("keygen-done")
}

func TestHostEditorKey(t *testing.T) {
	h := newHarness(t)
	k, _ := sshxGenerate("laptop")
	h.a.vault.SaveKey(k)
	e := hosts.NewEditor(h.a.App, hosts.Defaults())
	e.SetAuth(1)
	h.a.OpenDocked(core.Right, e)
	h.frames(10)
	h.shot("host-key")

	if e.KeyIndex() != 1 {
		t.Errorf("with SSH key the key wasn't chosen (Value=%d)", e.KeyIndex())
	}
}

func TestKeyRowHover(t *testing.T) {
	h := newHarness(t)
	k, _ := sshxGenerate("a-key-with-a-very-long-name")
	h.a.vault.SaveKey(k)
	k2, _ := sshxGenerate("short")
	h.a.vault.SaveKey(k2)
	h.a.tools.open[core.Right] = 1
	h.frames(12)
	h.move(1250, 120)
	h.frames(6)
	h.shot("keys-hover")
}

func TestScreenRunsDemoSSH(t *testing.T) {
	h := newHarness(t)
	h.frames(60)
	h.move(1100, 470)
	h.move(1101, 471)
	r := &h.a.welcome.Art
	scr := r.ScreenPos()
	h.click(scr.X, scr.Y)

	if len(h.a.ws.tabs) != 1 {
		t.Fatalf("%d tabs opened", len(h.a.ws.tabs))
	}

	if k := h.a.ws.activePane().sess.Kind; k != session.SSH {
		t.Errorf("opened a %v session, want SSH to the recent host", k)
	}
}

func TestHideAddresses(t *testing.T) {
	if got := privacy.RedactIPs("root@203.0.113.22:22 and 127.0.0.1:5432 and fe80:0:0:0:1:2:3:4"); got != "root@•••.•••.•••.•••:22 and 127.0.0.1:5432 and ••••:••••::••••" {
		t.Errorf("redactIPs = %q", got)
	}

	h := newHarness(t)
	h.a.vault.SaveHost(vault.Host{Address: "203.0.113.22", Username: "root", LastUsed: time.Now()})
	h.a.settings.HideAddresses = true
	h.frames(30)
	h.shot("hide-ips")
}

func TestPaletteScrollsOnlyAtEdge(t *testing.T) {
	h := newHarness(t)
	h.key("P", key.ModCtrl|key.ModShift)
	h.frames(4)

	for i := 0; i < 3; i++ {
		h.key(key.NameDownArrow, 0)
	}

	if first := h.a.palette.Scroll().First; first != 0 {
		t.Fatalf("the list scrolled after only 3 arrows (First=%d)", first)
	}

	for i := 0; i < 20; i++ {
		h.key(key.NameDownArrow, 0)
	}

	h.shot("palette-scrolled")
	pos := h.a.palette.Scroll()

	if sel := h.a.palette.Selected(); sel < pos.First || sel >= pos.First+pos.Count {
		t.Errorf("selection %d isn't visible (First=%d Count=%d)", sel, pos.First, pos.Count)
	}
}

func TestTabOverflow(t *testing.T) {
	h := newHarness(t)

	for i := 0; i < 12; i++ {
		h.a.RunLocal("term "+strconv.Itoa(i), "cmd.exe", "/k", "echo "+strconv.Itoa(i))
	}

	h.frames(20)
	h.shot("tabs-overflow")
	c := &h.a.chrome

	if len(c.hiddenTabs) == 0 {
		t.Fatal("with 12 tabs none went to the dropdown")
	}

	for _, i := range c.hiddenTabs {
		if i == h.a.ws.active {
			t.Fatal("the active tab got hidden")
		}
	}

	first := c.hiddenTabs[0]
	c.moreTabs.Click()
	h.frames(4)
	h.shot("tabs-overflow-menu")

	if !c.tabMenu.IsOpen() {
		t.Fatal("the arrow didn't open the dropdown")
	}

	h.key(key.NameDownArrow, 0)
	h.key(key.NameReturn, 0)
	h.frames(4)

	if h.a.ws.active != first {
		t.Errorf("active %d, want the hidden one %d", h.a.ws.active, first)
	}
}

func TestMenuKeyboard(t *testing.T) {
	h := newHarness(t)
	h.click(235, 235)

	if !h.a.tools.hosts.MoreMenuOpen() {
		t.Fatal("it didn't open")
	}

	h.key(key.NameDownArrow, 0)
	h.key(key.NameDownArrow, 0)
	h.key(key.NameReturn, 0)
	h.frames(4)

	if !h.a.tools.hosts.EditingFolder() {
		t.Error("↓↓ Enter in the menu didn't pick \"New folder…\"")
	}
}

func TestStatusBarHover(t *testing.T) {
	h := newHarness(t)
	h.a.Connect(vault.Host{Label: "slow", Address: "10.255.255.1", Port: 22, Username: "root"})
	h.frames(4)
	h.a.FakeStats()
	h.frames(4)
	h.shot("statusbar")
	h.move(470, 885)
	h.frames(4)
	h.shot("statusbar-hover")

	if h.a.chrome.metrics.Items() == 0 {
		t.Fatal("the bar drew no metrics")
	}
}

func TestSidebarSearchClick(t *testing.T) {
	h := newHarness(t)
	h.click(130, 84)

	if !h.a.palette.IsOpen() {
		t.Error("clicking the sidebar search didn't open the palette")
	}
}
