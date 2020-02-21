package utils

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
)

type SSHAgentWrapper struct {
	Socket string

	sshAgentBinary string
	pid            int
}

func CreateSSHAgentWrapper() (*SSHAgentWrapper, error) {
	path, err := exec.LookPath("ssh-agent")
	if err != nil {
		return nil, fmt.Errorf("Could not find ssh-agent in your system")
	}

	return &SSHAgentWrapper{"", path, 0}, nil
}

func (w *SSHAgentWrapper) AddKey(path string) error {
	_, _, _, err := ExecuteAndCollect([]string{}, w.sshAgentBinary, path)
	if err != nil {
		return fmt.Errorf("Could not add ssh key: %s", err.Error())
	}
	return nil
}

func (w *SSHAgentWrapper) Start(socketPath string) error {
	_, sout, serr, err := ExecuteAndCollect([]string{}, w.sshAgentBinary, "-a", socketPath)
	if err != nil {
		return fmt.Errorf("Could not start ssh-agent: %s: %s", err.Error(), serr)
	}

	// re := regexp.MustCompile(`SSH_AUTH_SOCK=(.+);`)
	// match := re.FindStringSubmatch(sout)
	// if match == nil {
	// 	return fmt.Errorf("Could not find ssh-agent socket")
	// }
	w.Socket = socketPath

	re := regexp.MustCompile(`SSH_AGENT_PID=(\d+);`)
	match := re.FindStringSubmatch(sout)
	if match == nil {
		return fmt.Errorf("Could not find ssh-agent PID")
	}

	pid, err := strconv.Atoi(match[1])
	if err != nil {
		return fmt.Errorf("Could not parse ssh-agent PID")
	}
	w.pid = pid

	PrintInfo("Started ssh-agent (pid=%d)", w.pid)
	return nil
}

func (w *SSHAgentWrapper) Stop() error {
	proc, err := os.FindProcess(w.pid)
	if err != nil {
		return fmt.Errorf("Could not find ssh-agent process: %s", err.Error())
	}

	PrintInfo("Stopping ssh-agent")
	return proc.Kill()
}
