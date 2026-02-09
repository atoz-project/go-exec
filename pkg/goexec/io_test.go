package goexec

import (
	"context"
	"io"
	"strings"
	"testing"
)

type noopOutputProvider struct{}

func (noopOutputProvider) GetOutput(context.Context, io.Writer) error { return nil }
func (noopOutputProvider) Clean(context.Context) error                { return nil }

func TestExecutionInput_CommandLine_Command(t *testing.T) {
	in := &ExecutionInput{
		Command:   "cmd.exe /c whoami",
		Arguments: "ignored",
	}
	got := in.CommandLine()
	if len(got) != 2 {
		t.Fatalf("CommandLine len=%d, want 2", len(got))
	}
	if got[0] != "cmd.exe" || got[1] != "/c whoami" {
		t.Fatalf("CommandLine got=%q, want [cmd.exe /c whoami]", got)
	}
}

func TestExecutionInput_QuotedString_ExecutablePathWithSpaces(t *testing.T) {
	in := &ExecutionInput{
		ExecutablePath: `C:\Program Files\A B\app.exe`,
		Arguments:      "arg1",
	}
	got := in.quotedString()
	want := `"C:\Program Files\A B\app.exe" arg1`
	if got != want {
		t.Fatalf("quotedString got=%q, want %q", got, want)
	}
}

func TestExecutionIO_CommandLine_OutputRedirect_WrapsAndQuotes(t *testing.T) {
	execIO := &ExecutionIO{
		Input: &ExecutionInput{
			ExecutablePath: `C:\Program Files\A B\app.exe`,
			Arguments:      "arg1",
		},
		Output: &ExecutionOutput{
			RemotePath: `C:\Windows\Temp\out.txt`,
			Provider:   noopOutputProvider{},
		},
	}

	cmd := execIO.CommandLine()
	if len(cmd) != 2 {
		t.Fatalf("CommandLine len=%d, want 2", len(cmd))
	}
	if cmd[0] != `C:\Windows\System32\cmd.exe` {
		t.Fatalf("CommandLine[0]=%q, want cmd.exe path", cmd[0])
	}
	if !strings.HasPrefix(cmd[1], "/C ") {
		t.Fatalf("CommandLine[1]=%q missing space after /C", cmd[1])
	}
	if !strings.Contains(cmd[1], `"C:\Program Files\A B\app.exe" arg1`) {
		t.Fatalf("CommandLine[1]=%q missing quoted inner command", cmd[1])
	}
	if !strings.Contains(cmd[1], `>C:\Windows\Temp\out.txt 2>&1`) {
		t.Fatalf("CommandLine[1]=%q missing output redirection", cmd[1])
	}
}

func TestExecutionIO_String_QuotesExecutablePath(t *testing.T) {
	execIO := &ExecutionIO{
		Input: &ExecutionInput{
			ExecutablePath: `C:\Program Files\A B\app.exe`,
			Arguments:      "arg1",
		},
		Output: &ExecutionOutput{},
	}
	got := execIO.String()
	want := `"C:\Program Files\A B\app.exe" arg1`
	if got != want {
		t.Fatalf("ExecutionIO.String got=%q, want %q", got, want)
	}
}
