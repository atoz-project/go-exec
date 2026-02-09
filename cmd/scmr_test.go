package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestScmrChangeCommand_RequiresExecutablePath(t *testing.T) {
	f := scmrChangeCmd.Flags().Lookup("executable-path")
	if f == nil {
		t.Fatalf("expected scmr change to have flag executable-path")
	}
	ann, ok := f.Annotations[cobra.BashCompOneRequiredFlag]
	if !ok || len(ann) == 0 || ann[0] != "true" {
		t.Fatalf("expected executable-path to be required; annotations=%v", f.Annotations)
	}
}

func TestScmrChangeCommand_RequiresServiceName(t *testing.T) {
	f := scmrChangeCmd.Flags().Lookup("service-name")
	if f == nil {
		t.Fatalf("expected scmr change to have flag service-name")
	}
	ann, ok := f.Annotations[cobra.BashCompOneRequiredFlag]
	if !ok || len(ann) == 0 || ann[0] != "true" {
		t.Fatalf("expected service-name to be required; annotations=%v", f.Annotations)
	}
}
