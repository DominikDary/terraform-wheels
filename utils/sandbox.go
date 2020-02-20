package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	// "runtime"
	"strings"
)

type ProjectSandbox struct {
	baseDir string
}

func OpenSandbox(baseDir string) (*ProjectSandbox, error) {
	fPath, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("could not compute absolute path: %s", err.Error())
	}

	_, err = os.Stat(fPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("could not get path stat: %s", err.Error())
		}

		err := os.Mkdir(fPath, os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("could not create sandbox dir: %s", err.Error())
		}
	}

	return &ProjectSandbox{fPath}, nil
}

/**
 * @brief      Return the full path to the given file
 */
func (s *ProjectSandbox) GetFilePath(name string) string {
	return filepath.Join(s.baseDir, name)
}

/**
 * @brief      Checks if a file exists
 */
func (s *ProjectSandbox) HasFile(name string) bool {
	_, err := os.Stat(filepath.Join(s.baseDir, name))
	return err == nil
}

/**
 * @brief      Returns true if the plugin exists
 */
func (s *ProjectSandbox) HasPlugin(name string) (bool, error) {
	// pluginsDir := filepath.Join(s.baseDir, ".terraform", "plugins", "%s_%s".format(runtime.GOOS, runtime.GOARCH))

	// // First lookup terraform in the environment
	// path, err := exec.LookPath(filepath.Join(pluginsDir, name))
	// if err == nil {
	// 	w := CreateTeraformWrapper(path)
	// 	if ver, err := w.GetVersion(); err == nil {
	// 		if strings.HasPrefix(ver, "0.11.") {
	// 			PrintInfo("Using system terraform v%s", ver)
	// 			return w, nil
	// 		}
	// 	}
	// }
	return false, nil
}

func (s *ProjectSandbox) IsEmpty() (bool, error) {
	foundFile := false
	err := filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if info.Name()[0] == '.' || strings.Contains(path, "/.") { // Ignore hidden
			return nil
		}

		foundFile = true
		return nil
	})
	if err != nil {
		return false, err
	}

	return !foundFile, nil
}

func (s *ProjectSandbox) HasTerraformFiles() (bool, error) {
	hasTf := false
	err := filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
		if info.Name()[0] == '.' || strings.Contains(path, "/.") { // Ignore hidden
			return nil
		}
		if strings.HasSuffix(path, ".tf") {
			hasTf = true
		}
		return nil
	})
	if err != nil {
		return false, err
	}

	return hasTf, nil
}

/**
 * @brief      Returns the full path to the terraform binary from within the
 * 						 sandbox directory.
 */
func (s *ProjectSandbox) GetTerraform(sshAgent *SSHAgentWrapper) (*TerraformWrapper, error) {
	terraformDir := filepath.Join(s.baseDir, ".terraform")

	// First lookup terraform in the environment
	path, err := exec.LookPath("terraform")
	if err == nil {
		w := CreateTeraformWrapper(path, sshAgent)
		if ver, err := w.GetVersion(); err == nil {
			if strings.HasPrefix(ver, "0.11.") {
				PrintInfo("Using system terraform v%s", ver)
				return w, nil
			}
		}
	}

	fBinPath := filepath.Join(terraformDir, "bin")
	fPath := filepath.Join(fBinPath, "terraform")
	if err = os.MkdirAll(fBinPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("Unable to create terraform directory")
	}

	_, err = os.Stat(fPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("Could not check if terraform binary exists: %s", err.Error())
		}

		// Findt he upstream URL to use
		url, checksum, err := upstreamGetTerraform()
		if err != nil {
			return nil, err
		}

		// Download terraform
		err = Download(url, WithDefaults).
			AndShowProgress("Downloading terraform").
			AndValidateChecksum(checksum).
			EventuallyUnzipTo(fBinPath, 0)
		if err != nil {
			FatalError(err)
		}
	}

	// Try to use the binary
	w := CreateTeraformWrapper(fPath, sshAgent)
	ver, err := w.GetVersion()
	if err != nil {
		return nil, fmt.Errorf("Unable to execute the cached terraform binary. Try deleting .terraform directory and re-run again.")
	}
	if ver != upstreamTerraformVersion {
		return nil, fmt.Errorf("Unexpected cached terraform version. Try deleting .terraform directory and re-run again")
	}

	PrintInfo("Using project-local terraform v%s", upstreamTerraformVersion)
	return w, nil
}

/**
 * @brief      Returns the full path to the terraform binary from within the
 * 						 sandbox directory.
 */
func (s *ProjectSandbox) InitProject() error {
	currUserStr := "somebody"
	if u, err := user.Current(); err == nil {
		currUserStr = u.Username
	}

	var lines []string = []string{
		`provider "aws" {`,
		`  # Change your default region here`,
		`  region = "us-west-2"`,
		`}`,
		``,
		`# Used to determine your public IP for forwarding rules`,
		`data "http" "whatismyip" {`,
		`  url = "http://whatismyip.akamai.com/"`,
		`}`,
		``,
		`module "dcos" {`,
		`  source  = "dcos-terraform/dcos/aws"`,
		fmt.Sprintf(`  version = "~> %s"`, GetLatestModuleVersion("0.2.0")),
		``,
		`  providers = {`,
		`    aws = "aws"`,
		`  }`,
		``,
		`  cluster_name        				= "my-dcos-demo"`,
		`  cluster_name_random_string = true`,
		`  ssh_public_key_file 				= "cluster-key.pub"`,
		`  admin_ips           				= ["${data.http.whatismyip.body}/32"]`,
		``,
		`  num_masters        = 1`,
		`  num_private_agents = 1`,
		`  num_public_agents  = 1`,
		``,
		fmt.Sprintf(`  dcos_version = "%s"`, GetLatestDCOSVersion("open", "2.0.0")),
		``,
		`  # dcos_variant              = "ee"`,
		`  # dcos_license_key_contents = "${file("./license.txt")}"`,
		`  # Make sure to set your credentials if you do not want the default EE`,
		`  # dcos_superuser_username      = "superuser-name"`,
		`  # dcos_superuser_password_hash = "${file("./dcos_superuser_password_hash.sha512")}"`,
		`  dcos_variant = "open"`,
		``,
		`  dcos_instance_os             = "centos_7.5"`,
		`  bootstrap_instance_type      = "t2.medium"`,
		`  masters_instance_type        = "t2.medium"`,
		`  private_agents_instance_type = "t2.medium"`,
		`  public_agents_instance_type  = "t2.medium"`,
		``,
		`  tags = {`,
		`    "expiration" = "1h"`,
		fmt.Sprintf(`    "owner"      = "%s"`, currUserStr),
		`  }`,
		`}`,
		``,
		`output "masters-ips" {`,
		`  value = "${module.dcos.masters-ips}"`,
		`}`,
		``,
		`output "cluster-address" {`,
		`  value = "${module.dcos.masters-loadbalancer}"`,
		`}`,
		``,
		`output "public-agents-loadbalancer" {`,
		`  value = "${module.dcos.public-agents-loadbalancer}"`,
		`}`,
	}

	fPrivateKey := filepath.Join(s.baseDir, "cluster-key")
	fPublicKey := filepath.Join(s.baseDir, "cluster-key.pub")
	err := CreateRSAKeyPair(fPrivateKey, fPublicKey)
	if err != nil {
		return fmt.Errorf("Could not generate RSA keypair: %s", err.Error())
	}

	contents := []byte(strings.Join(lines, "\n"))
	fMain := filepath.Join(s.baseDir, "main.tf")
	err = ioutil.WriteFile(fMain, contents, 0644)
	if err != nil {
		return fmt.Errorf("Could not create main project file: %s", err.Error())
	}

	return nil
}
