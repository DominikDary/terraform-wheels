package main

import (
	"fmt"
	"os"
	"strings"

	. "github.com/mesosphere-incubator/terraform-launch/utils"
)

func showHelpFor(cmd string) {
	me := os.Args[0]

	if cmd == "init" {
		fmt.Println("")
		fmt.Println("You are now ready to deploy a DC/OS cluster. Have a look on main.tf")
		fmt.Println("and adjust it according to your needs. You can find more information")
		fmt.Println("regarding the available options here:")
		fmt.Println("")
		fmt.Println("https://github.com/dcos-terraform/terraform-aws-dcos#dcos-on-aws")
		fmt.Println("")
		fmt.Println("When you are satisfied with your configuration, you can deploy a cluster")
		fmt.Println("using the following commands:")
		fmt.Println()
		fmt.Printf("  %s plan -out=plan.out\n", me)
		fmt.Printf("  %s apply plan.out\n", me)
		fmt.Println()
	}

	if cmd == "plan" {
		fmt.Println("")
		fmt.Println("Terraform has now created a plan file, containing all the operation")
		fmt.Println("it has to perform in order to reach your configuration state. Use the")
		fmt.Println("following command to actually apply them:")
		fmt.Println("")
		fmt.Printf("  %s apply plan.out", me)
		fmt.Println("")
	}
}

func showHelp() {
	fmt.Println("")
	fmt.Println("DC/OS Commands:")
	fmt.Println("    dcos               Builds or changes infrastructure")
}

func shouldShowHelp(args []string) bool {
	for _, arg := range args {
		if strings.Contains(arg, "help") {
			return true
		}
	}
	return false
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		FatalError(err)
	}

	sandbox, err := OpenSandbox(cwd)
	if err != nil {
		FatalError(err)
	}

	isEmpty, err := sandbox.IsEmpty()
	if err != nil {
		FatalError(err)
	}

	hasTfFiles, err := sandbox.HasTerraformFiles()
	if err != nil {
		FatalError(err)
	}

	prepared := false
	if (len(os.Args) > 1) && (os.Args[1] == "init") {
		if isEmpty {
			err = sandbox.InitProject()
			if err != nil {
				FatalError(err)
			}
			prepared = true
		} else if !hasTfFiles {
			err = sandbox.InitProject()
			if err != nil {
				FatalError(err)
			}
			prepared = true
		}
	}

	if (isEmpty || !hasTfFiles) && !prepared {
		FatalError(fmt.Errorf("Your current directory does not contain terraform files. Please run `init` to prepare it."))
	}

	sshagent, err := CreateSSHAgentWrapper()
	if err != nil {
		FatalError(err)
	}

	tf, err := sandbox.GetTerraform(sshagent)
	if err != nil {
		FatalError(err)
	}

	if !IsAWSCredsOK() {
		FatalError(fmt.Errorf("Could not find (still valid) AWS credentials in your enviroment. Use `maws login` and make sure to export the AWS_PROFILE"))
	}

	err = sshagent.Start()
	if err != nil {
		FatalError(err)
	}
	if sandbox.HasFile("cluster-key") {
		err = sshagent.AddKey(sandbox.GetFilePath("cluster-key"))
		if err != nil {
			FatalError(err)
		}
	} else {
		FatalError(fmt.Errorf("The script currently requires the private cluster key to be saved in a file called 'cluster-key'"))
	}

	tf.Invoke(os.Args[1:])

	// Inject our additional commands to terraform
	if (len(os.Args) == 1) || shouldShowHelp(os.Args) {
		showHelp()
	}

	err = sshagent.Stop()
	if err != nil {
		FatalError(err)
	}

}
