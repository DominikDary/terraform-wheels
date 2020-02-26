package plugins

import (
  // "encoding/json"
  "flag"
  "fmt"
  "io/ioutil"
  "os"
  "strings"

  . "github.com/logrusorgru/aurora"
  . "github.com/mesosphere-incubator/terraform-launch/utils"
  "gopkg.in/yaml.v3"
)

type DcosLaunchInputConfig struct {
  // Universal Parameters
  KeyHelper             bool                   `yaml:"key_helper"`
  Provider              string                 `yaml:"provider"`
  SshPort               int                    `yaml:"ssh_port"`
  SshPrivateKey         string                 `yaml:"ssh_private_key"`
  SshPrivateKeyFilename string                 `yaml:"ssh_private_key_filename"`
  SshUser               string                 `yaml:"ssh_user"`
  Tags                  map[string]interface{} `yaml:"tags"`
  ZenHelper             bool                   `yaml:"zen_helper"`

  // (Undocumented)
  Enterprise bool `yaml:"dcos-enterprise"`

  // Template-based deploy params
  TemplateParameters map[string]interface{} `yaml:"template_parameters"`
  TemplateUrl        string                 `yaml:"template_url"`

  // On-prem
  DcosConfig               map[string]interface{} `yaml:"dcos_config"`
  DcosVersion              string                 `yaml:"dcos_version"`
  DcosInstallerUrl         string                 `yaml:"installer_url"`
  DeploymentName           string                 `yaml:"deployment_name"`
  FaultDomainHelper        string                 `yaml:"fault_domain_helper"`
  GenconfDir               string                 `yaml:"genconf_dir"`
  InstallPrereqs           bool                   `yaml:"install_prereqs"`
  InstallerPort            int                    `yaml:"installer_port"`
  NumMasters               int                    `yaml:"num_masters"`
  NumPrivateAgents         int                    `yaml:"num_private_agents"`
  NumPublicAgents          int                    `yaml:"num_public_agents"`
  OnpremInstallParallelism int                    `yaml:"onprem_install_parallelism"`
  Platform                 string                 `yaml:"platform"`
  PrereqsScriptFilename    string                 `yaml:"prereqs_script_filename"`

  // AWS On-Prem
  AwsRegion              string                   `yaml:"aws_region"`
  AdminLocation          string                   `yaml:"admin_location"`
  AwsKeyName             string                   `yaml:"aws_key_name"`
  BootstrapSshUser       string                   `yaml:"bootstrap_ssh_user"`
  InstanceDeviceName     string                   `yaml:"instance_device_name"`
  InstanceType           string                   `yaml:"instance_type"`
  OsName                 string                   `yaml:"os_name"`
  AwsBlockDeviceMappings []map[string]interface{} `yaml:"aws_block_device_mappings"`
  IamRolePermissions     []map[string]interface{} `yaml:"iam_role_permissions"`
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

func (p *PluginImportClusterCmdImport) importSSHKeys(cfg *DcosLaunchInputConfig, project *ProjectSandbox) ([]string, error) {
  sshKey := "cluster-key.pub"
  if cfg.DeploymentName != "" {
    sshKey = fmt.Sprintf("cluster-%s-key.pub", cfg.DeploymentName)
  }
  fPublicKey := sshKey
  fPrivateKey := GetPrivateKeyNameFromPublic(sshKey)

  if cfg.KeyHelper {
    PrintInfo("Generating SSH key-pair because `key_helper` is used")

    err := CreateRSAKeyPair(fPrivateKey, fPublicKey)
    if err != nil {
      return nil, fmt.Errorf("Could not create RSA keypair: %s", err.Error())
    }

    return []string{
      `# Using generated private/public key pair because key_helper is used`,
      fmt.Sprintf(`ssh_public_key_file = "%s"`, fPublicKey),
      "",
    }, nil
  } else if cfg.SshPrivateKey != "" {
    PrintInfo("Dumping private/public key pair from private key contents")

    privateKeyBytes := []byte(cfg.SshPrivateKey)

    err := ioutil.WriteFile(fPrivateKey, privateKeyBytes, 0644)
    if err != nil {
      return nil, fmt.Errorf("Error writing private key %s: %s", fPrivateKey, err.Error())
    }

    err = CreatePublicRSAKeyFromPrivate(privateKeyBytes, fPublicKey)
    if err != nil {
      return nil, fmt.Errorf("Error writing public key %s: %s", fPublicKey, err.Error())
    }

    return []string{
      fmt.Sprintf(`ssh_public_key_file = "%s"`, fPublicKey),
    }, nil

  } else if cfg.SshPrivateKeyFilename != "" {
    PrintInfo("Using private SSH key %s", Bold(cfg.SshPrivateKeyFilename))

    fPublicKey = GetPublicKeyNameFromPrivate(cfg.SshPrivateKeyFilename)
    _, err := os.Stat(fPublicKey)
    if err != nil {
      return nil, fmt.Errorf("Did not find the respective public key for %s (looking at %s)", fPublicKey)
    }

    return []string{
      fmt.Sprintf(`ssh_public_key_file = "%s"`, fPublicKey),
    }, nil
  } else {
    return nil, fmt.Errorf("Please use one of: `key_helper`, `ssh_private_key` or `ssh_private_key_filename`")
  }

  return nil, nil
}

func (p *PluginImportClusterCmdImport) importDcosConfig(cfg map[string]interface{}, project *ProjectSandbox) ([]string, error) {
  var lines []string = nil
  var rawDcosConfig map[string]interface{} = make(map[string]interface{})

  if cfg == nil {
    return nil, nil
  }

  mapVars := []string{
    "adminrouter_tls_1_0_enabled", "adminrouter_tls_1_1_enabled",
    "adminrouter_tls_1_2_enabled", "adminrouter_tls_cipher_suite",
    "agent_list", "audit_logging", "auth_cookie_secure_flag", "aws_access_key_id",
    "aws_region", "aws_secret_access_key", "aws_template_storage_access_key_id",
    "aws_template_storage_bucket", "aws_template_storage_bucket_path",
    "aws_template_storage_region_name", "aws_template_storage_secret_access_key",
    "aws_template_upload", "bootstrap_port", "bouncer_expiration_auth_token_days",
    "ca_certificate_chain_path", "ca_certificate_key_path", "ca_certificate_path",
    "calico_ipinip_mtu", "calico_network_cidr", "calico_veth_mtu",
    "calico_vxlan_enabled", "calico_vxlan_mtu", "calico_vxlan_port",
    "calico_vxlan_vni", "check_time", "cluster_docker_credentials",
    "cluster_docker_credentials_dcos_owned",
    "cluster_docker_credentials_enabled",
    "cluster_docker_credentials_write_to_etc",
    "cluster_docker_registry_enabled",
    "cluster_docker_registry_url", "cluster_name", "config", "custom_checks",
    "customer_key", "dns_bind_ip_blacklist", "dns_forward_zones", "dns_search",
    "docker_remove_delay", "download_url_checksum", "enable_docker_gc",
    "enable_gpu_isolation", "enable_mesos_input_plugin", "exhibitor_address",
    "exhibitor_azure_account_key", "exhibitor_azure_account_name", "exhibitor_azure_prefix",
    "exhibitor_explicit_keys", "exhibitor_storage_backend", "exhibitor_zk_hosts",
    "exhibitor_zk_path", "fault_domain_detect_contents", "fault_domain_enabled",
    "gc_delay", "gpus_are_scarce", "http_proxy", "https_proxy", "image_commit",
    "instance_os", "ip_detect_contents", "ip_detect_public_contents", "ip_detect_public_filename",
    "l4lb_enable_ipv6", "license_key_contents", "log_directory", "master_discovery",
    "master_dns_bindall", "master_external_loadbalancer", "master_list",
    "mesos_container_log_sink", "mesos_dns_set_truncate_bit",
    "mesos_max_completed_tasks_per_framework", "no_proxy", "num_masters",
    "oauth_enabled", "overlay_config_attempts", "overlay_enable", "overlay_mtu",
    "overlay_network", "package_storage_uri", "previous_version",
    "previous_version_master_index", "process_timeout", "public_agent_list",
    "resolvers", "rexray_config", "rexray_config_filename", "rexray_config_method",
    "s3_bucket", "s3_prefix", "security", "skip_checks", "staged_package_storage_uri",
    "superuser_password_hash", "superuser_username", "telemetry_enabled",
    "ucr_default_bridge_subnet", "use_proxy", "variant", "version", "versions_service_url",
    "zk_agent_credentials", "zk_master_credentials", "zk_super_credentials",
  }

  for k, iv := range cfg {
    hasMapping := false
    for _, n := range mapVars {
      if n == k {
        hasMapping = true
        break
      }
    }

    if hasMapping {
      // If there is a direct mapping convert the X to dcos_x variable and
      // export it as proper terraform variable
      k = "dcos_" + k
      switch v := iv.(type) {
      case map[string]interface{}:
        lines = append(lines, fmt.Sprintf("%s = {", k))
        for ek, e := range v {
          lines = append(lines, fmt.Sprintf("  %s = %s,", ek, FormatJSON(e)))
        }
        lines = append(lines, "}")
      case []interface{}:
        lines = append(lines, fmt.Sprintf("%s = [", k))
        for _, e := range v {
          lines = append(lines, fmt.Sprintf("  %s,", FormatJSON(e)))
        }
        lines = append(lines, "]")
      case interface{}:
        lines = append(lines, fmt.Sprintf("%s = %s", k, FormatJSON(v)))
      }

    } else {
      // If there is no mapping, append it to raw DCOS config vars
      rawDcosConfig[k] = iv
    }
  }

  if len(lines) > 0 {
    lines = append([]string{""}, lines...)
  }

  if len(rawDcosConfig) > 0 {
    bytes, err := yaml.Marshal(rawDcosConfig)
    if err != nil {
      return nil, fmt.Errorf("Could not encode raw DC/OS options: %s", err.Error())
    }

    lines = append(lines, "", "dcos_config = <<EOF")
    lines = append(lines, strings.Split(string(bytes), "\n")...)
    lines = append(lines, "EOF")
  }

  return lines, nil
}

func (p *PluginImportClusterCmdImport) importTags(cfg *DcosLaunchInputConfig, project *ProjectSandbox) ([]string, error) {
  var lines []string = nil

  if cfg.Tags != nil && len(cfg.Tags) > 0 {
    lines = append(lines, "", "tags = {")

    for k, v := range cfg.Tags {
      lines = append(lines, fmt.Sprintf("  %s = %s,", k, FormatJSON(v)))
    }

    lines = append(lines, "}")
  }

  return lines, nil
}

func (p *PluginImportClusterCmdImport) importAws(cfg *DcosLaunchInputConfig, project *ProjectSandbox) ([]string, error) {
  var lines []string = nil

  if cfg.InstallPrereqs {
    PrintWarning("Ignoring `install_prereqs` since it's always implied")
  }

  if cfg.DeploymentName != "" {
    lines = append(
      lines,
      fmt.Sprintf(`cluster_name = "%s"`, cfg.DeploymentName),
    )
  }

  if cfg.OsName != "" {
    PrintWarning("Consider removing `os_name` if you are not using a customized DC/OS AMI. " +
      "The Universal Installer already provides the recommended default.")
    lines = append(
      lines,
      fmt.Sprintf(`aws_ami = "%s"`, cfg.OsName),
    )
  }

  // Guess DC/OS version
  if cfg.DcosInstallerUrl != "" {
    lines = append(
      lines,
      fmt.Sprintf(`custom_dcos_download_path = "%s"`, cfg.DcosInstallerUrl),
    )
  } else if cfg.DcosVersion != "" {
    lines = append(
      lines,
      fmt.Sprintf(`dcos_version = "%s"`, cfg.DcosVersion),
    )
  } else {
    lines = append(
      lines,
      fmt.Sprintf(`dcos_version = "%s"`, GetLatestDCOSVersion("open", "2.0.0")),
    )
  }

  // Guess DC/OS variant
  if cfg.DcosConfig == nil {
    lines = append(lines, fmt.Sprintf(`dcos_variant = "open"`))
  } else {
    if _, ok := cfg.DcosConfig["variant"]; !ok {
      eeFlag := false

      // If we have a license key, assume that's an enterprise variant
      if _, ok := cfg.DcosConfig["license_key_contents"]; ok {
        eeFlag = true
      } else {
        // Otherwise, check if we are using a custom installer, and it contains
        // a pointer to an enterprise release. Older DC/OS versions did not
        // require an enterprise license to run.
        if cfg.DcosInstallerUrl != "" {
          if strings.Contains(cfg.DcosInstallerUrl, ".ee.") {
            eeFlag = true
          }
        }
      }

      if eeFlag {
        lines = append(lines, fmt.Sprintf(`dcos_variant = "ee"`))
      } else {
        lines = append(lines, fmt.Sprintf(`dcos_variant = "open"`))
      }
    }
  }

  if cfg.InstanceType != "" {
    lines = append(
      lines,
      fmt.Sprintf(`masters_instance_type = "%s"`, cfg.InstanceType),
      fmt.Sprintf(`private_agents_instance_type = "%s"`, cfg.InstanceType),
      fmt.Sprintf(`public_agents_instance_type = "%s"`, cfg.InstanceType),
    )
  }

  if cfg.NumPublicAgents != 0 {
    lines = append(
      lines,
      fmt.Sprintf(`num_public_agents = %d`, cfg.NumPublicAgents),
    )
  }
  if cfg.NumPrivateAgents != 0 {
    lines = append(
      lines,
      fmt.Sprintf(`num_private_agents = %d`, cfg.NumPrivateAgents),
    )
  }
  if cfg.NumMasters != 0 {
    lines = append(
      lines,
      fmt.Sprintf(`num_masters = %d`, cfg.NumMasters),
    )
  }

  return lines, nil
}

func (p *PluginImportClusterCmdImport) impotExtraVolumes(cfg *DcosLaunchInputConfig, project *ProjectSandbox) ([]string, error) {
  var volLines []string = nil

  for _, m := range cfg.AwsBlockDeviceMappings {
    expr := fmt.Sprintf("%#v", m)

    //
    // Mapping from Boto:
    // > https://boto3.amazonaws.com/v1/documentation/api/latest/reference/services/ec2.html#EC2.Client.create_image
    // To Terraform:
    // > https://www.terraform.io/docs/providers/aws/r/ebs_volume.html#argument-reference
    // Through the machinery:
    // > https://github.com/dcos-terraform/terraform-aws-instance/blob/support/0.2.x/main.tf#L123
    //
    if devName, ok := m["DeviceName"]; ok {
      if devNameStr, ok := devName.(string); ok {
        if devEbs, ok := m["Ebs"]; ok {
          if devEbsMap, ok := devEbs.(map[string]interface{}); ok {
            xSize := 0
            xIops := 0
            xType := ""

            if vSz, ok := devEbsMap["VolumeSize"]; ok {
              if vSzInt, ok := vSz.(int); ok {
                xSize = vSzInt
              } else {
                PrintWarning("Error in volume '%s': 'VolumeSize' is not an integer", expr)
              }
            }
            if vIops, ok := devEbsMap["Iops"]; ok {
              if vIopsInt, ok := vIops.(int); ok {
                xIops = vIopsInt
              } else {
                PrintWarning("Error in volume '%s': 'Iops' is not an integer", expr)
              }
            }
            if vType, ok := devEbsMap["VolumeType"]; ok {
              if vTypeStr, ok := vType.(string); ok {
                xType = vTypeStr
              } else {
                PrintWarning("Error in volume '%s': 'VolumeType' is not a string", expr)
              }
            }
            if _, ok := devEbsMap["SnapshotId"]; ok {
              PrintWarning("Error in volume '%s': 'SnapshotId' is not supported", expr)
            }
            if _, ok := devEbsMap["KmsKeyId"]; ok {
              PrintWarning("Error in volume '%s': 'KmsKeyId' is not supported", expr)
            }
            if _, ok := devEbsMap["Encrypted"]; ok {
              PrintWarning("Error in volume '%s': 'Encrypted' is not supported", expr)
            }
            if _, ok := devEbsMap["DeleteOnTermination"]; ok {
              PrintWarning("Ignoring 'DeleteOnTermination' on volume %s: Terraform will always remove it during destroy", devNameStr)
            }

            volLines = append(volLines, "{")
            volLines = append(volLines, fmt.Sprintf("device_name = %s", FormatJSON(devNameStr)))
            if xSize > 0 {
              volLines = append(volLines, fmt.Sprintf("size = %d", xSize))
            }
            if xIops > 0 {
              volLines = append(volLines, fmt.Sprintf("iops = %d", xIops))
            }
            if xType != "" {
              volLines = append(volLines, fmt.Sprintf("type = %s", FormatJSON(xType)))
            }
            volLines = append(volLines, "},")

          } else {
            PrintWarning("Not importing volume '%s': Invalid 'Ebs'", expr)
          }
        } else {
          PrintWarning("Not importing volume '%s': Missing 'Ebs'", expr)
        }
      } else {
        PrintWarning("Not importing volume '%s': Invalid 'DeviceName'", expr)
      }
    } else {
      PrintWarning("Not importing volume '%s': Missing 'DeviceName'", expr)
    }
  }

  var lines []string
  if len(volLines) > 0 {
    lines = append(lines, "private_agents_extra_volumes = [")
    lines = append(lines, volLines...)
    lines = append(lines, "]")
    lines = append(lines, "public_agents_extra_volumes = [")
    lines = append(lines, volLines...)
    lines = append(lines, "]")
  }

  return lines, nil
}

func (p *PluginImportClusterCmdImport) importOnpremAws(inputConfig *DcosLaunchInputConfig, project *ProjectSandbox) ([]string, error) {
  var cfgLines []string = nil

  // Import sections
  chunk, err := p.importSSHKeys(inputConfig, project)
  if err != nil {
    return nil, err
  } else {
    cfgLines = append(cfgLines, chunk...)
  }

  chunk, err = p.importAws(inputConfig, project)
  if err != nil {
    return nil, err
  } else {
    cfgLines = append(cfgLines, chunk...)
  }

  chunk, err = p.importDcosConfig(inputConfig.DcosConfig, project)
  if err != nil {
    return nil, err
  } else {
    cfgLines = append(cfgLines, chunk...)
  }

  chunk, err = p.impotExtraVolumes(inputConfig, project)
  if err != nil {
    return nil, err
  } else {
    cfgLines = append(cfgLines, chunk...)
  }

  chunk, err = p.importTags(inputConfig, project)
  if err != nil {
    return nil, err
  } else {
    cfgLines = append(cfgLines, chunk...)
  }

  return cfgLines, nil
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

  cfgFilename := fSet.Args()[0]

  PrintInfo("%s %s", "Importing dcos-lauch config YAML from", Bold(cfgFilename))

  configContents, err := ioutil.ReadFile(cfgFilename)
  if err != nil {
    return fmt.Errorf("Could not load %s: %s", cfgFilename, err.Error())
  }

  var inputConfig DcosLaunchInputConfig
  err = yaml.Unmarshal(configContents, &inputConfig)
  if err != nil {
    return fmt.Errorf("Could not parse %s: %s", cfgFilename, err.Error())
  }

  if inputConfig.DeploymentName != "" {
    fileName = fmt.Sprintf("cluster-%s.tf", inputConfig.DeploymentName)
  }

  // We currently only support provider: onprem and platform: aws
  if inputConfig.Provider != "onprem" {
    return fmt.Errorf("Unsupported provider '%s' we only support: onprem", inputConfig.Provider)
  }
  if inputConfig.Platform != "aws" {
    return fmt.Errorf("Unsupported platform '%s' we only support: aws", inputConfig.Platform)
  }

  if inputConfig.GenconfDir != "" {
    return fmt.Errorf("Custom `genconf_dir` is not supported with terraform")
  }

  cfgLines, err := p.importOnpremAws(&inputConfig, project)
  if err != nil {
    return err
  }

  // Collect default lines
  awsRegion := "us-west-2"
  if inputConfig.AwsRegion != "" {
    awsRegion = inputConfig.AwsRegion
  }
  preLines := []string{
    `provider "aws" {`,
    `  # Change your default region here`,
    fmt.Sprintf(`  region = "%s"`, awsRegion),
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
  bodyLines := []string{
    `  admin_ips                  = ["${data.http.whatismyip.body}/32"]`,
  }

  // The output variables must match the JSON file returned by dcos-launch
  postLines := []string{
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

  allLines := append(preLines, bodyLines...)
  allLines = append(allLines, cfgLines...)
  allLines = append(allLines, postLines...)

  contents := []byte(strings.Join(allLines, "\n"))

  PrintInfo("%s%s%s", Bold("Writing "), Bold(Green(fileName)), Bold(" containing information for deploying a DC/OS cluster on AWS"))
  return project.WriteFormattedTerraformFile(fileName, contents)
}
