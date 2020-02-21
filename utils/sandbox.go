package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	// "reflect"
	"strings"

	"github.com/gobwas/glob"
	"github.com/hashicorp/hcl"
	"github.com/imdario/mergo"
)

var RequiredTerraformVersionPrefix string = "0.11."

type ProjectSandbox struct {
	baseDir string

	// Structure is:
	// { resourceType: { resourceName: { .. merged fields .. } } }
	tfProject map[string]map[string]map[string]interface{}
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

	sandbox := &ProjectSandbox{fPath, make(map[string]map[string]map[string]interface{})}
	err = sandbox.ReloadTerraformProject()
	if err != nil {
		return nil, err
	}

	return sandbox, nil
}

func (s *ProjectSandbox) ReloadTerraformProject() error {
	tf, err := s.ReadTerraformProject()
	if err != nil {
		return fmt.Errorf("could not parse project files: %s", err.Error())
	}

	s.tfProject = tf
	return nil
}

/**
 * @brief      Return the full path to the given file
 */
func (s *ProjectSandbox) GetFilePath(name string) string {
	return filepath.Join(s.baseDir, name)
}

/**
 * @brief      REturns a file on the temporary directory
 */
func (s *ProjectSandbox) GetTemporaryPath(name string) (string, error) {
	terraformDir := filepath.Join(s.baseDir, ".terraform")
	fullPath := filepath.Join(terraformDir, name)

	if err := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); err != nil {
		return "", fmt.Errorf("Unable to create terraform directory")
	}

	return fullPath, nil
}

/**
 * @brief      Checks if a file exists
 */
func (s *ProjectSandbox) HasFile(name string) bool {
	_, err := os.Stat(filepath.Join(s.baseDir, name))
	return err == nil
}

/**
 * @brief      Determines whether the specified path is file in sandbox.
 */
func (s *ProjectSandbox) IsFileInSandbox(path string) bool {
	fPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	return strings.HasPrefix(fPath, s.baseDir)
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
 * @brief      Checks if terraform exists in the system or in the sandbox and
 *  					 we will not have to download it.
 *
 * @return     True if system terraform, False otherwise.
 */
func (s *ProjectSandbox) HasTerraform() bool {
	terraformDir := filepath.Join(s.baseDir, ".terraform")

	path, err := exec.LookPath("terraform")
	if err == nil {
		w := CreateTeraformWrapper(path)
		if ver, err := w.GetVersion(); err == nil {
			if strings.HasPrefix(ver, RequiredTerraformVersionPrefix) {
				return true
			}
		}
	}

	fBinPath := filepath.Join(terraformDir, "bin")
	fPath := filepath.Join(fBinPath, "terraform")
	_, err = os.Stat(fPath)
	return err == nil
}

/**
 * @brief      Returns the full path to the terraform binary from within the
 * 						 sandbox directory.
 */
func (s *ProjectSandbox) GetTerraform() (*TerraformWrapper, error) {
	terraformDir := filepath.Join(s.baseDir, ".terraform")

	// First lookup terraform in the environment
	path, err := exec.LookPath("terraform")
	if err == nil {
		w := CreateTeraformWrapper(path)
		if ver, err := w.GetVersion(); err == nil {
			if strings.HasPrefix(ver, RequiredTerraformVersionPrefix) {
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
	w := CreateTeraformWrapper(fPath)
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

/**
 * @brief      Returns the full path to the terraform binary from within the
 * 						 sandbox directory.
 */
func (s *ProjectSandbox) WriteFile(file string, contents []byte) error {
	err := ioutil.WriteFile(filepath.Join(s.baseDir, file), contents, 0644)
	if err != nil {
		return fmt.Errorf("Could not write %s: %s", file, err.Error())
	}

	return nil
}

func (s *ProjectSandbox) ReadFile(file string) ([]byte, error) {
	return ioutil.ReadFile(filepath.Join(s.baseDir, file))
}

func (s *ProjectSandbox) ReadTerraformFile(file string) (map[string]interface{}, error) {
	content, err := s.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Could not read %s: %s", file, err.Error())
	}

	dst := make(map[string]interface{})
	err = hcl.Unmarshal(content, &dst)
	if err != nil {
		return nil, fmt.Errorf("Could not parse %s: %s", file, err.Error())
	}

	return dst, nil
}

func (s *ProjectSandbox) ReadTerraformProject() (map[string]map[string]map[string]interface{}, error) {
	files, err := ioutil.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("Could not enumerate files: %s", err.Error())
	}

	dst := make(map[string]map[string]map[string]interface{})
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".tf") {
			continue
		}

		slice, err := s.ReadTerraformFile(file.Name())
		if err != nil {
			return nil, err
		}

		for resType, _resTypeArr := range slice {
			dstResType := make(map[string]map[string]interface{})
			if pv, ok := dst[resType]; ok {
				dstResType = pv
			}

			if resNamesArr, ok := _resTypeArr.([]map[string]interface{}); ok {
				for _, resNamesMap := range resNamesArr {
					for resName, _resNameArr := range resNamesMap {
						dstResName := make(map[string]interface{})
						if pv, ok := dstResType[resName]; ok {
							dstResName = pv
						}

						if resValueMapArray, ok := _resNameArr.([]map[string]interface{}); ok {
							for _, resValueMap := range resValueMapArray {
								err := mergo.Merge(&dstResName, resValueMap)
								if err != nil {
									return nil, fmt.Errorf("Could not merge resource '%s.%s': %s", resType, resName, err.Error())
								}
							}
						} else {
							return nil, fmt.Errorf("Unexpected resource '%s.%s' type: %v", resType, resName, resValueMapArray)
						}

						dstResType[resName] = dstResName
					}
				}
			} else {
				return nil, fmt.Errorf("Unexpected resource type '%s' type: %v", resType, _resTypeArr)
			}

			dst[resType] = dstResType
		}
	}

	return dst, nil
}

func (s *ProjectSandbox) EnsureDcosProviderTF() error {

	if !s.HasFile("provider-dcos.tf") {
		lines := []string{
			`provider "dcos" {}`,
			`data "dcos_token" "current" {}`,
			`data "dcos_base_url" "current" {}`,
		}

		return s.WriteFile("provider-dcos.tf", []byte(strings.Join(lines, "\n")))
	}

	return nil
}

func (s *ProjectSandbox) GetTerraformResourcesMatching(resType string, resField string, fieldGlob string) []map[string]interface{} {
	var ret []map[string]interface{} = nil
	g := glob.MustCompile(fieldGlob)

	if mods, ok := s.tfProject[resType]; ok {
		for _, mod := range mods {
			if modSource, ok := mod[resField]; ok {
				if modSourceStr, ok := modSource.(string); ok {
					if g.Match(modSourceStr) {
						ret = append(ret, mod)
					}
				}
			}
		}
	}

	return ret
}
