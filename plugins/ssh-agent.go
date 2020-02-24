package plugins

import (
  "fmt"
  "os"
  "strings"

  . "github.com/logrusorgru/aurora"
  . "github.com/mesosphere-incubator/terraform-launch/utils"
)

type PluginSSHAgent struct {
  agent *SSHAgentWrapper
}

func CreatePluginSSHAgent() *PluginSSHAgent {
  return &PluginSSHAgent{nil}
}

func (p *PluginSSHAgent) GetName() string {
  return "ssh-agent"
}

func (p *PluginSSHAgent) IsUsed(project *ProjectSandbox) (bool, error) {
  // We are loading the SSH-Agent plugin when the dcos-aws module is used
  // and has a public ssh key specified
  mods := project.GetTerraformResourcesMatching("module", "source", "*dcos-terraform/dcos/aws")
  used := false
  for _, mod := range mods {
    if _, ok := mod["ssh_public_key_file"]; ok {
      used = true
      break
    }
  }
  return used, nil
}

func (p *PluginSSHAgent) BeforeRun(project *ProjectSandbox, tf *TerraformWrapper, initRun bool) error {
  sshagent, err := CreateSSHAgentWrapper()
  if err != nil {
    return err
  }

  socketPath, err := project.GetTemporaryPath("tmp/ssh-agent.socket")
  if err != nil {
    return err
  }

  _, err = os.Stat(socketPath)
  if err == nil {
    err = os.Remove(socketPath)
    if err != nil {
      return fmt.Errorf("Could not delete old ssh-agent socket: %s", err.Error())
    }
  }

  err = sshagent.Start(socketPath)
  if err != nil {
    return err
  }

  p.agent = sshagent
  tf.SetEnv("SSH_AUTH_SOCK", sshagent.Socket)

  // Find the SSH keys used in the project
  var pubSSHKeys []string = nil
  mods := project.GetTerraformResourcesMatching("module", "source", "*dcos-terraform/dcos/aws")
  for _, mod := range mods {
    if sshKeyVar, ok := mod["ssh_public_key_file"]; ok {
      if sshKey, ok := sshKeyVar.(string); ok {
        pubSSHKeys = append(pubSSHKeys, sshKey)
      }
    }
  }

  // Validate keys
  for _, sshKey := range pubSSHKeys {
    // Check if this is a file in the sandbox that is just missing
    // in which case we will exploit the opportunity to create it
    if project.IsFileInSandbox(sshKey) && !project.HasFile(sshKey) {
      PrintInfo("Found a defined ssh key '%s', but missing from the project directory. Going to create a keypair for you", sshKey)

      fPrivateKey := project.GetFilePath(getPrivateKeyNameFromPub(sshKey))
      fPublicKey := project.GetFilePath(sshKey)
      err := CreateRSAKeyPair(fPrivateKey, fPublicKey)
      if err != nil {
        return fmt.Errorf("Could not create RSA keypair: %s", err.Error())
      }
    }

    // Try to deduce the private key from the public key
    privKey := getPrivateKeyNameFromPub(sshKey)
    _, err = os.Stat(privKey)
    if err != nil {
      return fmt.Errorf("Could not find private key for %s (searching for %s)", Bold(sshKey), privKey)
    }

    // Add it to the SSH agent
    PrintInfo("Loaded private key %s in ssh-agent", Bold(privKey))
    err = sshagent.AddKey(privKey)
    if err != nil {
      return err
    }
  }

  return nil
}

func (p *PluginSSHAgent) AfterRun(project *ProjectSandbox, tf *TerraformWrapper, tfErr error) error {
  err := p.agent.Stop()
  return err
}

func (p *PluginSSHAgent) GetCommands() []PluginCommand {
  return []PluginCommand{}
}

func (p *PluginSSHAgent) ensureSSHKey() {

}

func getPrivateKeyNameFromPub(name string) string {
  if strings.HasSuffix(name, ".pub") {
    return name[0 : len(name)-4]
  }

  return name + ".key"
}
