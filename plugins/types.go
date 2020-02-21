package plugins

import (
	. "github.com/mesosphere-incubator/terraform-launch/utils"
)

type PluginCommand interface {
	GetName() string
	GetDescription() string
	Handle(project *ProjectSandbox, tf *TerraformWrapper) error
}

type Plugin interface {
	GetName() string

	IsUsed(project *ProjectSandbox) (bool, error)
	BeforeRun(project *ProjectSandbox, tf *TerraformWrapper) error
	AfterRun(project *ProjectSandbox, tf *TerraformWrapper, tfErr error) error

	GetCommands() []PluginCommand
}
