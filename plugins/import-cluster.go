package plugins

import (
  "flag"
  "fmt"
  "io/ioutil"
  "os"

  . "github.com/logrusorgru/aurora"
  . "github.com/mesosphere-incubator/terraform-launch/utils"
  // "gopkg.in/yaml.v2"
)

type DcosLaunchInputConfig struct {
  // Universal Parameters
  KeyHelper             bool              `yaml:"key_helper"`
  Provider              string            `yaml:"provider"`
  SshPort               int               `yaml:"ssh_port"`
  SshPrivateKey         string            `yaml:"ssh_private_key"`
  SshPrivateKeyFilename string            `yaml:"ssh_private_key_filename"`
  SshUser               string            `yaml:"ssh_user"`
  Tags                  map[string]string `yaml:"tags"`
  ZenHelper             bool              `yaml:"zen_helper"`

  // (Undocumented)
  Enterprise bool `yaml:"dcos-enterprise"`

  // Template-based deploy params
  TemplateParameters map[string]interface{} `yaml:"template_parameters"`
  TemplateUrl        string                 `yaml:"template_url"`

  // On-prem
  DcosConfig               map[string]interface{} `yaml:"dcos_config"`
  DeploymentName           string                 `yaml:"deployment_name"`
  FaultDomainHelper        string                 `yaml:"fault_domain_helper"`
  GenconfDir               string                 `yaml:"genconf_dir"`
  InstallPrereqs           bool                   `yaml:"install_prereqs"`
  InstallerPort            int                    `yaml:"installer_port"`
  InstallerUrl             string                 `yaml:"installer_url"`
  NumMasters               int                    `yaml:"num_masters"`
  NumPrivateAgents         int                    `yaml:"num_private_agents"`
  NumPublicAgents          int                    `yaml:"num_public_agents"`
  OnpremInstallParallelism int                    `yaml:"onprem_install_parallelism"`
  Platform                 string                 `yaml:"platform"`
  PrereqsScriptFilename    string                 `yaml:"prereqs_script_filename"`

  // AWS On-Prem
  AdminLocation          string              `yaml:"admin_location"`
  AwsBlockDeviceMappings []map[string]string `yaml:"aws_block_device_mappings"`
  AwsKeyName             string              `yaml:"aws_key_name"`
  BootstrapSshUser       string              `yaml:"bootstrap_ssh_user"`
  IamRolePermissions     []map[string]string `yaml:"iam_role_permissions"`
  InstanceDeviceName     string              `yaml:"instance_device_name"`
  InstanceType           string              `yaml:"instance_type"`
  OsName                 string              `yaml:"os_name"`
}

type PluginImportCluster struct {
}

func CreatePluginImportCluster() *PluginImportCluster {
  return &PluginImportCluster{}
}

func (p *PluginImportCluster) GetName() string {
  return "import-cluster"
}

func (p *PluginImportCluster) IsUsed(project *ProjectSandbox) (bool, error) {
  return false, nil
}

func (p *PluginImportCluster) BeforeRun(project *ProjectSandbox, tf *TerraformWrapper, initRun bool) error {
  return nil
}

func (p *PluginImportCluster) AfterRun(project *ProjectSandbox, tf *TerraformWrapper, tfErr error) error {
  return nil
}

func (p *PluginImportCluster) GetCommands() []PluginCommand {
  return []PluginCommand{
    &PluginImportClusterCmdImport{},
  }
}

type PluginImportClusterCmdImport struct {
}

func (p *PluginImportClusterCmdImport) GetName() string {
  return "import-cluster"
}

func (p *PluginImportClusterCmdImport) GetDescription() string {
  return "Imports cluster configuration YAML in dcos-lauch format"
}

func (p *PluginImportClusterCmdImport) importKeyHelper(cfg *DcosLaunchInputConfig, project *ProjectSandbox) ([]string, error) {
  if cfg.KeyHelper {
    PrintInfo("Generating SSH key-pair because of `key_helper: true`")

    sshKey := "cluster-key"

    fPrivateKey := project.GetFilePath(GetPrivateKeyNameFromPub(sshKey))
    fPublicKey := project.GetFilePath(sshKey)
    err := CreateRSAKeyPair(fPrivateKey, fPublicKey)
    if err != nil {
      return nil, fmt.Errorf("Could not create RSA keypair: %s", err.Error())
    }
  }

  return nil, nil
}

func (p *PluginImportClusterCmdImport) PrintHelp() {
  fmt.Printf("Usage: %s %s [-help] [options] filename.yaml\n", os.Args[0], Bold(p.GetName()))
  fmt.Println("")
  fmt.Println("This command will convert the given dcos-lauch YAML configuration file into")
  fmt.Println("a terraform deployment module.")
  fmt.Println("")
  fmt.Println("Options:")
}

func (p *PluginImportClusterCmdImport) Handle(args []string, project *ProjectSandbox, tf *TerraformWrapper) error {
  var fileName string = "cluster-imported.tf"

  fSet := flag.NewFlagSet(p.GetName(), flag.ContinueOnError)

  help := fSet.Bool("help", false, "Show this help message")
  fSet.BoolVar(help, "h", false, "Show this help message")
  err := fSet.Parse(args)
  if err != nil {
    return err
  }

  if *help {
    p.PrintHelp()
    fSet.PrintDefaults()
    return nil
  }

  if len(fSet.Args()) < 1 {
    p.PrintHelp()
    fSet.PrintDefaults()
    return fmt.Errorf("Please specify the path to the configuration YAML to load")
  }

  _, err = ioutil.ReadFile(fSet.Args()[1])
  if err != nil {
    return fmt.Errorf("Could not load %s: %s", fSet.Args()[1], err.Error())
  }

  contents := []byte{}

  PrintInfo("%s", Bold("Writing "+fileName+" containing information for deploying a DC/OS cluster on AWS"))

  return project.WriteFile(fileName, contents)
}
