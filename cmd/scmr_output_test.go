package cmd

import "testing"

// TestScmrCreateHasOutputFlags verifies that the scmr create command registers
// execution output flags (--out, --out-method, --out-timeout, --no-delete-out).
// These flags are currently commented out in scmrCreateCmdInit.
func TestScmrCreateHasOutputFlags(t *testing.T) {
	requiredFlags := []string{"out", "out-method", "out-timeout", "no-delete-out"}

	for _, name := range requiredFlags {
		if f := scmrCreateCmd.Flags().Lookup(name); f == nil {
			t.Errorf("scmr create missing flag --%s", name)
		}
	}
}

// TestScmrChangeHasOutputFlags verifies that the scmr change command registers
// execution output flags (--out, --out-method, --out-timeout, --no-delete-out).
// These flags are currently commented out in scmrChangeCmdInit.
func TestScmrChangeHasOutputFlags(t *testing.T) {
	requiredFlags := []string{"out", "out-method", "out-timeout", "no-delete-out"}

	for _, name := range requiredFlags {
		if f := scmrChangeCmd.Flags().Lookup(name); f == nil {
			t.Errorf("scmr change missing flag --%s", name)
		}
	}
}
