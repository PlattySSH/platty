package app

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/plattyssh/platty/internal/session"
	"github.com/plattyssh/platty/internal/sshx/sshtest"
	"github.com/plattyssh/platty/internal/transfer"
	"github.com/plattyssh/platty/internal/vault"
)

func TestSFTPQueue(t *testing.T) {
	addr, stop, err := sshtest.ListenDemo()

	if err != nil {
		t.Fatal(err)
	}

	defer stop()

	h := newHarness(t)
	host, port, _ := net.SplitHostPort(addr)
	pn, _ := strconv.Atoi(port)
	saved, _ := h.a.vault.SaveHost(vault.Host{Label: "demo", Address: host, Port: pn, Username: sshtest.User, Auth: vault.AuthPassword, Password: sshtest.Password})
	h.a.ConnectTrusted(saved, true)

	if !waitFor(h, func() bool {
		p := h.a.ws.activePane()
		st, _ := p.sess.State()
		return st == session.Connected
	}) {
		t.Fatal("the demo session didn't connect")
	}

	h.a.tools.show(h.a.App, h.a.tools.sftp)

	dir := filepath.Join(t.TempDir(), "site")
	os.MkdirAll(filepath.Join(dir, "css"), 0o755)
	os.WriteFile(filepath.Join(dir, "index.html"), bytes.Repeat([]byte("<p>"), 50_000), 0o644)
	os.WriteFile(filepath.Join(dir, "css", "app.css"), []byte("body{}"), 0o644)

	if !waitFor(h, func() bool { return h.a.tools.sftp.Ready(h.a.ActiveSession().ID) }) {
		t.Fatal("SFTP never opened")
	}

	go h.a.tools.sftp.UploadDropped(h.a.ActiveSession().ID, []string{dir})

	if !waitFor(h, func() bool {
		jobs := h.a.transfers.Jobs()
		return len(jobs) == 1 && jobs[0].Finished()
	}) {
		t.Fatal("the upload never finished")
	}

	j := h.a.transfers.Jobs()[0]

	if j.State() != transfer.Done || j.Files() != 2 {
		t.Fatalf("upload: state %v, %d files, %v", j.State(), j.Files(), j.Err())
	}

	h.frames(10)
	h.shot("sftp-queue")
}
