package transfer

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/sftp"
)

func pipeClient(t *testing.T) *sftp.Client {
	cr, sw := io.Pipe()
	sr, cw := io.Pipe()

	server, err := sftp.NewServer(struct {
		io.Reader
		io.WriteCloser
	}{sr, sw})

	if err != nil {
		t.Fatal(err)
	}

	go server.Serve()

	client, err := sftp.NewClientPipe(cr, cw)

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		sw.Close()
		cw.Close()
		client.Close()
		server.Close()
	})

	return client
}

func wait(t *testing.T, j *Job) {
	deadline := time.Now().Add(10 * time.Second)

	for !j.Finished() {
		if time.Now().After(deadline) {
			t.Fatal("the transfer never finished")
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func TestFolderRoundTrip(t *testing.T) {
	c := pipeClient(t)
	src := t.TempDir()
	os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	os.WriteFile(filepath.Join(src, "a.txt"), []byte("alpha"), 0o644)
	os.WriteFile(filepath.Join(src, "sub", "b.txt"), bytes.Repeat([]byte("b"), 100_000), 0o644)

	remote := filepath.ToSlash(filepath.Join(t.TempDir(), "up"))
	q := New(nil)
	up := q.Add(c, &Job{Kind: Upload, Name: "up", Local: src, Remote: remote}, nil)
	wait(t, up)

	if up.State() != Done {
		t.Fatalf("upload: %v", up.Err())
	}

	if done, total := up.Progress(); done != total || total != 100_005 || up.Files() != 2 {
		t.Errorf("upload progress %d/%d, %d files", done, total, up.Files())
	}

	local := filepath.Join(t.TempDir(), "down")
	down := q.Add(c, &Job{Kind: Download, Name: "down", Local: local, Remote: remote}, nil)
	wait(t, down)

	if down.State() != Done {
		t.Fatalf("download: %v", down.Err())
	}

	got, _ := os.ReadFile(filepath.Join(local, "sub", "b.txt"))

	if len(got) != 100_000 {
		t.Errorf("the nested file came back with %d bytes", len(got))
	}

	q.ClearFinished()

	if len(q.Jobs()) != 0 {
		t.Error("ClearFinished left jobs behind")
	}
}

func TestCancel(t *testing.T) {
	c := pipeClient(t)
	src := filepath.Join(t.TempDir(), "big.bin")
	os.WriteFile(src, bytes.Repeat([]byte("x"), 20<<20), 0o644)

	q := New(nil)
	j := q.Add(c, &Job{Kind: Upload, Name: "big.bin", Local: src, Remote: filepath.ToSlash(filepath.Join(t.TempDir(), "big.bin"))}, nil)
	q.Cancel(j.ID)
	wait(t, j)

	if j.State() != Cancelled {
		t.Errorf("state after cancel: %v (%v)", j.State(), j.Err())
	}
}
