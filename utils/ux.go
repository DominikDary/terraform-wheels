package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"

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

func ReadPrompt(message string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("%s: ", message)
	text, _ := reader.ReadString('\n')
	return text
}

func ReadYN(message string) bool {
	for {
		ans := strings.ToLower(ReadPrompt(message + " [y/n]"))
		if ans == "y" || ans == "yes" {
			return true
		}
		if ans == "n" || ans == "no" {
			return true
		}
		fmt.Println("\nInvalid option please specify 'yes' or 'no'")
	}
}
