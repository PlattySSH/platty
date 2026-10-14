package main

import (
	"os"
	"path/filepath"
	"strings"

	platty "github.com/plattyssh/platty/internal/app"
)

const (
	esc    = "\x1b["
	reset  = esc + "0m"
	bold   = esc + "1m"
	dim    = esc + "2m"
	green  = esc + "32m"
	yellow = esc + "33m"
	blue   = esc + "34m"
	cyan   = esc + "36m"
	red    = esc + "31m"
)

func prompt(cmd string) string {
	return green + bold + "deploy@web-01" + reset + ":" + blue + bold + "~/api" + reset + "$ " + cmd
}

var gitSession = strings.Join([]string{
	prompt("git log --oneline --graph -6"),
	yellow + "* 3f2a91c" + reset + " (" + cyan + bold + "HEAD -> main" + reset + ", " + red + "origin/main" + reset + ") feat: stream logs over websockets",
	yellow + "* 8c41d07" + reset + " fix: retry => backoff when the queue is full",
	yellow + "*   e19b2aa" + reset + " Merge pull request #142 from ana/metrics",
	yellow + "|\\  " + reset,
	yellow + "| * 52d0f3e" + reset + " feat: expose p95 latency at /metrics",
	yellow + "|/  " + reset,
	yellow + "* 0c7e4b1" + reset + " chore: bump Go to 1.26",
	"",
	prompt("ls --color"),
	blue + bold + "cmd" + reset + "  " + blue + bold + "internal" + reset + "  " + blue + bold + "migrations" + reset + "  go.mod  go.sum  Makefile  " + green + bold + "deploy.sh" + reset,
	"",
	prompt("grep -n \"!= nil\" internal/queue/worker.go | head -4"),
	green + "41" + reset + ":\tif err != nil {",
	green + "58" + reset + ":\tif job.Retries >= max && err != nil {",
	green + "77" + reset + ":\tfor ctx.Err() == nil && w.next() != nil {",
	green + "93" + reset + ":\t\treturn fmt.Errorf(\"worker -> %w\", err)",
	"",
	prompt(""),
}, "\r\n")

var logSession = strings.Join([]string{
	prompt("tail -f /var/log/nginx/access.log"),
	dim + "10.0.1.21" + reset + " GET  /api/v1/jobs           " + green + "200" + reset + "  12ms",
	dim + "10.0.1.21" + reset + " POST /api/v1/jobs           " + green + "201" + reset + "  48ms",
	dim + "10.0.1.34" + reset + " GET  /api/v1/jobs/1842/logs " + green + "101" + reset + "   3ms",
	dim + "10.0.1.21" + reset + " GET  /healthz               " + green + "200" + reset + "   1ms",
	dim + "10.0.1.57" + reset + " GET  /api/v1/users/me       " + yellow + "304" + reset + "   4ms",
	dim + "10.0.1.34" + reset + " DELETE /api/v1/jobs/1838    " + green + "204" + reset + "  21ms",
	dim + "10.0.1.21" + reset + " GET  /api/v1/jobs?page=2    " + green + "200" + reset + "  15ms",
	dim + "10.0.1.88" + reset + " POST /api/v1/login          " + red + "401" + reset + "   9ms",
}, "\r\n")

func runScript(p *platty.Preview, title, text string, split bool) {
	path := filepath.Join(os.TempDir(), "platty-preview-"+title+".txt")
	os.WriteFile(path, []byte(text), 0o644)

	cmd := []string{"powershell.exe", "-NoLogo", "-NoProfile", "-Command",
		"[Console]::Out.Write([IO.File]::ReadAllText('" + path + "')); Start-Sleep 30"}

	if split {
		p.Split(true, title, cmd...)
		return
	}

	p.RunLocal(title, cmd...)
}
