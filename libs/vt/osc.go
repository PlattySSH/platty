package vt

import (
	"bytes"
	"encoding/base64"
	"net/url"
)

const maxLinks = 1<<16 - 1

func WithClipboard(fn func(text string)) Option {
	return func(t *Terminal) {
		t.onClipboard = fn
	}
}

func (t *Terminal) setLink(data []byte) {
	_, uri, ok := bytes.Cut(data, []byte{';'})

	if !ok || len(uri) == 0 {
		t.cur.pen.Link = 0
		return
	}

	s := string(uri)

	if t.linkIDs == nil {
		t.linkIDs = map[string]uint16{}
	}

	id, ok := t.linkIDs[s]

	if !ok {
		if len(t.links) >= maxLinks {
			t.links, t.linkIDs = nil, map[string]uint16{}
		}

		t.links = append(t.links, s)
		id = uint16(len(t.links))
		t.linkIDs[s] = id
	}

	t.cur.pen.Link = id
}

func (t *Terminal) Link(id uint16) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if id == 0 || int(id) > len(t.links) {
		return ""
	}

	return t.links[id-1]
}

func (t *Terminal) setCwd(data string) {
	u, err := url.Parse(data)

	if err != nil || u.Scheme != "file" {
		return
	}

	t.cwdHost, t.cwdPath = u.Host, u.Path
}

func (t *Terminal) Cwd() (host, path string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.cwdHost, t.cwdPath
}

func (t *Terminal) setClipboard(data []byte) {
	_, payload, ok := bytes.Cut(data, []byte{';'})

	if !ok || t.onClipboard == nil || string(payload) == "?" {
		return
	}

	text, err := base64.StdEncoding.DecodeString(string(payload))

	if err != nil {
		return
	}

	t.onClipboard(string(text))
}
