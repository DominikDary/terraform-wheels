package plugins

import (
  "fmt"
  "strings"

  . "github.com/logrusorgru/aurora"
  . "github.com/mesosphere-incubator/terraform-wheels/utils"
)

type PluginDcosProvider struct {
}

func CreatePluginDcosProvider() *PluginDcosProvider {
  return &PluginDcosProvider{}
}

func (p *PluginDcosProvider) GetName() string {
  return "dcos-provider"
}

func (p *PluginDcosProvider) IsUsed(project *ProjectSandbox) (bool, error) {
  dcos_data := project.GetTerraformResourcesMatchingName("data", "dcos_*")
  dcos_resource := project.GetTerraformResourcesMatchingName("resource", "dcos_*")
  return len(dcos_data) > 0 || len(dcos_resource) > 0, nil
}

func (p *PluginDcosProvider) BeforeRun(project *ProjectSandbox, tf *TerraformWrapper, initRun bool) error {

  // If we are missing a DC/OS provider file, create it now
  provider := project.GetTerraformResourcesMatchingName("provider", "dcos")
  if len(provider) == 0 {

    content := []byte(strings.Join(p.getProviderContents(project), "\n"))
    filename := "provider-dcos.tf"

    err := project.WriteFormattedTerraformFile(filename, content)
    if err != nil {
      return err
    }

    PrintInfo("You are using dcos_ resources but you don't have a DC/OS provider. I created %s for you, please have a look", Bold(filename))
  }

  return nil
}

func (p *PluginDcosProvider) AfterRun(project *ProjectSandbox, tf *TerraformWrapper, tfErr error) error {
  return nil
}

func (p *PluginDcosProvider) GetCommands() []PluginCommand {
  return []PluginCommand{}
}

func (p *PluginDcosProvider) getProviderContents(project *ProjectSandbox) []string {
  var cfg []string = []string{
    `// This connects to DC/OS and provides the dcos_* resources`,
    `provider "dcos" {`,
  }

  // Check if we also have a launch module
  mods := project.GetTerraformResourcesMatching("module", "source", "*dcos-terraform/dcos/aws")
  if len(mods) == 0 {
    // No launch module, we only rely on CLI
    cfg = append(cfg, "}")
    return cfg
  }

  // Get the first deployment module
  awsMod := mods[0]
  awsModName := awsMod["_name"].(string)
  cfg = append(cfg, fmt.Sprintf(`  dcos_url = "${module.%s.masters-loadbalancer}"`, awsModName))

  // Get variant
  variant := "open"
  if v, ok := awsMod["dcos_variant"]; ok {
    variant = v.(string)
  }

  // If we have an ee variant, we can have password
  if variant == "ee" {
    cfg = append(cfg, fmt.Sprintf(`  user = "bootstrapuser"`))
    cfg = append(cfg, fmt.Sprintf(`  password = "deleteme"`))
  }

  cfg = append(cfg, "}")
  return cfg
}
