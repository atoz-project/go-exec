package goexec

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

type OutputProvider interface {
	GetOutput(ctx context.Context, writer io.Writer) (err error)
	Clean(ctx context.Context) (err error)
}

type ExecutionIO struct {
	Cleaner

	Input  *ExecutionInput
	Output *ExecutionOutput
}

type ExecutionOutput struct {
	NoDelete   bool
	RemotePath string
	Timeout    time.Duration
	Provider   OutputProvider
	Writer     io.WriteCloser
}

type ExecutionInput struct {
	StageFile      io.ReadCloser
	Executable     string
	ExecutablePath string
	Arguments      string
	Command        string
}

func (execIO *ExecutionIO) GetOutput(ctx context.Context) (err error) {
	if execIO.Output.Provider != nil {
		ctx = context.WithValue(ctx, ContextOptionOutputTimeout, execIO.Output.Timeout)
		return execIO.Output.Provider.GetOutput(ctx, execIO.Output.Writer)
	}
	return nil
}

func (execIO *ExecutionIO) Clean(ctx context.Context) (err error) {
	if execIO.Output.Provider != nil {
		return execIO.Output.Provider.Clean(ctx)
	}
	return nil
}

func (execIO *ExecutionIO) CommandLine() (cmd []string) {
	if execIO.Output.Provider != nil && execIO.Output.RemotePath != "" {
		// Wrap the command with cmd.exe to redirect output to a remote file.
		// We must quote the original executable path when it contains spaces so
		// CreateProcess/cmd.exe parse it correctly.
		inner := execIO.Input.quotedString()
		return []string{
			`C:\Windows\System32\cmd.exe`,
			fmt.Sprintf(`/C %s >%s 2>&1`, inner, execIO.Output.RemotePath),
		}
	}
	return execIO.Input.CommandLine()
}

func (execIO *ExecutionIO) String() (str string) {
	cmd := execIO.CommandLine()
	// Ensure that executable paths are quoted (Windows-style), not Go-string-escaped.
	exe := strings.TrimSpace(cmd[0])
	if strings.Contains(exe, " ") {
		exe = quoteWindowsArg(exe)
	}
	rest := strings.TrimSpace(strings.Join(cmd[1:], " "))
	if rest == "" {
		str = exe
	} else {
		str = exe + " " + rest
	}
	return strings.TrimSpace(str) // trim whitespace
}

func (i *ExecutionInput) CommandLine() (cmd []string) {
	cmd = make([]string, 2)
	cmd[1] = i.Arguments

	switch {
	case i.Command != "":
		copy(cmd, strings.SplitN(i.Command, " ", 2))
	case i.ExecutablePath != "":
		cmd[0] = i.ExecutablePath
	case i.Executable != "":
		cmd[0] = i.Executable
	}

	return cmd
}

func (i *ExecutionInput) String() string {
	return strings.TrimSpace(strings.Join(i.CommandLine(), " "))
}

func (i *ExecutionInput) quotedString() string {
	cmd := i.CommandLine()
	if len(cmd) == 0 {
		return ""
	}

	exe := strings.TrimSpace(cmd[0])
	args := ""
	if len(cmd) > 1 {
		args = strings.TrimSpace(cmd[1])
	}

	// Quote only the executable path; do not attempt to re-quote the full command line.
	if strings.Contains(exe, " ") {
		exe = quoteWindowsArg(exe)
	}

	if args == "" {
		return exe
	}
	return strings.TrimSpace(exe + " " + args)
}

func quoteWindowsArg(s string) string {
	// Minimal quoting for Windows command lines:
	// - wrap in double quotes
	// - escape embedded quotes
	// NOTE: Windows quoting rules are subtle; this is intentionally scoped to
	// quoting executable paths (which should not normally contain quotes).
	if strings.Contains(s, `"`) {
		s = strings.ReplaceAll(s, `"`, `\"`)
	}
	return `"` + s + `"`
}

func (i *ExecutionInput) Reader() (reader io.Reader) {
	if i.StageFile != nil {
		return i.StageFile
	}
	return strings.NewReader(i.String())
}
