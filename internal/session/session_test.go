package session

import (
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/plattyssh/vt"

	"github.com/plattyssh/platty/internal/sshx"
	"github.com/plattyssh/platty/internal/sshx/sshtest"
	"github.com/plattyssh/platty/internal/vault"
)

func screen(t *vt.Terminal) string {
	var f vt.Frame
	t.Snapshot(&f, 0)
	cols, rows := t.Size()
	return t.TextRange(vt.Pos{Line: f.TopLine - f.Scrollback}, vt.Pos{Line: f.TopLine + rows - 1, Col: cols - 1})
}

func waitFor(t *testing.T, s *Session, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		if strings.Contains(screen(s.Terminal()), want) {
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("%q didn't show up in:\n%s", want, screen(s.Terminal()))
}

func TestSSHSessionLifecycle(t *testing.T) {
	addr := sshtest.Server(t, nil)
	h, p, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(p)

	v, _ := vault.OpenAt(filepath.Join(t.TempDir(), "vault.dat"))
	host := vault.Host{Address: h, Port: port, Username: sshtest.User, Auth: vault.AuthPassword, Password: sshtest.Password}

	s := New(Options{
		Kind:   SSH,
		Target: sshx.Target{Host: host},
		Vault:  v,
		Hooks:  sshx.Hooks{ConfirmHostKey: func(sshx.HostKeyInfo) bool { return true }},
		Cols:   80,
		Rows:   24,
	})
	defer s.Close()

	waitFor(t, s, "welcome")

	if st, _ := s.State(); st != Connected {
		t.Fatalf("state %v", st)
	}

	if s.Client() == nil {
		t.Fatal("no SSH client for SFTP")
	}

	s.Send([]byte("eco"))
	waitFor(t, s, "$ eco")

	s.Client().Close()
	waitFor(t, s, "Press Enter to reconnect")

	s.Send([]byte("\r"))
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		if st, _ := s.State(); st == Connected {
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("Enter didn't reconnect")
}

func TestSSHSessionAuthFailure(t *testing.T) {
	addr := sshtest.Server(t, nil)
	h, p, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(p)

	v, _ := vault.OpenAt(filepath.Join(t.TempDir(), "vault.dat"))
	s := New(Options{
		Kind:   SSH,
		Target: sshx.Target{Host: vault.Host{Address: h, Port: port, Username: "otro", Auth: vault.AuthPassword, Password: "mal"}},
		Vault:  v,
		Hooks:  sshx.Hooks{ConfirmHostKey: func(sshx.HostKeyInfo) bool { return true }},
		Cols:   80,
		Rows:   24,
	})
	defer s.Close()

	waitFor(t, s, "Press Enter to retry")

	if st, err := s.State(); st != Failed || err == nil {
		t.Fatalf("state %v err %v", st, err)
	}
}

type memLog struct {
	strings.Builder
	closed bool
}

func (m *memLog) Close() error {
	m.closed = true
	return nil
}

func TestLogAndMirror(t *testing.T) {
	addr := sshtest.Server(t, nil)
	h, p, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(p)

	v, _ := vault.OpenAt(filepath.Join(t.TempDir(), "vault.dat"))
	host := vault.Host{Address: h, Port: port, Username: sshtest.User, Auth: vault.AuthPassword, Password: sshtest.Password}

	s := New(Options{
		Kind:   SSH,
		Target: sshx.Target{Host: host},
		Vault:  v,
		Hooks:  sshx.Hooks{ConfirmHostKey: func(sshx.HostKeyInfo) bool { return true }},
		Cols:   80,
		Rows:   24,
	})
	defer s.Close()

	log := &memLog{}
	s.StartLog(log)
	waitFor(t, s, "welcome")
	s.writeLog([]byte("\x1b[31mred\x1b[0m text\r\n"))

	var mirrored []byte
	s.SetMirror(func(p []byte) { mirrored = append(mirrored, p...) })
	s.Send([]byte("ls\r"))

	if string(mirrored) != "ls\r" {
		t.Errorf("mirror got %q", mirrored)
	}

	if !s.Logging() {
		t.Fatal("the session isn't logging")
	}

	s.StopLog()

	if got := log.String(); !strings.Contains(got, "welcome") || !strings.Contains(got, "red text\r\n") || strings.Contains(got, "\x1b") {
		t.Errorf("log: %q", got)
	}

	if !log.closed {
		t.Error("StopLog didn't close the file")
	}
}

func TestStartupCommand(t *testing.T) {
	addr := sshtest.Server(t, nil)
	h, p, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(p)

	v, _ := vault.OpenAt(filepath.Join(t.TempDir(), "vault.dat"))
	host := vault.Host{Address: h, Port: port, Username: sshtest.User, Auth: vault.AuthPassword, Password: sshtest.Password, StartupCommand: "tmux attach"}

	s := New(Options{
		Kind:   SSH,
		Target: sshx.Target{Host: host},
		Vault:  v,
		Hooks:  sshx.Hooks{ConfirmHostKey: func(sshx.HostKeyInfo) bool { return true }},
		Cols:   80,
		Rows:   24,
	})
	defer s.Close()

	waitFor(t, s, "tmux attach")
}
