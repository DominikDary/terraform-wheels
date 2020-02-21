package plugins

import (
	"fmt"
	"os/user"
	"strings"

	. "github.com/mesosphere-incubator/terraform-launch/utils"
)

type PluginAddService struct {
}

func CreatePluginAddService() *PluginAddService {
	return &PluginAddService{}
}

func (p *PluginAddService) GetName() string {
	return "add-service"
}

func (p *PluginAddService) IsUsed(project *ProjectSandbox) (bool, error) {
	return false, nil
}

func (p *PluginAddService) BeforeRun(project *ProjectSandbox, tf *TerraformWrapper) error {
	return nil
}

func (p *PluginAddService) AfterRun(project *ProjectSandbox, tf *TerraformWrapper, tfErr error) error {
	return nil
}

func (p *PluginAddService) GetCommands() []PluginCommand {
	return []PluginCommand{
		&PluginAddServiceCmdAddService{},
	}
}

type PluginAddServiceCmdAddService struct {
}

func (p *PluginAddServiceCmdAddService) GetName() string {
	return "add-service"
}

func (p *PluginAddServiceCmdAddService) GetDescription() string {
	return "Adds a configuration file to deploy an SDK service on DC/OS"
}

func (p *PluginAddServiceCmdAddService) Handle(project *ProjectSandbox, tf *TerraformWrapper) error {
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

	contents := []byte(strings.Join(lines, "\n"))
	return project.WriteFile("cluster-aws.tf", contents)
}
