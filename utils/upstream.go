package utils

import (
  "fmt"
  "runtime"
)

var upstreamTerraformVersion string = "0.11.14"

func upstreamGetTerraform() (string, string, error) {
  if runtime.GOOS == "linux" {
    if runtime.GOARCH == "amd64" {
      return "https://releases.hashicorp.com/terraform/0.11.14/terraform_0.11.14_linux_amd64.zip",
        "9b9a4492738c69077b079e595f5b2a9ef1bc4e8fb5596610f69a6f322a8af8dd",
        nil
    } else if runtime.GOARCH == "386" {
      return "https://releases.hashicorp.com/terraform/0.11.14/terraform_0.11.14_linux_386.zip",
        "0b6b2c61b80a35646df2cb7d443efeba3f4dedcdecbabab3b2626c2ea8976e87",
        nil
    }
  } else if runtime.GOOS == "darwin" {
    if runtime.GOARCH == "amd64" {
      return "https://releases.hashicorp.com/terraform/0.11.14/terraform_0.11.14_darwin_amd64.zip",
        "829bdba148afbd61eab4aafbc6087838f0333d8876624fe2ebc023920cfc2ad5",
        nil
    }
  }

  return "", "", fmt.Errorf("You are running an unsupported OS/Arch combination")
}
