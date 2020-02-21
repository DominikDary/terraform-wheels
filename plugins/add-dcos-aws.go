package plugins

import (
  "flag"
  "fmt"
  "os"
  "os/user"

  . "github.com/logrusorgru/aurora"
  . "github.com/mesosphere-incubator/terraform-launch/utils"
)

type PluginDcosAws struct {
  showInstructions bool
  createdFile      string
}

func CreatePluginDcosAws() *PluginDcosAws {
  return &PluginDcosAws{false, ""}
}

func (p *PluginDcosAws) GetName() string {
  return "dcos-aws"
}

func (p *PluginDcosAws) IsUsed(project *ProjectSandbox) (bool, error) {
  mods := project.GetTerraformResourcesMatching("module", "source", "*dcos-terraform/dcos/aws")
  return mods != nil, nil
}

func (p *PluginDcosAws) BeforeRun(project *ProjectSandbox, tf *TerraformWrapper, initRun bool) error {
  if !IsAWSCredsOK() {
    err := fmt.Errorf("Could not find (still valid) AWS credentials in your enviroment. Use `maws login` and make sure to export the AWS_PROFILE")
    if initRun {
      PrintWarning(err.Error())
    } else {
      FatalError(err)
    }
  }

  return nil
}

func (p *PluginDcosAws) AfterRun(project *ProjectSandbox, tf *TerraformWrapper, tfErr error) error {
  if p.showInstructions {
    fmt.Println("")
    fmt.Println(Bold("You can now delpoy a cluster on Amazon AWS"))
    fmt.Println("")
    fmt.Printf("The file %s was generated in your project directory and it describes\n", p.createdFile)
    fmt.Println("the resources that are needed to be crated in order to deploy a DC/OS")
    fmt.Println("cluster on Amazon. Your next steps are:")
    fmt.Println("")
    fmt.Printf("  1. %s plan -out=plan.out  # To prepare your deployment\n", os.Args[0])
    fmt.Printf("  2. %s apply plan.out      # To create the deployment\n", os.Args[0])
    fmt.Println("")
  }
  return nil
}

func (p *PluginDcosAws) GetCommands() []PluginCommand {
  return []PluginCommand{
    &PluginDcosAwsCmdAddCluster{p},
  }
}

type PluginDcosAwsCmdAddCluster struct {
  parent *PluginDcosAws
}

func (p *PluginDcosAwsCmdAddCluster) GetName() string {
  return "add-aws-cluster"
}

func (p *PluginDcosAwsCmdAddCluster) GetDescription() string {
  return "Adds a configuration file to deploy a DC/OS cluster on AWS"
}

func (p *PluginDcosAwsCmdAddCluster) Handle(args []string, project *ProjectSandbox, tf *TerraformWrapper) error {
  var tfc TerraformFileConfig
  var fileName string = "cluster-aws.tf"

  currUserStr := "somebody"
  if u, err := user.Current(); err == nil {
    currUserStr = u.Username
  }

  tfc.Flags = flag.NewFlagSet(p.GetName(), flag.ContinueOnError)
  tfc.Flags.String("masters_instance_type", "", "[MASTERS] Instance type")
  tfc.Flags.String("public_agents_os", "", "[PUBLIC AGENTS] Operating system to use. Instead of using your own AMI you could use a provided OS.")
  tfc.Flags.String("public_agents_hostname_format", "", "[PUBLIC AGENTS] Format the hostname inputs are index+1, region, cluster_name")
  tfc.Flags.String("public_agents_additional_ports", "", "List of additional ports allowed for public access on public agents (80 and 443 open by default)")
  tfc.Flags.String("ssh_public_key_file", "", "Path to SSH public key. This is mandatory but can be set to an empty string if you want to use ssh_public_key with the key as string.")
  tfc.Flags.String("bootstrap_os", "", "[BOOTSTRAP] Operating system to use. Instead of using your own AMI you could use a provided OS.")
  tfc.Flags.String("bootstrap_root_volume_type", "", "[BOOTSTRAP] Root volume type")
  tfc.Flags.String("bootstrap_associate_public_ip_address", "", "[BOOTSTRAP] Associate a public ip address with there instances")
  tfc.Flags.String("private_agents_os", "", "[PRIVATE AGENTS] Operating system to use. Instead of using your own AMI you could use a provided OS.")
  tfc.Flags.String("tags", "", "Add custom tags to all resources (use key=value format)")
  tfc.Flags.String("masters_root_volume_size", "", "[MASTERS] Root volume size in GB")
  tfc.Flags.String("private_agents_associate_public_ip_address", "", "[PRIVATE AGENTS] Associate a public ip address with there instances")
  tfc.Flags.String("public_agents_user_data", "", "[PUBLIC AGENTS] User data to be used on these instances (cloud-init)")
  tfc.Flags.String("additional_public_agent_ips", "", "Additional public agent IPs.")
  tfc.Flags.String("num_public_agents", "", "Specify the amount of public agents. These agents will host marathon-lb and edgelb")
  tfc.Flags.String("public_agents_root_volume_size", "", "[PUBLIC AGENTS] Root volume size")
  tfc.Flags.String("masters_internal_acm_cert_arn", "", "ACM certifacte to be used for the internal masters load balancer")
  tfc.Flags.String("public_agents_acm_cert_arn", "", "ACM certifacte to be used for the public agents load balancer")
  tfc.Flags.String("private_agents_user_data", "", "[PRIVATE AGENTS] User data to be used on these instances (cloud-init)")
  tfc.Flags.String("public_agents_aws_ami", "", "[PUBLIC AGENTS] AMI to be used")
  tfc.Flags.String("additional_windows_private_agent_ips", "", "Additional windows private agent IPs")
  tfc.Flags.String("additional_windows_private_agent_passwords", "", "Additional windows private agent passwords to be used for WinRM")
  tfc.Flags.String("ansible_user", "", "The Ansible user that is used to run the Ansible Tasks.")
  tfc.Flags.String("subnet_range", "", "Private IP space to be used in CIDR format")
  tfc.Flags.String("ansible_bundled_container", "", "Docker container with bundled dcos-ansible and ansible executables")
  tfc.Flags.String("aws_key_name", "", "Specify the aws ssh key to use. We assume its already loaded in your SSH agent. Set ssh_public_key_file to empty string")
  tfc.Flags.String("num_private_agents", "", "Specify the amount of private agents. These agents will provide your main resources")
  tfc.Flags.String("bootstrap_hostname_format", "", "[BOOTSTRAP] Format the hostname inputs are index+1, region, cluster_name")
  tfc.Flags.String("masters_associate_public_ip_address", "", "[MASTERS] Associate a public ip address with there instances")
  tfc.Flags.String("masters_hostname_format", "", "[MASTERS] Format the hostname inputs are index+1, region, cluster_name")
  tfc.Flags.String("cluster_name_random_string", "", "Add a random string to the cluster name")
  tfc.Flags.String("additional_private_agent_ips", "", "Additional private agent IPs.")
  tfc.Flags.String("bootstrap_instance_type", "", "[BOOTSTRAP] Instance type")
  tfc.Flags.String("masters_os", "", "[MASTERS] Operating system to use. Instead of using your own AMI you could use a provided OS.")
  tfc.Flags.String("masters_iam_instance_profile", "", "[MASTERS] Instance profile to be used for these instances")
  tfc.Flags.String("masters_user_data", "", "[MASTERS] User data to be used on these instances (cloud-init)")
  tfc.Flags.String("private_agents_iam_instance_profile", "", "[PRIVATE AGENTS] Instance profile to be used for these instances")
  tfc.Flags.String("cluster_name", "", "Name of the DC/OS cluster")
  tfc.Flags.String("aws_ami", "", "AMI that will be used for the instances instead of the Mesosphere chosen default images. Custom AMIs must fulfill the Mesosphere DC/OS system-requirements: See https://docs.mesosphere.com/1.12/installing/production/system-requirements/")
  tfc.Flags.String("public_agents_extra_volumes", "", "[PUBLIC AGENTS] Extra volumes for each public agent")
  tfc.Flags.String("additional_windows_private_agent_os_user", "", "Additional windows private agent os user to be used for WinRM")
  tfc.Flags.String("bootstrap_root_volume_size", "", "[BOOTSTRAP] Root volume size in GB")
  tfc.Flags.String("bootstrap_iam_instance_profile", "", "[BOOTSTRAP] Instance profile to be used for these instances")
  tfc.Flags.String("private_agents_instance_type", "", "[PRIVATE AGENTS] Instance type")
  tfc.Flags.String("public_agents_access_ips", "", "List of ips allowed access to public agents. admin_ips are joined to this list (use multiple times to add multiple values)")
  tfc.Flags.String("ansible_additional_config", "", "Add additional config options to ansible. This is getting merged with generated defaults. Do not specify `dcos:`")
  tfc.Flags.String("dcos_instance_os", "", "Operating system to use. Instead of using your own AMI you could use a provided OS.")
  tfc.Flags.String("public_agents_associate_public_ip_address", "", "[PUBLIC AGENTS] Associate a public ip address with there instances")
  tfc.Flags.String("with_replaceable_masters", "", "Create S3 bucket for exhibitor and configure DC/OS to use it.")
  tfc.Flags.String("private_agents_extra_volumes", "", "[PRIVATE AGENTS] Extra volumes for each private agent")
  tfc.Flags.String("public_agents_root_volume_type", "", "[PUBLIC AGENTS] Specify the root volume type.")
  tfc.Flags.String("admin_ips", "", "List of CIDR admin IPs (use multiple times to add multiple values)")
  tfc.Flags.String("availability_zones", "", "List of availability_zones to be used as the same format that are required by the platform/cloud providers. i.e `['RegionZone']` (use multiple times to add multiple values)")
  tfc.Flags.String("bootstrap_aws_ami", "", "[BOOTSTRAP] AMI to be used")
  tfc.Flags.String("private_agents_aws_ami", "", "[PRIVATE AGENTS] AMI to be used")
  tfc.Flags.String("private_agents_root_volume_type", "", "[PRIVATE AGENTS] Root volume type")
  tfc.Flags.String("accepted_internal_networks", "", "Subnet ranges for all internal networks (use multiple times to add multiple values)")
  tfc.Flags.String("masters_acm_cert_arn", "", "ACM certifacte to be used for the masters load balancer")
  tfc.Flags.String("ssh_public_key", "", "SSH public key in authorized keys format (e.g. 'ssh-rsa ..') to be used with the instances. Make sure you added this key to your ssh-agent.")
  tfc.Flags.String("num_masters", "", "Specify the amount of masters. For redundancy you should have at least 3")
  tfc.Flags.String("private_agents_root_volume_size", "", "[PRIVATE AGENTS] Root volume size in GB")
  tfc.Flags.String("private_agents_hostname_format", "", "[PRIVATE AGENTS] Format the hostname inputs are index+1, region, cluster_name")
  tfc.Flags.String("public_agents_instance_type", "", "[PUBLIC AGENTS] Instance type")
  tfc.Flags.String("masters_aws_ami", "", "[MASTERS] AMI to be used")
  tfc.Flags.String("public_agents_iam_instance_profile", "", "[PUBLIC AGENTS] Instance profile to be used for these instances")

  tfc.ListFlags = []string{"public_agents_access_ips", "admin_ips", "availability_zones", "accepted_internal_networks"}
  tfc.MapFlags = []string{"tags"}

  help := tfc.Flags.Bool("help", false, "Show this help message")
  tfc.Flags.BoolVar(help, "h", false, "Show this help message")
  err := tfc.Flags.Parse(args)
  if err != nil {
    FatalError(err)
  }

  if *help {
    fmt.Printf("Usage: %s %s [-help] [options]\n", os.Args[0], Bold(p.GetName()))
    fmt.Println("")
    fmt.Printf("This command will generate a '%s' file in the project directory\n", fileName)
    fmt.Println("that describes a deployment of a DC/OS cluster on AWS. A file with sane defaults")
    fmt.Println("is created for you. You can override the values with the following flags:")
    fmt.Println("")
    fmt.Println("Options:")
    tfc.PrintOptionHelp()
    return nil
  }

  tfc.PreLines = []string{
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
  }
  tfc.BodyLines = []string{
    `  cluster_name               = "my-dcos-demo"`,
    `  cluster_name_random_string = true`,
    `  ssh_public_key_file        = "cluster-key.pub"`,
    `  admin_ips                  = ["${data.http.whatismyip.body}/32"]`,
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
  }
  tfc.PostLines = []string{
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

  contents, err := tfc.Generate()
  if err != nil {
    return err
  }

  PrintInfo("%s", Bold("Writing "+fileName+" containing information for deploying a DC/OS cluster on AWS"))
  p.parent.showInstructions = true
  p.parent.createdFile = fileName

  return project.WriteFile(fileName, contents)
}
