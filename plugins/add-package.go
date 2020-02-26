package plugins

import (
  "flag"
  "fmt"
  "os"
  "strings"

  . "github.com/logrusorgru/aurora"
  . "github.com/mesosphere-incubator/terraform-wheels/utils"
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

func (p *PluginAddService) BeforeRun(project *ProjectSandbox, tf *TerraformWrapper, initRun bool) error {
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
  return "add-package"
}

func (p *PluginAddServiceCmdAddService) GetDescription() string {
  return "Adds a configuration file to deploy a package from Universe on DC/OS"
}

func (p *PluginAddServiceCmdAddService) Handle(args []string, project *ProjectSandbox, tf *TerraformWrapper) error {

  fSet := flag.NewFlagSet(p.GetName(), flag.ContinueOnError)
  fServiceName := fSet.String("name", "", "The name of the service to deploy")
  fPackageName := fSet.String("package", "", "The name of the service to deploy")
  fPackageVersion := fSet.String("version", "latest", "The version of the package to install")
  fConfig := fSet.String("config", "", "Optional path to a configuration file to import")
  fAppId := fSet.String("appid", "", "The ID of the application to assign when deployed on DC/OS")

  help := fSet.Bool("help", false, "Show this help message")
  fSet.BoolVar(help, "h", false, "Show this help message")
  err := fSet.Parse(args)
  if err != nil {
    FatalError(err)
  }

  if *help {
    fmt.Printf("Usage: %s %s [-help] [options]\n", os.Args[0], Bold(p.GetName()))
    fmt.Println("")
    fmt.Println("This command will generate a service-xxx.tf file in the project directory")
    fmt.Println("that describes a deployment of a universe service on DC/OS.")
    fmt.Println("")
    fmt.Println("Options:")
    fSet.PrintDefaults()
    return nil
  }

  if *fPackageName == "" {
    fSet.PrintDefaults()
    return fmt.Errorf("Please specify the package name with -package=")
  }
  if *fServiceName == "" {
    *fServiceName = *fPackageName
  }
  if *fAppId == "" {
    *fAppId = *fServiceName
  }

  var configLines []string
  if *fConfig != "" {
    configLines, err = LoadServiceJsonToConfigLines(*fConfig)
    if err != nil {
      return fmt.Errorf("Could not load config from %s: %s", *fConfig, err.Error())
    }
  }

  var fileName string = fmt.Sprintf("service-%s.tf", *fServiceName)
  var lines []string = []string{
    `// Specify which upstream repository to use for installing this package`,
    fmt.Sprintf(`resource "dcos_package_repo" "%s" {`, *fServiceName),
    `  name = "Universe"`,
    `  url  = "https://universe.mesosphere.com/repo"`,
    `}`,
    ``,
    `// Select the package version to deploy`,
    fmt.Sprintf(`data "dcos_package_version" "%s" {`, *fServiceName),
    fmt.Sprintf(`  repo_url = "${dcos_package_repo.%s.url}"`, *fServiceName),
    ``,
    fmt.Sprintf(`  name    = "%s"`, *fPackageName),
    fmt.Sprintf(`  version = "%s"`, *fPackageVersion),
    `}`,
    ``,
    `// Configure the service to deploy`,
    fmt.Sprintf(`data "dcos_package_config" "%s" {`, *fServiceName),
    fmt.Sprintf(`  version_spec = "${data.dcos_package_version.%s.spec}"`, *fServiceName),
  }
  lines = append(lines, configLines...)
  lines = append(lines, []string{
    `}`,
    ``,
    `// Deploy the service`,
    fmt.Sprintf(`module "%s" {`, *fServiceName),
    `  source = "github.com/mesosphere/data-services-terraform/modules/ds-deploy"`,
    ``,
    fmt.Sprintf(`  config          = "${data.dcos_package_config.%s.config}"`, *fServiceName),
    fmt.Sprintf(`  app_id          = "%s"`, *fAppId),
    fmt.Sprintf(`  service_account = "%s-principal"`, strings.ReplaceAll(*fAppId, "/", "__")),
    `}`,
  }...)

  PrintInfo("%s%s%s", Bold("Writing "), Bold(Green(fileName)), Bold(" containing information for deploying a service on top of DC/OS"))
  contents := []byte(strings.Join(lines, "\n") + "\n")
  return project.WriteFormattedTerraformFile(fileName, contents)
}
