package backup

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/plattyssh/platty/internal/vault"
)

func sample() vault.Data {
	return vault.Data{
		Version:    1,
		Hosts:      []vault.Host{{ID: "h1", Address: "10.0.0.1", Password: "hunter2", Group: "Prod"}},
		Keys:       []vault.Key{{ID: "k1", Label: "laptop", Private: "PRIVATE", Passphrase: "pass"}},
		Forwards:   []vault.Forward{{ID: "f1", HostID: "h1"}},
		Snippets:   []vault.Snippet{{ID: "s1", Command: "uptime"}},
		KnownHosts: []vault.KnownHost{{Host: "10.0.0.1:22", Key: "AAAA"}},
		Folders:    []string{"Prod"},
	}
}

func TestRoundTrip(t *testing.T) {
	all := Selection{Hosts: true, Keys: true, Secrets: true, Forwards: true, Snippets: true, KnownHosts: true}
	raw, err := Export(sample(), all, "a long enough password")

	if err != nil {
		t.Fatal(err)
	}

	for _, secret := range []string{"hunter2", "PRIVATE", "uptime"} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Errorf("%q is readable in the export file", secret)
		}
	}

	if _, err := Import(raw, "the wrong password"); err != ErrWrongPassword {
		t.Errorf("wrong password: %v", err)
	}

	d, err := Import(raw, "a long enough password")

	if err != nil {
		t.Fatal(err)
	}

	if d.Hosts[0].Password != "hunter2" || d.Keys[0].Passphrase != "pass" || len(d.Forwards) != 1 || len(d.KnownHosts) != 1 {
		t.Errorf("imported data: %+v", d)
	}
}

func TestSelection(t *testing.T) {
	d := Filter(sample(), Selection{Hosts: true, Keys: true})

	if d.Hosts[0].Password != "" || d.Keys[0].Passphrase != "" {
		t.Error("secrets were exported without Secrets")
	}

	if len(d.Forwards) != 0 || len(d.Snippets) != 0 || len(d.KnownHosts) != 0 {
		t.Error("unselected data was exported")
	}

	if s := sample(); s.Hosts[0].Password != "hunter2" {
		t.Error("Filter changed the original data")
	}

	if d := Filter(sample(), Selection{Forwards: true}); len(d.Forwards) != 0 {
		t.Error("forwarding rules were exported without their hosts")
	}

	if _, err := Export(sample(), Selection{Hosts: true}, "short"); err != ErrShortPassword {
		t.Errorf("short password: %v", err)
	}

	if p := GeneratePassword(); len(p) < MinPassword {
		t.Errorf("generated password %q is too short", p)
	}
}

func TestMergeIntoVault(t *testing.T) {
	v, err := vault.OpenAt(filepath.Join(t.TempDir(), "vault.dat"))

	if err != nil {
		t.Fatal(err)
	}

	v.TrustHost(vault.KnownHost{Host: "10.0.0.1:22", Key: "BBBB"})
	st, err := v.Merge(sample())

	if err != nil {
		t.Fatal(err)
	}

	if st.Hosts != 1 || st.Keys != 1 || st.Conflicts != 1 || st.KnownHosts != 0 {
		t.Errorf("stats: %+v", st)
	}

	if k, _ := v.KnownHost("10.0.0.1:22"); k.Key != "BBBB" {
		t.Error("an import replaced a trusted host key")
	}
}

func TestSSHConfig(t *testing.T) {
	hosts := []vault.Host{
		{ID: "a", Label: "Web Server", Address: "10.0.0.1", Username: "deploy", Port: 2222, Password: "hunter2"},
		{ID: "b", Label: "web server", Address: "10.0.0.2", JumpID: "a"},
	}
	got := string(SSHConfig(hosts))
	want := "Host web-server\n    HostName 10.0.0.1\n    User deploy\n    Port 2222\n\nHost web-server-2\n    HostName 10.0.0.2\n    ProxyJump web-server\n"

	if got != want {
		t.Fatalf("ssh config:\n%s\nwant:\n%s", got, want)
	}
}
