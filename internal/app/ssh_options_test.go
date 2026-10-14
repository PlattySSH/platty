package app

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/plattyssh/platty/internal/app/core"
	"github.com/plattyssh/platty/internal/app/hosts"
	"github.com/plattyssh/platty/internal/sshx"
	"github.com/plattyssh/platty/internal/vault"
)

func TestSSHOptionsShots(t *testing.T) {
	h := newHarness(t)
	key, _ := sshx.GenerateEd25519("ci-deploy", "")
	_, caPriv, _ := ed25519.GenerateKey(rand.Reader)
	ca, _ := ssh.NewSignerFromKey(caPriv)
	pub, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(key.Public))
	cert := &ssh.Certificate{Key: pub, CertType: ssh.UserCert, ValidBefore: ssh.CertTimeInfinity}
	cert.SignCert(rand.Reader, ca)
	key.Certificate = string(ssh.MarshalAuthorizedKey(cert))
	h.a.vault.SaveKey(key)

	h.a.tools.open[core.Right] = 1
	h.frames(10)
	h.shot("keychain-cert")

	e := hosts.NewEditor(h.a.App, vault.Host{Label: "build", Address: "10.0.0.9", Username: "ci", StartupCommand: "tmux attach", ForwardAgent: true})
	h.a.OpenDocked(core.Right, e)
	h.frames(10)
	h.shot("host-advanced")
}
