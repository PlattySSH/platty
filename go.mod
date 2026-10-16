module github.com/plattyssh/platty

go 1.26.5

require (
	gioui.org v0.10.3
	gioui.org/x v0.10.3
	github.com/go-text/typesetting v0.3.5
	github.com/pkg/sftp v1.13.11
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/crypto v0.57.0
	golang.org/x/exp/shiny v0.0.0-20260908205506-85c1c2202aba // indirect
	golang.org/x/image v0.46.0
	golang.org/x/sys v0.48.0
)

require (
	gioui.org/shader v1.0.9 // indirect
	git.wow.st/gmp/jni v0.0.0-20210610011705-34026c7e22d0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/kr/fs v0.1.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)

require (
	github.com/plattyssh/glyphs v0.0.0
	github.com/plattyssh/pty v0.0.0
	github.com/plattyssh/vt v0.0.0
)

// The libraries live in libs/ until they are published under github.com/PlattySSH.
replace (
	github.com/plattyssh/glyphs => ./libs/glyphs
	github.com/plattyssh/pty => ./libs/pty
	github.com/plattyssh/vt => ./libs/vt
)
