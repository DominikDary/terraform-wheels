package utils

import (
	"fmt"
	"regexp"
)

type TerraformWrapper struct {
	terraformPath string
	env           []string
}

func CreateTeraformWrapper(fName string) *TerraformWrapper {
	return &TerraformWrapper{fName, nil}
}

func (w *TerraformWrapper) SetEnv(key string, value string) {
	w.env = append(w.env, fmt.Sprintf("%s=%s", key, value))
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
	_, err := ExecuteAndPassthrough(w.env, w.terraformPath, args...)
	return err
}
