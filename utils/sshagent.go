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
  Pid    int

  sshAgentBinary string
  sshAddBinary   string
}

func CreateSSHAgentWrapper() (*SSHAgentWrapper, error) {
  pathAgent, err := exec.LookPath(ExecutableName("ssh-agent"))
  if err != nil {
    return nil, fmt.Errorf("Could not find ssh-agent in your system")
  }

  pathAdd, err := exec.LookPath(ExecutableName("ssh-add"))
  if err != nil {
    return nil, fmt.Errorf("Could not find ssh-add in your system")
  }

  return &SSHAgentWrapper{"", 0, pathAgent, pathAdd}, nil
}

func (w *SSHAgentWrapper) AddKey(path string) error {
  _, _, _, err := ExecuteAndCollect([]string{
    fmt.Sprintf("SSH_AUTH_SOCK=%s", w.Socket),
  }, w.sshAddBinary, path)
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
  //  return fmt.Errorf("Could not find ssh-agent socket")
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
  w.Pid = pid

  PrintInfo("Started ssh-agent (pid=%d)", w.Pid)
  return nil
}

func (w *SSHAgentWrapper) Stop() error {
  proc, err := os.FindProcess(w.Pid)
  if err != nil {
    return fmt.Errorf("Could not find ssh-agent process: %s", err.Error())
  }

  PrintInfo("Stopping ssh-agent")
  return proc.Kill()
}
