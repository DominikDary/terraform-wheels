package utils

import (
	"fmt"
	"os"

	. "github.com/logrusorgru/aurora"
)

func FatalError(err error) {
	fmt.Printf("%s %s\n", Red("Error:"), err.Error())
	os.Exit(1)
}

func PrintInfo(format string, a ...interface{}) {
	args := append([]interface{}{Cyan("Info: ")}, a...)
	fmt.Printf("%s "+format+"\n", args...)
}

func PrintWarning(format string, a ...interface{}) {
	args := append([]interface{}{Bold(Yellow("Warn: "))}, a...)
	fmt.Printf("%s "+format+"\n", args...)
}
