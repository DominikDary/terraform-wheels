package plugins

import (
  "flag"
  "fmt"
  "os"

  . "github.com/logrusorgru/aurora"
  . "github.com/mesosphere-incubator/terraform-launch/utils"
)

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

func (p *PluginImportClusterCmdImport) Handle(args []string, project *ProjectSandbox, tf *TerraformWrapper) error {
  var fileName string = "cluster-aws.tf"

  fSet := flag.NewFlagSet(p.GetName(), flag.ContinueOnError)

  help := fSet.Bool("help", false, "Show this help message")
  fSet.BoolVar(help, "h", false, "Show this help message")
  err := fSet.Parse(args)
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
    fSet.PrintDefaults()
    return nil
  }

  contents := []byte{}

  PrintInfo("%s", Bold("Writing "+fileName+" containing information for deploying a DC/OS cluster on AWS"))

  return project.WriteFile(fileName, contents)
}
