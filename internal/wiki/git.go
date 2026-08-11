package wiki

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type gitRunner struct {
	out io.Writer
	err io.Writer
}

func (g gitRunner) output(dir string, network bool, host string, args ...string) (string, error) {
	cmdArgs := args
	if dir != "" {
		cmdArgs = append([]string{"-C", dir}, args...)
	}
	cmd := exec.Command("git", cmdArgs...)
	if network {
		configureGitAuth(cmd, host)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		if detail == "" {
			detail = err.Error()
		}
		return stdout.String(), fmt.Errorf("git %s: %s", args[0], detail)
	}
	return stdout.String(), nil
}

func (g gitRunner) stream(dir string, network bool, host string, args ...string) error {
	cmdArgs := args
	if dir != "" {
		cmdArgs = append([]string{"-C", dir}, args...)
	}
	cmd := exec.Command("git", cmdArgs...)
	if network {
		configureGitAuth(cmd, host)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = g.out
	cmd.Stderr = g.err
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s failed: %w", args[0], err)
	}
	return nil
}

func configureGitAuth(cmd *exec.Cmd, host string) {
	executable, err := os.Executable()
	if err != nil {
		return
	}
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS="+executable,
		"GH_WIKI_ASKPASS_HOST="+host,
	)
}
