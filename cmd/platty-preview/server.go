package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	demoUser     = "platty"
	demoPassword = "hunter2"
)

func listenDemo() (addr string, stop func(), err error) {
	return listen(demoFS())
}

func listen(mem *sftp.Handlers) (addr string, stop func(), err error) {
	_, hostPriv, _ := ed25519.GenerateKey(rand.Reader)
	hostSigner, _ := ssh.NewSignerFromKey(hostPriv)

	cfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == demoUser && string(pass) == demoPassword {
				return nil, nil
			}

			return nil, fmt.Errorf("wrong password")
		},
	}
	cfg.AddHostKey(hostSigner)

	l, err := net.Listen("tcp", "127.0.0.1:0")

	if err != nil {
		return "", nil, err
	}

	go func() {
		for {
			conn, err := l.Accept()

			if err != nil {
				return
			}

			go serveConn(conn, cfg, mem)
		}
	}()

	return l.Addr().String(), func() { l.Close() }, nil
}

func serveConn(conn net.Conn, cfg *ssh.ServerConfig, mem *sftp.Handlers) {
	_, chans, reqs, err := ssh.NewServerConn(conn, cfg)

	if err != nil {
		return
	}

	go ssh.DiscardRequests(reqs)

	for nc := range chans {
		switch nc.ChannelType() {
		case "session":
			ch, reqs, _ := nc.Accept()
			go serveSession(ch, reqs, mem)
		case "direct-tcpip":
			var p struct {
				Host     string
				Port     uint32
				OrigHost string
				OrigPort uint32
			}
			ssh.Unmarshal(nc.ExtraData(), &p)
			out, err := net.Dial("tcp", net.JoinHostPort(p.Host, strconv.Itoa(int(p.Port))))

			if err != nil {
				nc.Reject(ssh.ConnectionFailed, err.Error())
				continue
			}

			ch, reqs, _ := nc.Accept()
			go ssh.DiscardRequests(reqs)
			go func() {
				go io.Copy(out, ch)
				io.Copy(ch, out)
				ch.Close()
				out.Close()
			}()
		default:
			nc.Reject(ssh.UnknownChannelType, "no")
		}
	}
}

func serveSession(ch ssh.Channel, reqs <-chan *ssh.Request, mem *sftp.Handlers) {
	for req := range reqs {
		switch req.Type {
		case "pty-req", "env", "window-change":
			req.Reply(true, nil)
		case "shell":
			req.Reply(true, nil)
			go func() {
				ch.Write([]byte("welcome\r\n$ "))
				buf := make([]byte, 256)

				for {
					n, err := ch.Read(buf)

					if err != nil {
						return
					}

					ch.Write(buf[:n])
				}
			}()
		case "subsystem":
			var name struct{ Name string }
			ssh.Unmarshal(req.Payload, &name)

			if name.Name != "sftp" {
				req.Reply(false, nil)
				continue
			}

			req.Reply(true, nil)

			if mem != nil {
				rs := sftp.NewRequestServer(ch, *mem)
				go func() {
					rs.Serve()
					ch.Close()
				}()
				continue
			}

			srv, _ := sftp.NewServer(ch)
			go func() {
				srv.Serve()
				ch.Close()
			}()
		default:
			req.Reply(false, nil)
		}
	}
}

func demoFS() *sftp.Handlers {
	h := sftp.InMemHandler()

	for _, d := range []string{"/home", "/home/deploy", "/home/deploy/app", "/home/deploy/app/releases", "/home/deploy/logs", "/home/deploy/backups", "/home/deploy/.ssh", "/etc"} {
		h.FileCmd.Filecmd(sftp.NewRequest("Mkdir", d))
	}

	files := map[string]int{
		"/home/deploy/.bashrc":                  3771,
		"/home/deploy/.profile":                 807,
		"/home/deploy/docker-compose.yml":       1840,
		"/home/deploy/deploy.sh":                2210,
		"/home/deploy/README.md":                5120,
		"/home/deploy/backup-2026-10-01.tar.gz": 48_234_112,
		"/home/deploy/.env":                     412,
	}

	for name, size := range files {
		w, err := h.FilePut.Filewrite(sftp.NewRequest("Put", name))

		if err != nil {
			continue
		}

		w.WriteAt(make([]byte, size), 0)
	}

	return &h
}
