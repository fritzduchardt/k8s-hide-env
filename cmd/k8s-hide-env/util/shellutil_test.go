package util

import (
	"testing"
)

func TestReconcileShellCommand(t *testing.T) {

	tests := []struct {
		testCase    string
		commands    []string
		args        []string
		expected    string
		expectError bool
	}{
		{
			"no command and args",
			[]string{},
			[]string{},
			"",
			true,
		},
		{
			"only command",
			[]string{"tail", "-f"},
			[]string{},
			shellCommandPrefix + " tail -f",
			false,
		},
		{
			"only args",
			[]string{},
			[]string{"tail", "-f"},
			shellCommandPrefix + " tail -f",
			false,
		},
		{
			"command and args",
			[]string{"tail", "-f"},
			[]string{"/dev/null"},
			shellCommandPrefix + " tail -f /dev/null",
			false,
		},
		{
			"shell in command",
			[]string{"sh", "-c", "tail -f /dev/null"},
			[]string{},
			shellCommandPrefix + " tail -f /dev/null",
			false,
		},
		{
			"shell in args",
			[]string{},
			[]string{"sh", "-c", "tail -f /dev/null"},
			shellCommandPrefix + " tail -f /dev/null",
			false,
		},
		{
			"shell in args and command",
			[]string{"sh", "-c", "tail -f /dev/null"},
			[]string{"sh", "-c", "tail -f /dev/null"},
			shellCommandPrefix + " tail -f /dev/null tail -f /dev/null",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testCase, func(t *testing.T) {
			actual, err := ReconcileShellCommand(tt.commands, tt.args)
			if tt.expectError && err == nil {
				t.Error("Expected error but did not throw any.")
			} else if actual != tt.expected {
				t.Errorf("Expected: %s, got result: %s", tt.expected, actual)
			}
		})
	}
}
