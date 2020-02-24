package utils

import (
  "fmt"
  "io"
  "io/ioutil"
  "os"
  "os/exec"
  "os/signal"
  "strings"
  "syscall"
)

func updateEnv(a []string, b []string) []string {
  merged := make(map[string]string)

  for _, e := range a {
    pair := strings.SplitN(e, "=", 2)
    merged[pair[0]] = pair[1]
  }
  for _, e := range b {
    pair := strings.SplitN(e, "=", 2)
    merged[pair[0]] = pair[1]
  }

  var ret []string = nil
  for k, v := range merged {
    ret = append(ret, fmt.Sprintf("%s=%s", k, v))
  }

  return ret
}

/**
 * Run the given command and pipe stdout/stderr
 */
func ExecuteAndPassthrough(env []string, binary string, args ...string) (int, error) {
  cmd := exec.Command(binary, args...)
  cmd.Stdin = os.Stdin
  cmd.Env = updateEnv(os.Environ(), env)

  stdout, err := cmd.StdoutPipe()
  if err != nil {
    return 0, fmt.Errorf("Unable to open StdOut Pipe: %s", err.Error())
  }
  stderr, err := cmd.StderrPipe()
  if err != nil {
    return 0, fmt.Errorf("Unable to open StdErr Pipe: %s", err.Error())
  }
  if err := cmd.Start(); err != nil {
    return 0, err
  }

  // Async readers of the Stdout/Err
  go func() {
    _, _ = io.Copy(os.Stdout, stdout)
  }()
  go func() {
    _, _ = io.Copy(os.Stderr, stderr)
  }()

  // Forward interrupt signals to the launched process
  sigs := make(chan os.Signal, 1)
  signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
  go func() {
    for {
      sig := <-sigs
      if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
        return
      }
      cmd.Process.Signal(sig)
    }
  }()

  // Wait until the command is completed and remove the signal handlers
  err = cmd.Wait()
  signal.Reset(syscall.SIGINT, syscall.SIGTERM)
  sigs <- syscall.SIGINT

  if err != nil {
    // Get exit code on non-zero exits
    if exiterr, ok := err.(*exec.ExitError); ok {
      if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
        return status.ExitStatus(), nil
      }
    } else {
      return 0, err
    }
  }

  return 0, nil
}

/**
 * Change directory and run the given command and pipe stdout/stderr
 */
func ExecuteInFolderAndPassthrough(workDir string, binary string, args ...string) (int, error) {
  cmd := exec.Command(binary, args...)
  cmd.Env = os.Environ()
  cmd.Dir = workDir

  stdout, err := cmd.StdoutPipe()
  if err != nil {
    return 0, fmt.Errorf("Unable to open StdOut Pipe: %s", err.Error())
  }
  stderr, err := cmd.StderrPipe()
  if err != nil {
    return 0, fmt.Errorf("Unable to open StdErr Pipe: %s", err.Error())
  }
  if err := cmd.Start(); err != nil {
    return 0, err
  }

  // Async readers of the Stdout/Err
  go func() {
    _, _ = io.Copy(os.Stdout, stdout)
  }()
  go func() {
    _, _ = io.Copy(os.Stderr, stderr)
  }()

  if err := cmd.Wait(); err != nil {
    // Get exit code on non-zero exits
    if exiterr, ok := err.(*exec.ExitError); ok {
      if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
        return status.ExitStatus(), nil
      }
    } else {
      return 0, err
    }
  }

  return 0, nil
}

/**
 * Change directory and run the given command and pipe stdout/stderr
 */
func ExecuteAndCollect(env []string, binary string, args ...string) (int, string, string, error) {
  cmd := exec.Command(binary, args...)
  cmd.Env = updateEnv(os.Environ(), env)

  stdout, err := cmd.StdoutPipe()
  if err != nil {
    return 0, "", "", fmt.Errorf("Unable to open StdOut Pipe: %s", err.Error())
  }
  stderr, err := cmd.StderrPipe()
  if err != nil {
    return 0, "", "", fmt.Errorf("Unable to open StdErr Pipe: %s", err.Error())
  }
  if err := cmd.Start(); err != nil {
    return 0, "", "", err
  }

  // Read buffers
  ssout, err := ioutil.ReadAll(stdout)
  if err != nil {
    return 0, "", "", fmt.Errorf("Unable to read stdout: %s", err.Error())
  }
  stdout.Close()
  sserr, err := ioutil.ReadAll(stderr)
  if err != nil {
    return 0, "", "", fmt.Errorf("Unable to read stderr: %s", err.Error())
  }
  stderr.Close()

  if err := cmd.Wait(); err != nil {
    // Get exit code on non-zero exits
    if exiterr, ok := err.(*exec.ExitError); ok {
      if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
        return status.ExitStatus(), string(ssout), string(sserr), nil
      }
    } else {
      return 0, string(ssout), string(sserr), err
    }
  }

  return 0, string(ssout), string(sserr), nil
}

/**
 * Change directory and run the given command and pipe stdout/stderr
 */
func ShellExecuteInFolderAndPassthrough(workDir string, cmdline string) (int, error) {
  cmd := exec.Command("sh", "-c", cmdline)
  cmd.Env = os.Environ()
  cmd.Dir = workDir

  stdout, err := cmd.StdoutPipe()
  if err != nil {
    return 0, fmt.Errorf("Unable to open StdOut Pipe: %s", err.Error())
  }
  stderr, err := cmd.StderrPipe()
  if err != nil {
    return 0, fmt.Errorf("Unable to open StdErr Pipe: %s", err.Error())
  }
  if err := cmd.Start(); err != nil {
    return 0, err
  }

  // Async readers of the Stdout/Err
  go func() {
    _, _ = io.Copy(os.Stdout, stdout)
  }()
  go func() {
    _, _ = io.Copy(os.Stderr, stderr)
  }()

  if err := cmd.Wait(); err != nil {
    // Get exit code on non-zero exits
    if exiterr, ok := err.(*exec.ExitError); ok {
      if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
        return status.ExitStatus(), nil
      }
    } else {
      return 0, err
    }
  }

  return 0, nil
}

/**
 * Execute silently and return exit code
 */
func ExecuteSilently(binary string, args ...string) (int, error) {
  cmd := exec.Command(binary, args...)
  cmd.Env = os.Environ()
  if err := cmd.Start(); err != nil {
    return 0, err
  }

  if err := cmd.Wait(); err != nil {
    // Get exit code on non-zero exits
    if exiterr, ok := err.(*exec.ExitError); ok {
      if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
        return status.ExitStatus(), nil
      }
    } else {
      return 0, err
    }
  }

  return 0, nil
}

/**
 * Execute silently on a shell terminal return exit code
 */
func ShellExecuteSilently(cmdline string) (int, error) {
  cmd := exec.Command("sh", "-c", cmdline)
  cmd.Env = os.Environ()
  if err := cmd.Start(); err != nil {
    return 0, err
  }

  if err := cmd.Wait(); err != nil {
    // Get exit code on non-zero exits
    if exiterr, ok := err.(*exec.ExitError); ok {
      if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
        return status.ExitStatus(), nil
      }
    } else {
      return 0, err
    }
  }

  return 0, nil
}
