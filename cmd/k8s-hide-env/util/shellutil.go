package util

import "strings"
import "errors"

const shellCommandPrefix = ". /k8s-hide-env/container.env &&"

func ReconcileShellCommand(commands []string, args []string) (string, error) {

	if len(commands) == 0 && len(args) == 0 {
		return "", errors.New("either command or args have to be defined in K8s manifest")
	}

	allCommands := []string{shellCommandPrefix}
	allCommands = append(allCommands, removeShellCommand(commands)...)
	allCommands = append(allCommands, removeShellCommand(args)...)
	return strings.Join(allCommands, " "), nil
}

func removeShellCommand(oldCommands []string) []string {

	if len(oldCommands) > 1 && oldCommands[0] == "sh" && oldCommands[1] == "-c" {
		return oldCommands[2:]
	}
	return oldCommands
}
