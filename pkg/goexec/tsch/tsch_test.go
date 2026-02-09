package tschexec

import (
	"testing"
	"time"
)

func TestXmlDuration(t *testing.T) {
	if got := xmlDuration(0); got != "PT0S" {
		t.Fatalf("xmlDuration(0)=%q, want %q", got, "PT0S")
	}
	if got := xmlDuration(5 * time.Second); got != "PT5S" {
		t.Fatalf("xmlDuration(5s)=%q, want %q", got, "PT5S")
	}
	if got := xmlDuration(1500 * time.Millisecond); got != "PT1S" {
		t.Fatalf("xmlDuration(1.5s)=%q, want %q", got, "PT1S")
	}
	if got := xmlDuration(-1 * time.Second); got != "PT0S" {
		t.Fatalf("xmlDuration(-1s)=%q, want %q", got, "PT0S")
	}
}

func TestValidateTaskName(t *testing.T) {
	if err := ValidateTaskName("Task_01"); err != nil {
		t.Fatalf("expected valid task name: %v", err)
	}
	if err := ValidateTaskName("Task Name"); err != nil {
		// The current regex allows spaces after the first character; keep this as a
		// "should be valid" case unless the validation changes.
		t.Fatalf("expected valid task name with space: %v", err)
	}
	if err := ValidateTaskName("Task/Name"); err == nil {
		t.Fatalf("expected invalid task name with slash")
	}
	if err := ValidateTaskName(`Task\\Name`); err == nil {
		t.Fatalf("expected invalid task name with backslash")
	}
}

func TestValidateTaskPath(t *testing.T) {
	// Valid examples for the current regex.
	if err := ValidateTaskPath(`\Task01`); err != nil {
		t.Fatalf("expected valid task path: %v", err)
	}
	if err := ValidateTaskPath(`\Microsoft\Windows\UPnP\UPnPHostConfig`); err != nil {
		t.Fatalf("expected valid task path: %v", err)
	}

	// Invalid examples.
	if err := ValidateTaskPath("Task01"); err == nil {
		t.Fatalf("expected invalid task path without leading backslash")
	}
	if err := ValidateTaskPath(`\\Task01`); err == nil {
		t.Fatalf("expected invalid task path with double leading backslash")
	}
	if err := ValidateTaskPath(`\ Task01`); err == nil {
		t.Fatalf("expected invalid task path with leading space")
	}
	if err := ValidateTaskPath(`\Task:01`); err == nil {
		t.Fatalf("expected invalid task path containing ':'")
	}
	if err := ValidateTaskPath(`\Task/01`); err == nil {
		t.Fatalf("expected invalid task path containing '/'")
	}
}
