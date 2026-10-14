package sshconfig

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Entry struct {
	Alias    string
	HostName string
	User     string
	Port     int
	Identity string
	Jump     string
}

func Parse(path string) ([]Entry, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	var out []Entry
	var cur *Entry

	sc := bufio.NewScanner(f)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, ok := strings.Cut(line, " ")

		if !ok {
			key, val, ok = strings.Cut(line, "=")
		}

		if !ok {
			continue
		}

		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.Trim(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(val), "=")), `"`)

		if key == "host" {
			cur = nil
			alias := strings.Fields(val)[0]

			if strings.ContainsAny(alias, "*?!") {
				continue
			}

			out = append(out, Entry{Alias: alias, Port: 22})
			cur = &out[len(out)-1]
			continue
		}

		if cur == nil {
			continue
		}

		switch key {
		case "hostname":
			cur.HostName = val
		case "user":
			cur.User = val
		case "port":
			cur.Port, _ = strconv.Atoi(val)
		case "identityfile":
			cur.Identity = ExpandHome(val)
		case "proxyjump":
			cur.Jump = strings.Split(val, ",")[0]
		}
	}

	return out, sc.Err()
}

func ExpandHome(p string) string {
	if strings.HasPrefix(p, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[1:])
	}

	return p
}

func Format(entries []Entry) string {
	var b strings.Builder

	for i, e := range entries {
		if i > 0 {
			b.WriteString("\n")
		}

		b.WriteString("Host " + e.Alias + "\n")
		b.WriteString("    HostName " + e.HostName + "\n")

		if e.User != "" {
			b.WriteString("    User " + e.User + "\n")
		}

		if e.Port != 0 && e.Port != 22 {
			b.WriteString("    Port " + strconv.Itoa(e.Port) + "\n")
		}

		if e.Identity != "" {
			b.WriteString("    IdentityFile " + e.Identity + "\n")
		}

		if e.Jump != "" {
			b.WriteString("    ProxyJump " + e.Jump + "\n")
		}
	}

	return b.String()
}

func Alias(name string) string {
	f := strings.FieldsFunc(strings.ToLower(name), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-')
	})

	if len(f) == 0 {
		return "host"
	}

	return strings.Join(f, "-")
}
