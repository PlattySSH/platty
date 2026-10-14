package version

import (
	"runtime"
	"runtime/debug"
)

const (
	Repo    = "https://github.com/PlattySSH/platty"
	Website = "https://plattyssh.github.io"
)

var Version = "0.1.0-dev"

func Go() string {
	return runtime.Version()
}

func Commit() string {
	info, ok := debug.ReadBuildInfo()

	if !ok {
		return ""
	}

	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && len(s.Value) >= 7 {
			return s.Value[:7]
		}
	}

	return ""
}
