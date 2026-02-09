package goexec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

type nopWriteCloser struct {
	io.Writer
}

func (n nopWriteCloser) Close() error { return nil }

type recordingMethod struct {
	calls *[]string

	connectErr error
	initErr    error
	execErr    error
	cleanErr   error
}

func (m *recordingMethod) Connect(context.Context) error {
	*m.calls = append(*m.calls, "connect")
	return m.connectErr
}

func (m *recordingMethod) Init(context.Context) error {
	*m.calls = append(*m.calls, "init")
	return m.initErr
}

func (m *recordingMethod) Execute(context.Context, *ExecutionIO) error {
	*m.calls = append(*m.calls, "execute")
	return m.execErr
}

func (m *recordingMethod) Clean(context.Context) error {
	*m.calls = append(*m.calls, "moduleClean")
	return m.cleanErr
}

type recordingProvider struct {
	calls *[]string

	getErr   error
	cleanErr error
	wantTO   time.Duration
}

func (p *recordingProvider) GetOutput(ctx context.Context, w io.Writer) error {
	*p.calls = append(*p.calls, "getOutput")

	if p.wantTO != 0 {
		v := ctx.Value(ContextOptionOutputTimeout)
		got, ok := v.(time.Duration)
		if !ok || got != p.wantTO {
			return errors.New("missing or unexpected ContextOptionOutputTimeout")
		}
	}

	if p.getErr != nil {
		return p.getErr
	}
	if w != nil {
		_, _ = w.Write([]byte("ok"))
	}
	return nil
}

func (p *recordingProvider) Clean(context.Context) error {
	*p.calls = append(*p.calls, "providerClean")
	return p.cleanErr
}

func TestExecuteCleanMethod_OrderAndOutput(t *testing.T) {
	calls := []string{}

	m := &recordingMethod{calls: &calls, cleanErr: errors.New("ignored")}
	p := &recordingProvider{calls: &calls, wantTO: 123 * time.Second}

	outBuf := new(bytes.Buffer)
	execIO := &ExecutionIO{
		Input: &ExecutionInput{
			Executable: "whoami",
			Arguments:  "/all",
		},
		Output: &ExecutionOutput{
			Provider: p,
			Timeout:  123 * time.Second,
			Writer:   nopWriteCloser{outBuf},
		},
	}

	ctx := zerolog.New(io.Discard).WithContext(context.Background())

	if err := ExecuteCleanMethod(ctx, m, execIO); err != nil {
		t.Fatalf("ExecuteCleanMethod returned error: %v", err)
	}

	wantCalls := []string{"connect", "init", "execute", "moduleClean", "getOutput", "providerClean"}
	if len(calls) != len(wantCalls) {
		t.Fatalf("calls=%v, want %v", calls, wantCalls)
	}
	for i := range wantCalls {
		if calls[i] != wantCalls[i] {
			t.Fatalf("calls=%v, want %v", calls, wantCalls)
		}
	}

	if got := outBuf.String(); got != "ok" {
		t.Fatalf("output=%q, want %q", got, "ok")
	}
}

func TestExecuteCleanMethod_OutputErrorStillCleansProvider(t *testing.T) {
	calls := []string{}

	m := &recordingMethod{calls: &calls}
	p := &recordingProvider{calls: &calls, getErr: errors.New("boom")}

	execIO := &ExecutionIO{
		Input: &ExecutionInput{Executable: "whoami"},
		Output: &ExecutionOutput{
			Provider: p,
			Timeout:  1 * time.Second,
			Writer:   nopWriteCloser{io.Discard},
		},
	}

	ctx := zerolog.New(io.Discard).WithContext(context.Background())

	if err := ExecuteCleanMethod(ctx, m, execIO); err == nil {
		t.Fatalf("expected error")
	}

	// providerClean must still run due to deferred execIO.Clean().
	found := false
	for _, c := range calls {
		if c == "providerClean" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("calls=%v missing providerClean", calls)
	}
}

func TestExecuteMethod_ConnectErrorShortCircuits(t *testing.T) {
	calls := []string{}

	m := &recordingMethod{calls: &calls, connectErr: errors.New("nope")}
	execIO := &ExecutionIO{Input: &ExecutionInput{Executable: "whoami"}}
	ctx := zerolog.New(io.Discard).WithContext(context.Background())

	if err := ExecuteMethod(ctx, m, execIO); err == nil {
		t.Fatalf("expected error")
	}

	if len(calls) != 1 || calls[0] != "connect" {
		t.Fatalf("calls=%v, want [connect]", calls)
	}
}

func TestExecuteCleanMethod_ExecuteErrorStillCleansModule(t *testing.T) {
	calls := []string{}

	m := &recordingMethod{calls: &calls, connectErr: errors.New("nope")}
	execIO := &ExecutionIO{Input: &ExecutionInput{Executable: "whoami"}}
	ctx := zerolog.New(io.Discard).WithContext(context.Background())

	if err := ExecuteCleanMethod(ctx, m, execIO); err == nil {
		t.Fatalf("expected error")
	}

	// moduleClean must still run even when ExecuteMethod returns an error.
	want := []string{"connect", "moduleClean"}
	if len(calls) != len(want) {
		t.Fatalf("calls=%v, want %v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls=%v, want %v", calls, want)
		}
	}
}

type recordingAuxMethod struct {
	calls *[]string

	connectErr error
	initErr    error
	callErr    error
	cleanErr   error
}

func (m *recordingAuxMethod) Connect(context.Context) error {
	*m.calls = append(*m.calls, "connect")
	return m.connectErr
}

func (m *recordingAuxMethod) Init(context.Context) error {
	*m.calls = append(*m.calls, "init")
	return m.initErr
}

func (m *recordingAuxMethod) Call(context.Context) error {
	*m.calls = append(*m.calls, "call")
	return m.callErr
}

func (m *recordingAuxMethod) Clean(context.Context) error {
	*m.calls = append(*m.calls, "moduleClean")
	return m.cleanErr
}

func TestExecuteCleanAuxiliaryMethod_ReturnsExecuteErrorAndStillCleans(t *testing.T) {
	calls := []string{}

	m := &recordingAuxMethod{
		calls:    &calls,
		callErr:  errors.New("boom"),
		cleanErr: errors.New("ignored"),
	}
	ctx := zerolog.New(io.Discard).WithContext(context.Background())

	if err := ExecuteCleanAuxiliaryMethod(ctx, m); err == nil {
		t.Fatalf("expected error")
	}

	want := []string{"connect", "init", "call", "moduleClean"}
	if len(calls) != len(want) {
		t.Fatalf("calls=%v, want %v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls=%v, want %v", calls, want)
		}
	}
}
