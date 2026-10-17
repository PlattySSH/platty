package shell

import (
	"os"
	"os/exec"
	"path/filepath"
)

var cmd = Profile{ID: "cmd", Name: "Command Prompt", Command: []string{"cmd.exe"}}

func Detect() []Profile {
	candidates := []Profile{
		{ID: "pwsh", Name: "PowerShell 7", Command: []string{"pwsh.exe", "-NoLogo"}},
		{ID: "powershell", Name: "Windows PowerShell", Command: []string{"powershell.exe", "-NoLogo"}},
		cmd,
		{ID: "wsl", Name: "WSL", Command: []string{"wsl.exe", "~"}},
	}

	var out []Profile

	for _, p := range candidates {
		if _, err := exec.LookPath(p.Command[0]); err == nil {
			out = append(out, p)
		}
	}

	for _, bash := range []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Git", "bin", "bash.exe"),
		filepath.Join(os.Getenv("LocalAppData"), "Programs", "Git", "bin", "bash.exe"),
	} {
		if _, err := os.Stat(bash); err == nil {
			out = append(out, Profile{ID: "gitbash", Name: "Git Bash", Command: []string{bash, "--login", "-i"}})
			break
		}
	}

	if len(out) == 0 {
		out = append(out, cmd)
	}

	return out
}
