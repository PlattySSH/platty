package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gioui.org/gpu/headless"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	platty "github.com/plattyssh/platty/internal/app"
	"github.com/plattyssh/platty/internal/sshx/sshtest"
	"github.com/plattyssh/platty/internal/vault"
)

type scenario struct {
	name  string
	theme string
	setup func(p *platty.Preview)
}

func main() {
	out := flag.String("o", "screens", "output directory")
	width := flag.Int("w", 1440, "ancho")
	height := flag.Int("h", 900, "alto")
	only := flag.String("only", "", "render only this scenario")
	flag.Parse()

	if err := run(*out, *width, *height, *only); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func sampleVault(dir string) (*vault.Vault, error) {
	v, err := vault.OpenAt(filepath.Join(dir, "vault.dat"))

	if err != nil {
		return nil, err
	}

	hosts := []vault.Host{
		{Label: "web-01", Address: "10.0.1.11", Username: "deploy", Group: "Production/Web", Color: 0, Auth: vault.AuthKey},
		{Label: "web-02", Address: "10.0.1.12", Username: "deploy", Group: "Production/Web", Color: 0, Auth: vault.AuthKey},
		{Label: "postgres-main", Address: "10.0.2.5", Username: "postgres", Group: "Production", Color: 3, Auth: vault.AuthKey},
		{Label: "bastion", Address: "bastion.example.com", Username: "ana", Group: "Production", Color: 4, Auth: vault.AuthAgent},
		{Label: "staging", Address: "staging.example.com", Username: "ana", Group: "Staging", Color: 2, Auth: vault.AuthPassword},
		{Label: "raspberry", Address: "192.168.1.40", Username: "pi", Group: "Home", Color: 1, Auth: vault.AuthPassword},
		{Label: "nas", Address: "192.168.1.10", Username: "admin", Group: "Home", Color: 5, Auth: vault.AuthPassword},
	}

	for i, h := range hosts {
		h.LastUsed = time.Now().Add(-time.Duration(i*37) * time.Minute)
		v.SaveHost(h)
	}

	v.CreateFolder("Clients/Acme")

	v.SaveKey(vault.Key{Label: "laptop-ed25519", Type: "ssh-ed25519", Public: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOmV4t3hXl0dZqv0fTxS0QGQb9b3vGfNnJQ0q6f7wz9P laptop"})
	v.SaveKey(vault.Key{Label: "github-deploy", Type: "ssh-rsa", Public: ""})
	v.SaveSnippet(vault.Snippet{Label: "Restart nginx", Command: "sudo systemctl restart nginx"})
	v.SaveSnippet(vault.Snippet{Label: "API logs", Command: "journalctl -u api -f --since '10 min ago'"})
	v.SaveSnippet(vault.Snippet{Label: "Disk usage", Command: "df -h | sort -k5 -r | head"})
	return v, nil
}

var testHost vault.Host

func scenarios() []scenario {
	demo := func(p *platty.Preview) {
		runScript(p, "web-01", gitSession, false)
	}

	return []scenario{
		{"welcome-carbon", "carbon", func(p *platty.Preview) { p.OpenTool(0) }},
		{"welcome-paper", "paper", func(p *platty.Preview) { p.OpenTool(0) }},
		{"terminal-carbon", "carbon", func(p *platty.Preview) {
			p.OpenTool(0)
			demo(p)
		}},
		{"split-midnight", "midnight", func(p *platty.Preview) {
			p.OpenTool(3)
			demo(p)
			runScript(p, "logs", logSession, true)
		}},
		{"palette-carbon", "carbon", func(p *platty.Preview) {
			p.OpenTool(0)
			demo(p)
			p.OpenPalette("")
		}},
		{"host-editor-carbon", "carbon", func(p *platty.Preview) {
			p.OpenTool(0)
			p.ShowHostEditor(vault.Host{Label: "web-03", Address: "10.0.1.13", Username: "deploy", Group: "Production", Port: 22, Auth: vault.AuthPassword})
		}},
		{"ssh-hostkey-carbon", "carbon", func(p *platty.Preview) {
			p.OpenTool(0)
			p.ConnectTrusted(testHost, false)
		}},
		{"ssh-sftp-carbon", "carbon", func(p *platty.Preview) {
			p.OpenTool(2)
			p.ConnectTrusted(testHost, true)
			p.Wait(800 * time.Millisecond)
			p.FakeStats()
			p.OpenTool(5)
		}},
		{"ssh-stats-carbon", "carbon", func(p *platty.Preview) {
			p.OpenTool(0)
			p.ConnectTrusted(testHost, true)
			p.Wait(800 * time.Millisecond)
			p.FakeStats()
		}},
		{"settings-carbon", "carbon", func(p *platty.Preview) { p.ShowSettings(1) }},
		{"forwards-carbon", "carbon", func(p *platty.Preview) { p.OpenTool(2) }},
		{"snippets-carbon", "carbon", func(p *platty.Preview) { p.OpenTool(3) }},
		{"known-carbon", "carbon", func(p *platty.Preview) { p.OpenTool(4) }},
		{"keychain-graphite", "graphite", func(p *platty.Preview) {
			p.OpenTool(1)
			p.Toast("Public key copied", "Paste it into ~/.ssh/authorized_keys on the server.")
		}},
	}
}

func run(out string, width, height int, only string) error {
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp("", "platty-preview")

	if err != nil {
		return err
	}

	defer os.RemoveAll(tmp)

	v, err := sampleVault(tmp)

	if err != nil {
		return err
	}

	addr, stop, err := sshtest.ListenDemo()

	if err != nil {
		return err
	}

	defer stop()

	h, port, _ := net.SplitHostPort(addr)
	pn, _ := strconv.Atoi(port)
	testHost, _ = v.SaveHost(vault.Host{Label: "demo-server", Address: h, Port: pn, Username: sshtest.User,
		Auth: vault.AuthPassword, Password: sshtest.Password, Group: "Staging", Color: 2})
	v.SaveForward(vault.Forward{Label: "Postgres prod", HostID: testHost.ID, Kind: vault.ForwardLocal, BindPort: 15432, DestHost: "localhost", DestPort: 5432})
	v.SaveForward(vault.Forward{Label: "Proxy SOCKS", HostID: testHost.ID, Kind: vault.ForwardDynamic, BindPort: 1080})

	win, err := headless.NewWindow(width, height)

	if err != nil {
		return err
	}

	defer win.Release()

	for _, sc := range scenarios() {
		if only != "" && sc.name != only {
			continue
		}

		p := platty.NewPreview(v, sc.theme)
		sc.setup(p)

		gtx := layout.Context{
			Ops:         new(op.Ops),
			Constraints: layout.Exact(image.Pt(width, height)),
			Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
			Now:         time.Now(),
		}

		for i := 0; i < 4; i++ {
			gtx.Ops.Reset()
			gtx.Now = time.Now()
			p.Frame(gtx)
			win.Frame(gtx.Ops)
			p.Wait(600 * time.Millisecond)
		}

		gtx.Ops.Reset()
		p.Frame(gtx)

		if err := win.Frame(gtx.Ops); err != nil {
			return err
		}

		img := image.NewRGBA(image.Rect(0, 0, width, height))

		if err := win.Screenshot(img); err != nil {
			return err
		}

		f, err := os.Create(filepath.Join(out, sc.name+".png"))

		if err != nil {
			return err
		}

		png.Encode(f, img)
		f.Close()
		p.Close()
		fmt.Println("ok", sc.name)
	}

	return nil
}
