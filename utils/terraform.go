package utils

import (
	"fmt"
	"regexp"
)

type TerraformWrapper struct {
	terraformPath string
	sshAgent      *SSHAgentWrapper
}

func CreateTeraformWrapper(fName string, sshAgent *SSHAgentWrapper) *TerraformWrapper {
	return &TerraformWrapper{fName, sshAgent}
}

func (w *TerraformWrapper) GetVersion() (string, error) {
	_, sout, _, err := ExecuteAndCollect([]string{}, w.terraformPath, "--version")
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`Terraform v([0-9\.]+)`)
	match := re.FindStringSubmatch(sout)
	if match == nil {
		return "", fmt.Errorf("Could not find terraform version")
	}

	return match[1], nil
}

func (w *TerraformWrapper) Invoke(args []string) error {
	env := []string{
		fmt.Sprintf("SSH_AUTH_SOCK=%s", w.sshAgent.socket),
	}
	_, err := ExecuteAndPassthrough(env, w.terraformPath, args...)
	return err
}
