package utils

import (
  "bufio"
  "fmt"
  "io"
  "os"
  "strings"

  . "github.com/logrusorgru/aurora"
  . "github.com/mattn/go-colorable"
)

type OptionsPrinter interface {
  PrintDefaults()
  SetOutput(output io.Writer)
}

var colorableStdout = NewColorableStdout()
var colorableStderr = NewColorableStderr()

func FatalError(err error) {
  colorableStderr.Write([]byte(fmt.Sprintf("%s %s\n", Red("Error:"), err.Error())))
  os.Exit(1)
}

func PrintInfo(format string, a ...interface{}) {
  args := append([]interface{}{Cyan("Info: ")}, a...)
  colorableStdout.Write([]byte(fmt.Sprintf("%s "+format+"\n", args...)))
}

func PrintWarning(format string, a ...interface{}) {
  args := append([]interface{}{Bold(Yellow("Warn: "))}, a...)
  colorableStdout.Write([]byte(fmt.Sprintf("%s "+format+"\n", args...)))
}

func PrintHelp(cmd string, cmdline string, message []interface{}, opts OptionsPrinter) {
  cmdlineStr := "[-help]"
  if opts != nil {
    cmdlineStr += " [options]"
  }
  if cmdline != "" {
    cmdlineStr += " " + cmdline
  }

  colorableStdout.Write([]byte(fmt.Sprintf("Usage: %s %s %s\n", os.Args[0], Bold(cmd), cmdlineStr)))
  if len(message) > 0 {
    colorableStdout.Write([]byte("\n"))
    for _, line := range message {
      colorableStdout.Write([]byte(fmt.Sprintf("%s\n", line)))
    }
  }

  if opts != nil {
    colorableStdout.Write([]byte("\n"))
    colorableStdout.Write([]byte("Options:\n"))
    colorableStdout.Write([]byte("\n"))
    opts.SetOutput(colorableStdout)
    opts.PrintDefaults()
  }
}

func PrintMessage(message []interface{}) {
  for _, line := range message {
    colorableStdout.Write([]byte(fmt.Sprintf("%s\n", line)))
  }
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
