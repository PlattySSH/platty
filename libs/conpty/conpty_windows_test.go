package conpty

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestStartEcho(t *testing.T) {
	p, err := Start(Options{Command: []string{"cmd.exe", "/c", "echo platty-ok"}, Cols: 80, Rows: 24})

	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	done := make(chan struct{})
	go func() {
		io.Copy(&out, p)
		close(done)
	}()

	code, err := p.Wait()

	if err != nil || code != 0 {
		t.Fatalf("wait: code=%d err=%v", code, err)
	}

	p.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the reader got no EOF after Close")
	}

	if !bytes.Contains(out.Bytes(), []byte("platty-ok")) {
		t.Fatalf("output: %q", out.String())
	}
}
