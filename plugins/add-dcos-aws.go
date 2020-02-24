package plugins

import (
  "flag"
  "fmt"
  "os"
  "os/exec"
  "os/user"

  . "github.com/logrusorgru/aurora"
  . "github.com/mesosphere-incubator/terraform-launch/utils"
  "gopkg.in/hlandau/passlib.v1"
  "gopkg.in/hlandau/passlib.v1/abstract"
  "gopkg.in/hlandau/passlib.v1/hash/sha2crypt"
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
  // Check if we are using the AWS provider
  mods := project.GetTerraformResourcesMatching("module", "source", "*dcos-terraform/dcos/aws")
  return mods != nil, nil
}

func (p *PluginDcosAws) BeforeRun(project *ProjectSandbox, tf *TerraformWrapper, initRun bool) error {
  if !IsAWSCredsOK() {
    // Check if we have maws and a profile already set. In which case we are
    // going to transparently do a maws credential refresh
    _, err := exec.LookPath("maws")
    if err == nil {
      profile, ok := os.LookupEnv("AWS_PROFILE")
      if ok {
        PrintInfo("Your AWS credentials have expired, going to refresh them using %s", Bold("maws"))

        _, _, serr, err := ExecuteAndCollect([]string{}, "maws", "login", profile)
        if err != nil {
          fmt.Println(serr)
          FatalError(fmt.Errorf("Failed to login with `maws`, please retry manually"))
        }

        // We should be OK by now
        if !IsAWSCredsOK() {
          FatalError(fmt.Errorf("Failed to refresh credentials with `maws`, please retry manually"))
        }

        // Everything looks good, don't continue with the prompt
        return nil
      }
    }

    err = fmt.Errorf("Could not find (still valid) AWS credentials in your enviroment. Use `maws login` and make sure to export the AWS_PROFILE")
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
  tfc.Flags.String("public_agents_root_volume_type", "", "[PUBLIC AGENTS] Specify the root volume type.")
  tfc.Flags.String("public_agents_iam_instance_profile", "", "[PUBLIC AGENTS] Instance profile to be used for these instances")
  tfc.Flags.String("dcos_master_discovery", "", "The Mesos master discovery method. The available options are static or master_http_loadbalancer. (recommend the use of master_http_loadbalancer)")
  tfc.Flags.String("dcos_no_proxy", "", "A YAML nested list (-) of addresses to exclude from the proxy. (optional)")
  tfc.Flags.String("dcos_agent_list", "", "used to list the agents in the config.yaml (optional)")
  tfc.Flags.String("bootstrap_os", "", "[BOOTSTRAP] Operating system to use. Instead of using your own AMI you could use a provided OS.")
  tfc.Flags.String("dcos_ca_certificate_path", "", "[Enterprise DC/OS] Path (relative to the $DCOS_INSTALL_DIR) to a file containing a single X.509 CA certificate in the OpenSSL PEM format. (optional)")
  tfc.Flags.String("dcos_mesos_max_completed_tasks_per_framework", "", "The number of completed tasks for each framework that the Mesos master will retain in memory. (optional)")
  tfc.Flags.String("dcos_superuser_username", "", "[Enterprise DC/OS] set the superuser username (recommended)")
  tfc.Flags.String("dcos_zk_super_credentials", "", "[Enterprise DC/OS] set the zk super credentials (recommended)")
  tfc.Flags.String("dcos_staged_package_storage_uri", "", "Where to temporarily store DC/OS packages while they are being added. (optional)")
  tfc.Flags.String("bootstrap_iam_instance_profile", "", "[BOOTSTRAP] Instance profile to be used for these instances")
  tfc.Flags.String("public_agents_hostname_format", "", "[PUBLIC AGENTS] Format the hostname inputs are index+1, region, cluster_name")
  tfc.Flags.String("dcos_oauth_enabled", "", "[Open DC/OS Only] Indicates whether to enable authentication for your cluster. (optional)")
  tfc.Flags.String("dcos_s3_prefix", "", "name of the s3 prefix for the exhibitor backend (recommended but required with dcos_exhibitor_address)")
  tfc.Flags.String("dcos_adminrouter_tls_1_0_enabled", "", "Indicates whether to enable TLSv1 support in Admin Router. (optional)")
  tfc.Flags.String("dcos_enable_gpu_isolation", "", "Indicates whether to enable GPU support in DC/OS. (optional)")
  tfc.Flags.String("dcos_ucr_default_bridge_subnet", "", "IPv4 subnet allocated to the mesos-bridge CNI network for UCR bridge-mode networking. (optional)")
  tfc.Flags.String("dcos_telemetry_enabled", "", "Change the telemetry option (optional)")
  tfc.Flags.String("dcos_rexray_config", "", "The REX-Ray configuration method for enabling external persistent volumes in Marathon. (optional)")
  tfc.Flags.String("bootstrap_aws_ami", "", "[BOOTSTRAP] AMI to be used")
  tfc.Flags.String("public_agents_additional_ports", "", "List of additional ports allowed for public access on public agents (80 and 443 open by default)")
  tfc.Flags.String("ansible_bundled_container", "", "Docker container with bundled dcos-ansible and ansible executables")
  tfc.Flags.String("dcos_aws_template_storage_bucket", "", "AWS CloudFormation bucket name (optional)")
  tfc.Flags.String("dcos_master_dns_bindall", "", "Indicates whether the master DNS port is open. (optional)")
  tfc.Flags.String("private_agents_extra_volumes", "", "[PRIVATE AGENTS] Extra volumes for each private agent")
  tfc.Flags.String("masters_acm_cert_arn", "", "ACM certifacte to be used for the masters load balancer")
  tfc.Flags.String("dcos_aws_template_upload", "", "To automatically upload the customized advanced templates to your S3 bucket. (optional)")
  tfc.Flags.String("dcos_exhibitor_zk_path", "", "the filepath that Exhibitor uses to store data (not recommended but required with exhibitor_storage_backend set to zookeeper. Use aws_s3 or azureinstead. Assumes external ZooKeeper is already online.)")
  tfc.Flags.String("dcos_process_timeout", "", "The allowable amount of time, in seconds, for an action to begin after the process forks. (optional)")
  tfc.Flags.String("num_masters", "", "Specify the amount of masters. For redundancy you should have at least 3")
  tfc.Flags.String("bootstrap_hostname_format", "", "[BOOTSTRAP] Format the hostname inputs are index+1, region, cluster_name")
  tfc.Flags.String("dcos_ca_certificate_chain_path", "", "[Enterprise DC/OS] Path (relative to the $DCOS_INSTALL_DIR) to a file containing the complete CA certification chain required for end-entity certificate verification, in the OpenSSL PEM format. (optional)")
  tfc.Flags.String("dcos_mesos_container_log_sink", "", "The log manager for containers (tasks). The options are to send logs to: 'journald', 'logrotate', 'journald+logrotate'. (optional)")
  tfc.Flags.String("dcos_overlay_mtu", "", "The maximum transmission unit (MTU) of the Virtual Ethernet (vEth) on the containers that are launched on the overlay. (optional)")
  tfc.Flags.String("dcos_overlay_network", "", "This group of parameters define an virtual network for DC/OS. (optional)")
  tfc.Flags.String("dcos_cluster_docker_registry_url", "", "The custom URL that Mesos uses to pull Docker images from. If set, it will configure the Mesos --docker_registry flag to the specified URL. (optional)")
  tfc.Flags.String("aws_ami", "", "AMI that will be used for the instances instead of the Mesosphere chosen default images. Custom AMIs must fulfill the Mesosphere DC/OS system-requirements: See https://docs.mesosphere.com/1.12/installing/production/system-requirements/")
  tfc.Flags.String("private_agents_root_volume_size", "", "[PRIVATE AGENTS] Root volume size in GB")
  tfc.Flags.String("dcos_https_proxy", "", "https proxy (optional)")
  tfc.Flags.String("dcos_gc_delay", "", "The maximum amount of time to wait before cleaning up the executor directories (optional)")
  tfc.Flags.String("bootstrap_associate_public_ip_address", "", "[BOOTSTRAP] Associate a public ip address with there instances")
  tfc.Flags.String("dcos_variant", "", "Specifies which DC/OS variant it should be: `open` (Open Source) or `ee` (Enterprise Edition)")
  tfc.Flags.String("dcos_aws_region", "", "AWS region for exhibitor storage (optional but required with dcos_exhibitor_address)")
  tfc.Flags.String("num_of_private_agents", "", "Specify the amount of private agents. These agents will provide your main resources")
  tfc.Flags.String("dcos_customer_key", "", "[Enterprise DC/OS] sets the customer key (optional)")
  tfc.Flags.String("dcos_dns_bind_ip_blacklist", "", "A list of IP addresses that DC/OS DNS resolvers cannot bind to. (optional)")
  tfc.Flags.String("num_public_agents", "", "Specify the amount of public agents. These agents will host marathon-lb and edgelb")
  tfc.Flags.String("tags", "", "Add custom tags to all resources (use key=value format)")
  tfc.Flags.String("bootstrap_root_volume_size", "", "[BOOTSTRAP] Root volume size in GB")
  tfc.Flags.String("dcos_adminrouter_tls_1_1_enabled", "", "Indicates whether to enable TLSv1.1 support in Admin Router. (optional)")
  tfc.Flags.String("dcos_ca_certificate_key_path", "", "[Enterprise DC/OS] Path (relative to the $DCOS_INSTALL_DIR) to a file containing a single X.509 certificate private key in the OpenSSL PEM format. (optional)")
  tfc.Flags.String("dcos_gpus_are_scarce", "", "Indicates whether to treat GPUs as a scarce resource in the cluster. (optional)")
  tfc.Flags.String("with_replaceable_masters", "", "Create S3 bucket for exhibitor and configure DC/OS to use it.")
  tfc.Flags.String("dcos_custom_checks", "", "Custom installation checks that are added to the default check configuration process. (optional)")
  tfc.Flags.String("masters_internal_acm_cert_arn", "", "ACM certifacte to be used for the internal masters load balancer")
  tfc.Flags.String("dcos_exhibitor_storage_backend", "", "options are static, aws_s3, azure, or zookeeper (recommended)")
  tfc.Flags.String("dcos_cluster_name", "", "sets the DC/OS cluster name")
  tfc.Flags.String("dcos_enable_docker_gc", "", "Indicates whether to run the docker-gc script, a simple Docker container and image garbage collection script, once every hour to clean up stray Docker containers. (optional)")
  tfc.Flags.String("ssh_public_key_file", "", "Path to SSH public key. This is mandatory but can be set to an empty string if you want to use ssh_public_key with the key as string.")
  tfc.Flags.String("masters_aws_ami", "", "[MASTERS] AMI to be used")
  tfc.Flags.String("public_agents_access_ips", "", "List of ips allowed access to public agents. admin_ips are joined to this list (use multiple times to add multiple values)")
  tfc.Flags.String("public_agents_acm_cert_arn", "", "ACM certifacte to be used for the public agents load balancer")
  tfc.Flags.String("dcos_master_list", "", "statically set your master nodes (not recommended but required with exhibitor_storage_backend set to static. Use aws_s3 or azure instead, that way you can replace masters in the cloud.)")
  tfc.Flags.String("dcos_calico_veth_mtu", "", "The MTU to set on the veth pair devices. (optional)")
  tfc.Flags.String("private_agents_user_data", "", "[PRIVATE AGENTS] User data to be used on these instances (cloud-init)")
  tfc.Flags.String("accepted_internal_networks", "", "Subnet ranges for all internal networks (use multiple times to add multiple values)")
  tfc.Flags.String("dcos_master_external_loadbalancer", "", "Allows DC/OS to configure certs around the External Load Balancer name. If not used SSL verfication issues will arrise. EE only. (recommended)")
  tfc.Flags.String("dcos_adminrouter_tls_cipher_suite", "", "[Enterprise DC/OS] Indicates whether to allow web browsers to send the DC/OS authentication cookie through a non-HTTPS connection. (optional)")
  tfc.Flags.String("dcos_auth_cookie_secure_flag", "", "[Enterprise DC/OS] allow web browsers to send the DC/OS authentication cookie through a non-HTTPS connection. (optional)")
  tfc.Flags.String("dcos_calico_vxlan_vni", "", "The virtual network ID used for calico VXLAN. (optional)")
  tfc.Flags.String("masters_instance_type", "", "[MASTERS] Instance type")
  tfc.Flags.String("private_agents_os", "", "[PRIVATE AGENTS] Operating system to use. Instead of using your own AMI you could use a provided OS.")
  tfc.Flags.String("public_agents_root_volume_size", "", "[PUBLIC AGENTS] Root volume size")
  tfc.Flags.String("dcos_previous_version", "", "DC/OS 1.9+ requires users to set this value to ensure users know the version. Terraform helps populate this value, but users can override it here. (recommended)")
  tfc.Flags.String("dcos_ip_detect_public_filename", "", "statically set your detect-ip-public path")
  tfc.Flags.String("bootstrap_root_volume_type", "", "[BOOTSTRAP] Root volume type")
  tfc.Flags.String("masters_user_data", "", "[MASTERS] User data to be used on these instances (cloud-init)")
  tfc.Flags.String("ansible_additional_config", "", "Add additional config options to ansible. This is getting merged with generated defaults. Do not specify `dcos:`")
  tfc.Flags.String("dcos_exhibitor_zk_hosts", "", "a comma-separated list of one or more ZooKeeper node IP and port addresses to use for configuring the internal Exhibitor instances. (not recommended but required with exhibitor_storage_backend set to ZooKeeper. Use aws_s3 or azure instead. Assumes external ZooKeeper is already online.)")
  tfc.Flags.String("cluster_name", "", "Name of the DC/OS cluster")
  tfc.Flags.String("masters_os", "", "[MASTERS] Operating system to use. Instead of using your own AMI you could use a provided OS.")
  tfc.Flags.String("masters_root_volume_size", "", "[MASTERS] Root volume size in GB")
  tfc.Flags.String("private_agents_associate_public_ip_address", "", "[PRIVATE AGENTS] Associate a public ip address with there instances")
  tfc.Flags.String("private_agents_hostname_format", "", "[PRIVATE AGENTS] Format the hostname inputs are index+1, region, cluster_name")
  tfc.Flags.String("cluster_name_random_string", "", "Add a random string to the cluster name")
  tfc.Flags.String("dcos_l4lb_enable_ipv6", "", "A boolean that indicates if layer 4 load balancing is available for IPv6 networks. (optional)")
  tfc.Flags.String("dcos_overlay_config_attempts", "", "Specifies how many failed configuration attempts are allowed before the overlay configuration modules stop trying to configure an virtual network. (optional)")
  tfc.Flags.String("num_private_agents", "", "Specify the amount of private agents. These agents will provide your main resources")
  tfc.Flags.String("masters_associate_public_ip_address", "", "[MASTERS] Associate a public ip address with there instances")
  tfc.Flags.String("public_agents_associate_public_ip_address", "", "[PUBLIC AGENTS] Associate a public ip address with there instances")
  tfc.Flags.String("additional_private_agent_ips", "", "Additional private agent IPs.")
  tfc.Flags.String("dcos_version", "", "Specifies which DC/OS version instruction to use. Options: 2.0.0, 1.13.6, 1.12.4, 1.11.12, etc. See https://versions.d2iq.com/version for a full list.")
  tfc.Flags.String("dcos_resolvers", "", "A YAML nested list (-) of DNS resolvers for your DC/OS cluster nodes. (recommended)")
  tfc.Flags.String("dcos_exhibitor_azure_account_name", "", "the azure account name for exhibitor storage (optional but required with dcos_exhibitor_address)")
  tfc.Flags.String("dcos_calico_vxlan_port", "", "The UDP port used for calico VXLAN. This configuration works when dcos_calico_vxlan_enabled is set to be true. (optional)")
  tfc.Flags.String("dcos_fault_domain_detect_contents", "", "[Enterprise DC/OS] fault domain script contents. Optional but required if no fault-domain-detect script present.")
  tfc.Flags.String("dcos_dns_search", "", "A space-separated list of domains that are tried when an unqualified domain is entered. (optional)")
  tfc.Flags.String("dcos_docker_remove_delay", "", "The amount of time to wait before removing stale Docker images stored on the agent nodes and the Docker image generated by the installer. (optional)")
  tfc.Flags.String("dcos_ip_detect_public_contents", "", "Allows DC/OS to be aware of your publicly routeable address for ease of use (recommended)")
  tfc.Flags.String("bootstrap_instance_type", "", "[BOOTSTRAP] Instance type")
  tfc.Flags.String("dcos_exhibitor_address", "", "The address of the load balancer in front of the masters (recommended)")
  tfc.Flags.String("num_of_public_agents", "", "Specify the amount of public agents. These agents will host marathon-lb and edgelb")
  tfc.Flags.String("dcos_rexray_config_method", "", "The REX-Ray configuration method for enabling external persistent volumes in Marathon. (optional)")
  tfc.Flags.String("dcos_bootstrap_port", "", "Port of the bootstrap URL")
  tfc.Flags.String("additional_windows_private_agent_passwords", "", "Additional windows private agent passwords to be used for WinRM")
  tfc.Flags.String("bootstrap_private_ip", "", "Private IP bootstrap nginx is listening on. Used to build the bootstrap URL.")
  tfc.Flags.String("dcos_use_proxy", "", "To enable use of proxy for internal routing (optional)")
  tfc.Flags.String("dcos_log_directory", "", "The path to the installer host logs from the SSH processes. (optional)")
  tfc.Flags.String("dcos_license_key_contents", "", "[Enterprise DC/OS] used to privide the license key of DC/OS for Enterprise Edition. Optional if license.txt is present on bootstrap node.")
  tfc.Flags.String("dcos_cluster_docker_credentials", "", "Dictionary of Docker credentials to pass. (optional)")
  tfc.Flags.String("dcos_calico_network_cidr", "", "Subnet allocated for calico")
  tfc.Flags.String("public_agents_os", "", "[PUBLIC AGENTS] Operating system to use. Instead of using your own AMI you could use a provided OS.")
  tfc.Flags.String("additional_public_agent_ips", "", "Additional public agent IPs.")
  tfc.Flags.String("additional_windows_private_agent_ips", "", "Additional windows private agent IPs")
  tfc.Flags.String("dcos_num_masters", "", "set the num of master nodes (required with exhibitor_storage_backend set to aws_s3, azure, ZooKeeper)")
  tfc.Flags.String("dcos_cluster_docker_credentials_dcos_owned", "", "Indicates whether to store the credentials file in /opt/mesosphere or /etc/mesosphere/docker_credentials. A sysadmin cannot edit /opt/mesosphere directly (optional)")
  tfc.Flags.String("custom_dcos_download_path", "", "insert location of dcos installer script (optional)")
  tfc.Flags.String("dcos_package_storage_uri", "", "Where to permanently store DC/OS packages. The value must be a file URL. (optional)")
  tfc.Flags.String("dcos_enable_mesos_input_plugin", "", "Indicates whether to enable Telegraf's Mesos input plugin to collect Mesos metrics from Mesos masters and agents. Options: `true` or `false` (optional)")
  tfc.Flags.String("dcos_versions_service_url", "", "DC/OS Versions Service allows to identify DC/OS versions")
  tfc.Flags.String("dcos_calico_ipinip_mtu", "", "The MTU to set on the Calico IPIP tunnel device. (optional)")
  tfc.Flags.String("admin_ips", "", "List of CIDR admin IPs (use multiple times to add multiple values)")
  tfc.Flags.String("private_agents_root_volume_type", "", "[PRIVATE AGENTS] Root volume type")
  tfc.Flags.String("dcos_zk_agent_credentials", "", "[Enterprise DC/OS] set the ZooKeeper agent credentials (recommended)")
  tfc.Flags.String("dcos_http_proxy", "", "http proxy (optional)")
  tfc.Flags.String("dcos_cluster_docker_credentials_write_to_etc", "", "Indicates whether to write a cluster credentials file. (optional)")
  tfc.Flags.String("dcos_calico_vxlan_enabled", "", "Control whether IP-in-IP or VXLAN mode is used for calico. (optional)")
  tfc.Flags.String("dcos_dns_forward_zones", "", "Allow to forward DNS to certain domain requests to specific server. The following syntax must be used in combination with Terraform string heredoc. (optional) (:warning: DC/OS 1.10+)")
  tfc.Flags.String("dcos_audit_logging", "", "[Enterprise DC/OS] enable security decisions are logged for Mesos, Marathon, and Jobs. (optional)")
  tfc.Flags.String("dcos_cluster_docker_credentials_enabled", "", "Indicates whether to pass the Mesos --docker_config option to Mesos. (optional)")
  tfc.Flags.String("aws_key_name", "", "Specify the aws ssh key to use. We assume its already loaded in your SSH agent. Set ssh_public_key_file to empty string")
  tfc.Flags.String("public_agents_extra_volumes", "", "[PUBLIC AGENTS] Extra volumes for each public agent")
  tfc.Flags.String("dcos_aws_template_storage_secret_access_key", "", "AWS secret key for the CloudFormation template (optional)")
  tfc.Flags.String("dcos_aws_secret_access_key", "", "AWS secret key for exhibitor storage (optional but required with dcos_exhibitor_address)")
  tfc.Flags.String("dcos_s3_bucket", "", "name of the s3 bucket for the exhibitor backend (recommended but required with dcos_exhibitor_address)")
  tfc.Flags.String("dcos_rexray_config_filename", "", "The REX-Ray configuration filename for enabling external persistent volumes in Marathon. (optional)")
  tfc.Flags.String("dcos_overlay_enable", "", "Enable to disable overlay (optional)")
  tfc.Flags.String("private_agents_iam_instance_profile", "", "[PRIVATE AGENTS] Instance profile to be used for these instances")
  tfc.Flags.String("ansible_user", "", "The Ansible user that is used to run the Ansible Tasks.")
  tfc.Flags.String("dcos_image_commit", "", "The commit hash for the build of DC/OS")
  tfc.Flags.String("dcos_config", "", "used to add any extra arguments in the config.yaml that are not specified here. (optional)")
  tfc.Flags.String("dcos_download_url_checksum", "", "Custom DC/OS download URL SHA256 Checksum. Empty string omits checking.")
  tfc.Flags.String("dcos_fault_domain_enabled", "", "[Enterprise DC/OS] used to control if fault domain is enabled")
  tfc.Flags.String("dcos_zk_master_credentials", "", "[Enterprise DC/OS] set the ZooKeeper master credentials (recommended)")
  tfc.Flags.String("dcos_previous_version_master_index", "", "Used to track the index of master for quering the previous DC/OS version during upgrading. (optional) applicable: 1.9+")
  tfc.Flags.String("dcos_cluster_docker_registry_enabled", "", "DC/OS cluster docker registry enabled")
  tfc.Flags.String("additional_windows_private_agent_os_user", "", "Additional windows private agent os user to be used for WinRM")
  tfc.Flags.String("dcos_bouncer_expiration_auth_token_days", "", "[Enterprise DC/OS] Sets the auth token time-to-live (TTL) for Identity and Access Management. (optional)")
  tfc.Flags.String("masters_hostname_format", "", "[MASTERS] Format the hostname inputs are index+1, region, cluster_name")
  tfc.Flags.String("public_agents_user_data", "", "[PUBLIC AGENTS] User data to be used on these instances (cloud-init)")
  tfc.Flags.String("dcos_aws_template_storage_region_name", "", "AWS CloudFormation region name (optional)")
  tfc.Flags.String("dcos_exhibitor_explicit_keys", "", "set whether you are using AWS API keys to grant Exhibitor access to S3. (optional)")
  tfc.Flags.String("dcos_ip_detect_contents", "", "Allows DC/OS to detect your private address. Use this to pass this as an input to the module rather than a file in side your bootstrap node. (recommended)")
  tfc.Flags.String("dcos_calico_vxlan_mtu", "", "The MTU to set on the Calico VXLAN tunnel device. (optional)")
  tfc.Flags.String("masters_iam_instance_profile", "", "[MASTERS] Instance profile to be used for these instances")
  tfc.Flags.String("private_agents_aws_ami", "", "[PRIVATE AGENTS] AMI to be used")
  tfc.Flags.String("dcos_security", "", "[Enterprise DC/OS] set the security level of DC/OS, either 'strict' or 'permissive'. Default is 'permissive'. (recommended)")
  tfc.Flags.String("dcos_aws_template_storage_access_key_id", "", "AWS key ID for CloudFormation template storage (optional)")
  tfc.Flags.String("dcos_aws_access_key_id", "", "AWS key ID for exhibitor storage (optional but required with dcos_exhibitor_address)")
  tfc.Flags.String("dcos_adminrouter_tls_1_2_enabled", "", "Indicates whether to enable TLSv1.2 support in Admin Router. (optional)")
  tfc.Flags.String("dcos_public_agent_list", "", "statically set your public agents (not recommended)")
  tfc.Flags.String("availability_zones", "", "List of availability_zones to be used as the same format that are required by the platform/cloud providers. i.e `['RegionZone']` (use multiple times to add multiple values)")
  tfc.Flags.String("dcos_instance_os", "", "Operating system to use. Instead of using your own AMI you could use a provided OS.")
  tfc.Flags.String("private_agents_instance_type", "", "[PRIVATE AGENTS] Instance type")
  tfc.Flags.String("dcos_skip_checks", "", "Upgrade option: Used to skip all dcos checks that may block an upgrade if any DC/OS component is unhealthly. (optional) applicable: 1.10+")
  tfc.Flags.String("dcos_aws_template_storage_bucket_path", "", "AWS CloudFormation bucket path (optional)")
  tfc.Flags.String("dcos_exhibitor_azure_account_key", "", "the azure account key for exhibitor storage (optional but required with dcos_exhibitor_address)")
  tfc.Flags.String("ssh_public_key", "", "SSH public key in authorized keys format (e.g. 'ssh-rsa ..') to be used with the instances. Make sure you added this key to your ssh-agent.")
  tfc.Flags.String("public_agents_aws_ami", "", "[PUBLIC AGENTS] AMI to be used")
  tfc.Flags.String("public_agents_instance_type", "", "[PUBLIC AGENTS] Instance type")
  tfc.Flags.String("subnet_range", "", "Private IP space to be used in CIDR format")
  tfc.Flags.String("dcos_exhibitor_azure_prefix", "", "the azure account name for exhibitor storage (optional but required with dcos_exhibitor_address)")
  tfc.Flags.String("dcos_mesos_dns_set_truncate_bit", "", "Indicates whether to set the truncate bit if the response is too large to fit in a single packet. (optional)")
  tfc.Flags.String("dcos_check_time", "", "Check if Network Time Protocol (NTP) is enabled during DC/OS startup. (optional)")
  tfc.Flags.String("dcos_superuser_password_hash", "", "[Enterprise DC/OS] set the superuser password hash (recommended)")

  fPassword := tfc.Flags.String("dcos_superuser_password", "", "The plain-text password to encode")
  fOwner := tfc.Flags.String("owner", currUserStr, "The user-name that owns this cluster")
  fExpire := tfc.Flags.String("expiration", "1h", "How long to keep the cluster running before cloud-cleaner tears it down")

  tfc.ListFlags = []string{"public_agents_access_ips", "accepted_internal_networks", "admin_ips", "availability_zones"}
  tfc.MapFlags = []string{"tags"}
  tfc.IgnoreFlags = []string{"owner", "expiration", "dcos_superuser_password"}

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

  // Hash password if given as hash input
  if *fPassword != "" {
    ctx := &passlib.Context{
      Schemes: []abstract.Scheme{
        sha2crypt.NewCrypter512(656000),
      },
    }

    hash, err := ctx.Hash(*fPassword)
    if err != nil {
      return fmt.Errorf("Could not encode password: %s", err.Error())
    }

    tfc.Flags.Set("dcos_superuser_password_hash", hash)
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
    fmt.Sprintf(`    "expiration" = "%s"`, *fExpire),
    fmt.Sprintf(`    "owner"      = "%s"`, *fOwner),
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
