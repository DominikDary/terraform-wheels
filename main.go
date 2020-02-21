package main

import (
	"fmt"
	"os"
	"strings"

	// . "github.com/mesosphere-incubator/terraform-launch/codegen"
	. "github.com/mesosphere-incubator/terraform-launch/plugins"
	. "github.com/mesosphere-incubator/terraform-launch/utils"
)

var plugins []Plugin = []Plugin{
	CreatePluginDcosAws(),
	CreatePluginSSHAgent(),
}

func showMissingTerraformHelp() {
	fmt.Println("Your system does not have terraform installed, or it's version is not")
	fmt.Printf("compatible with our %sx requirements. This means we cannot show you\n", RequiredTerraformVersionPrefix)
	fmt.Println("the terraform help screen. ")
	fmt.Println("")
	fmt.Println("This tool will automatically download the correct terraform version and")
	fmt.Println("place it in your current project directory when you try to use the")
	fmt.Println("following commands for the first time:")
}

func showPluginHelp() {
	fmt.Println("")
	fmt.Println("DC/OS Commands:")
	for _, plugin := range plugins {
		for _, cmd := range plugin.GetCommands() {
			fmt.Printf("    %-18s %s\n", cmd.GetName(), cmd.GetDescription())
		}
	}
}

func showInitUsage() {
	FatalError(fmt.Errorf("Your current directory does not contain terraform files. Please run `init` to prepare it."))
}

func shouldShowHelp(args []string) bool {
	for _, arg := range args {
		if strings.Contains(arg, "help") {
			return true
		}
	}
	return false
}

func showHelp(sandbox *ProjectSandbox) {
	// Show terraform help
	if sandbox.HasTerraform() {
		tf, err := sandbox.GetTerraform()
		if err != nil {
			FatalError(err)
		}
		tf.Invoke([]string{})
	} else {
		showMissingTerraformHelp()
	}

	// Show plugin help and exit
	showPluginHelp()
	os.Exit(1)
}

func invokeTerraform(sandbox *ProjectSandbox, tf *TerraformWrapper, plugins []Plugin, args []string) {
	isInit := false
	for _, arg := range args {
		if arg == "init" {
			isInit = true
			break
		}
	}

	// Pre-run
	for _, plugin := range plugins {
		err := plugin.BeforeRun(sandbox, tf, isInit)
		if err != nil {
			FatalError(fmt.Errorf("Could not start %s: %s", plugin.GetName(), err.Error()))
		}
	}

	// Run
	err := tf.Invoke(args)

	// Post-run
	for _, plugin := range plugins {
		perr := plugin.AfterRun(sandbox, tf, err)
		if perr != nil {
			FatalError(fmt.Errorf("Could not finalize %s: %s", plugin.GetName(), err.Error()))
		}
	}
}

func loadPlugins(sandbox *ProjectSandbox) []Plugin {
	var loadedPlugins []Plugin
	for _, plugin := range plugins {
		used, err := plugin.IsUsed(sandbox)
		if err != nil {
			FatalError(err)
		}

		if used {
			PrintInfo("Using plugin %s", plugin.GetName())
			loadedPlugins = append(loadedPlugins, plugin)
		}
	}

	return loadedPlugins
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		FatalError(err)
	}

	// Get a work directory sandbox
	sandbox, err := OpenSandbox(cwd)
	if err != nil {
		FatalError(err)
	}

	// sandbox.PrintVariableDefs()
	// return

	// isEmpty, err := sandbox.IsEmpty()
	// if err != nil {
	// 	FatalError(err)
	// }

	// Check the sandbox status
	hasTfFiles, err := sandbox.HasTerraformFiles()
	if err != nil {
		FatalError(err)
	}

	// Handle help prompt early
	if len(os.Args) <= 1 || strings.Contains(os.Args[1], "help") {
		showHelp(sandbox)
		return
	}

	// Check if this is a plugin command and delegate it to the respective handler
	if len(os.Args) > 1 {

		// Ignore flags until we find a command
		cmd_n := ""
		cmd_i := 0
		for i := 1; i < len(os.Args); i++ {
			if strings.HasPrefix(os.Args[i], "-") {
				continue
			}
			cmd_i = i
			cmd_n = os.Args[i]
			break
		}

		// If there was no command, show help
		if cmd_i == 0 {
			showHelp(sandbox)
			return
		}

		// Check if this is a plugin command
		for _, plugin := range plugins {
			for _, cmd := range plugin.GetCommands() {
				if cmd.GetName() == cmd_n {
					tf, err := sandbox.GetTerraform()
					if err != nil {
						FatalError(err)
					}

					err = cmd.Handle(os.Args[2:], sandbox, tf)
					if err != nil {
						FatalError(err)
					}

					nowHasTfFiles, err := sandbox.HasTerraformFiles()
					if err != nil {
						FatalError(err)
					}

					// If that's the first time we saw some tf files, take the opportunity
					// to run initialize, so the user has less things to do
					if !hasTfFiles && nowHasTfFiles {
						PrintInfo("Terraform project created, initializing now")

						err := sandbox.ReloadTerraformProject()
						if err != nil {
							FatalError(err)
						}

						loadedPlugins := loadPlugins(sandbox)
						invokeTerraform(sandbox, tf, loadedPlugins, []string{"init"})
					}

					return
				}
			}
		}
	}

	// Initialize terraform now
	tf, err := sandbox.GetTerraform()
	if err != nil {
		FatalError(err)
	}

	// Forward to terraform
	loadedPlugins := loadPlugins(sandbox)
	invokeTerraform(sandbox, tf, loadedPlugins, os.Args[1:])
}
