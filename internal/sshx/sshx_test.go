package sshx

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/plattyssh/platty/internal/sshx/sshtest"
	"github.com/plattyssh/platty/internal/vault"
)

const (
	testUser = sshtest.User
	testPass = sshtest.Password
)

func testServer(t *testing.T, authorized ssh.PublicKey) string {
	return sshtest.Server(t, authorized)
}

func testVault(t *testing.T) *vault.Vault {
	v, err := vault.OpenAt(filepath.Join(t.TempDir(), "vault.dat"))

	if err != nil {
		t.Fatal(err)
	}

	return v
}

func hostFor(addr string) vault.Host {
	h, p, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(p)
	return vault.Host{Address: h, Port: port, Username: testUser, Auth: vault.AuthPassword, Password: testPass}
}

func TestDialPasswordShellAndTOFU(t *testing.T) {
	addr := testServer(t, nil)
	v := testVault(t)

	asked := 0
	hooks := Hooks{ConfirmHostKey: func(HostKeyInfo) bool { asked++; return true }}

	c, err := Dial(context.Background(), Target{Host: hostFor(addr)}, v, hooks, 0)

	if err != nil {
		t.Fatal(err)
	}

	defer c.Close()

	sh, err := OpenShell(c, 80, 24, false)

	if err != nil {
		t.Fatal(err)
	}

	defer sh.Close()

	sh.Write([]byte("hello"))
	got := readUntil(t, sh, "hello")

	if !strings.Contains(got, "welcome") {
		t.Fatalf("output: %q", got)
	}

	c2, err := Dial(context.Background(), Target{Host: hostFor(addr)}, v, hooks, 0)

	if err != nil {
		t.Fatal(err)
	}

	c2.Close()

	if asked != 1 {
		t.Fatalf("asked %d times about the host key; want 1", asked)
	}
}

func TestRejectHostKey(t *testing.T) {
	addr := testServer(t, nil)
	v := testVault(t)

	_, err := Dial(context.Background(), Target{Host: hostFor(addr)}, v, Hooks{ConfirmHostKey: func(HostKeyInfo) bool { return false }}, 0)

	if err == nil {
		t.Fatal("connected even though the key was rejected")
	}
}

func TestDialWithGeneratedKey(t *testing.T) {
	k, err := GenerateEd25519("test", "frase")

	if err != nil {
		t.Fatal(err)
	}

	pub, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(k.Public))
	addr := testServer(t, pub)
	v := testVault(t)

	h := hostFor(addr)
	h.Auth, h.Password = vault.AuthKey, ""
	c, err := Dial(context.Background(), Target{Host: h, Key: &k}, v, Hooks{ConfirmHostKey: func(HostKeyInfo) bool { return true }}, 0)

	if err != nil {
		t.Fatal(err)
	}

	c.Close()
}

func TestSFTPRoundTrip(t *testing.T) {
	addr := testServer(t, nil)
	v := testVault(t)
	c, err := Dial(context.Background(), Target{Host: hostFor(addr)}, v, Hooks{ConfirmHostKey: func(HostKeyInfo) bool { return true }}, 0)

	if err != nil {
		t.Fatal(err)
	}

	defer c.Close()

	s, err := OpenSFTP(c)

	if err != nil {
		t.Fatal(err)
	}

	defer s.Close()

	dir := t.TempDir()
	dst := filepath.ToSlash(filepath.Join(dir, "uploaded.txt"))

	if _, err := s.Upload(strings.NewReader("content"), dst); err != nil {
		t.Fatal(err)
	}

	entries, err := s.List(filepath.ToSlash(dir))

	if err != nil || len(entries) != 1 || entries[0].Name != "uploaded.txt" {
		t.Fatalf("list: %+v %v", entries, err)
	}

	var buf bytes.Buffer
	s.Download(dst, &buf)

	if buf.String() != "content" {
		t.Fatalf("download: %q", buf.String())
	}
}

func TestLocalAndDynamicForward(t *testing.T) {
	addr := testServer(t, nil)
	v := testVault(t)
	c, err := Dial(context.Background(), Target{Host: hostFor(addr)}, v, Hooks{ConfirmHostKey: func(HostKeyInfo) bool { return true }}, 0)

	if err != nil {
		t.Fatal(err)
	}

	defer c.Close()

	echo, _ := net.Listen("tcp", "127.0.0.1:0")
	defer echo.Close()
	go func() {
		for {
			conn, err := echo.Accept()

			if err != nil {
				return
			}

			conn.Write([]byte("pong"))
			conn.Close()
		}
	}()
	eh, ep, _ := net.SplitHostPort(echo.Addr().String())
	destPort, _ := strconv.Atoi(ep)

	localPort := freePort(t)
	tun, err := StartForward(c, vault.Forward{Kind: vault.ForwardLocal, BindPort: localPort, DestHost: eh, DestPort: destPort})

	if err != nil {
		t.Fatal(err)
	}

	defer tun.Close()

	conn, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(localPort)))

	if err != nil {
		t.Fatal(err)
	}

	if b, _ := io.ReadAll(conn); string(b) != "pong" {
		t.Fatalf("local forward: %q", b)
	}

	socksPort := freePort(t)
	dyn, err := StartForward(c, vault.Forward{Kind: vault.ForwardDynamic, BindPort: socksPort})

	if err != nil {
		t.Fatal(err)
	}

	defer dyn.Close()

	sc, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(socksPort)))

	if err != nil {
		t.Fatal(err)
	}

	sc.Write([]byte{5, 1, 0})
	io.ReadFull(sc, make([]byte, 2))
	req := []byte{5, 1, 0, 1}
	req = append(req, net.ParseIP(eh).To4()...)
	req = binary.BigEndian.AppendUint16(req, uint16(destPort))
	sc.Write(req)
	io.ReadFull(sc, make([]byte, 10))

	if b, _ := io.ReadAll(sc); string(b) != "pong" {
		t.Fatalf("socks: %q", b)
	}
}

func freePort(t *testing.T) int {
	l, err := net.Listen("tcp", "127.0.0.1:0")

	if err != nil {
		t.Fatal(err)
	}

	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func readUntil(t *testing.T, r io.Reader, want string) string {
	t.Helper()

	var got bytes.Buffer
	buf := make([]byte, 256)
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		n, err := r.Read(buf)
		got.Write(buf[:n])

		if strings.Contains(got.String(), want) || err != nil {
			break
		}
	}

	return got.String()
}

func TestCertificateSigners(t *testing.T) {
	key, err := GenerateEd25519("user", "")

	if err != nil {
		t.Fatal(err)
	}

	_, caPriv, _ := ed25519.GenerateKey(rand.Reader)
	ca, _ := ssh.NewSignerFromKey(caPriv)
	pub, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(key.Public))
	cert := &ssh.Certificate{Key: pub, CertType: ssh.UserCert, ValidPrincipals: []string{"deploy"}, ValidBefore: ssh.CertTimeInfinity}

	if err := cert.SignCert(rand.Reader, ca); err != nil {
		t.Fatal(err)
	}

	text := string(ssh.MarshalAuthorizedKey(cert))

	if err := CheckCertificate(key, text); err != nil {
		t.Fatalf("check: %v", err)
	}

	other, _ := GenerateEd25519("other", "")

	if err := CheckCertificate(other, text); err != ErrCertMismatch {
		t.Errorf("a certificate for another key passed: %v", err)
	}

	key.Certificate = text
	signers, err := Signers(key, Hooks{})

	if err != nil || len(signers) != 2 {
		t.Fatalf("signers: %d %v", len(signers), err)
	}

	if _, ok := signers[0].PublicKey().(*ssh.Certificate); !ok {
		t.Error("the certificate isn't offered first")
	}
}
