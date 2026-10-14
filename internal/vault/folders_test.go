package vault

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestFolders(t *testing.T) {
	v, err := OpenAt(filepath.Join(t.TempDir(), "vault.dat"))

	if err != nil {
		t.Fatal(err)
	}

	web, _ := v.SaveHost(Host{Label: "web", Address: "10.0.0.1", Group: "Prod/Web"})
	db, _ := v.SaveHost(Host{Label: "db", Address: "10.0.0.2", Group: "Prod"})
	v.CreateFolder(" Cabin / NAS ")

	got := v.Snapshot().AllFolders()
	want := []string{"Cabin", "Cabin/NAS", "Prod", "Prod/Web"}

	if !slices.Equal(got, want) {
		t.Fatalf("AllFolders = %v, want %v", got, want)
	}

	v.RenameFolder("Prod", "Production")

	if h, _ := v.Host(web.ID); h.Group != "Production/Web" {
		t.Fatalf("rename: group = %q", h.Group)
	}

	v.RenameFolder("Production", "Production/Web/Otra")

	if h, _ := v.Host(db.ID); h.Group != "Production" {
		t.Fatalf("cycle: group = %q", h.Group)
	}

	v.DeleteFolder("Production/Web")

	if h, _ := v.Host(web.ID); h.Group != "Production" {
		t.Fatalf("delete: the host should move up to the parent folder, group = %q", h.Group)
	}

	v.MoveHost(db.ID, "Cabin/NAS")

	if h, _ := v.Host(db.ID); h.Group != "Cabin/NAS" {
		t.Fatalf("move: group = %q", h.Group)
	}

	got = v.Snapshot().AllFolders()
	want = []string{"Cabin", "Cabin/NAS", "Production"}

	if !slices.Equal(got, want) {
		t.Fatalf("AllFolders after changes = %v, want %v", got, want)
	}
}
