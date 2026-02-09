package scmrexec

import (
	"context"
	"io"
	"testing"

	"github.com/oiweiwei/go-msrpc/dcerpc"
	"github.com/oiweiwei/go-msrpc/msrpc/scmr/svcctl/v2"
	"github.com/rs/zerolog"
)

type fakeServiceStateCtl struct {
	current uint32

	startCalled bool
	stopCalled  bool
	pauseCalled bool

	lastControl uint32
}

func (f *fakeServiceStateCtl) QueryServiceStatus(_ context.Context, _ *svcctl.QueryServiceStatusRequest, _ ...dcerpc.CallOption) (*svcctl.QueryServiceStatusResponse, error) {
	return &svcctl.QueryServiceStatusResponse{
		ServiceStatus: &svcctl.ServiceStatus{CurrentState: f.current},
		Return:        0,
	}, nil
}

func (f *fakeServiceStateCtl) StartServiceW(_ context.Context, _ *svcctl.StartServiceWRequest, _ ...dcerpc.CallOption) (*svcctl.StartServiceWResponse, error) {
	f.startCalled = true
	f.current = ServiceRunning
	return &svcctl.StartServiceWResponse{Return: 0}, nil
}

func (f *fakeServiceStateCtl) ControlService(_ context.Context, r *svcctl.ControlServiceRequest, _ ...dcerpc.CallOption) (*svcctl.ControlServiceResponse, error) {
	f.lastControl = r.Control
	switch r.Control {
	case ServiceControlStop:
		f.stopCalled = true
		f.current = ServiceStopped
	case ServiceControlPause:
		f.pauseCalled = true
		f.current = ServicePaused
	}
	return &svcctl.ControlServiceResponse{
		ServiceStatus: &svcctl.ServiceStatus{CurrentState: f.current},
		Return:        0,
	}, nil
}

func TestRestoreServiceState_RunningStartsWhenStopped(t *testing.T) {
	ctl := &fakeServiceStateCtl{current: ServiceStopped}
	log := zerolog.New(io.Discard)

	if err := restoreServiceState(context.Background(), log, ctl, &svcctl.Handle{}, ServiceRunning); err != nil {
		t.Fatalf("restoreServiceState returned error: %v", err)
	}
	if !ctl.startCalled {
		t.Fatalf("expected StartServiceW to be called")
	}
	if ctl.stopCalled {
		t.Fatalf("did not expect stop to be called")
	}
}

func TestRestoreServiceState_StoppedStopsWhenRunning(t *testing.T) {
	ctl := &fakeServiceStateCtl{current: ServiceRunning}
	log := zerolog.New(io.Discard)

	if err := restoreServiceState(context.Background(), log, ctl, &svcctl.Handle{}, ServiceStopped); err != nil {
		t.Fatalf("restoreServiceState returned error: %v", err)
	}
	if !ctl.stopCalled {
		t.Fatalf("expected ControlService(stop) to be called")
	}
	if ctl.lastControl != ServiceControlStop {
		t.Fatalf("control=%d, want %d", ctl.lastControl, ServiceControlStop)
	}
}

func TestRestoreServiceState_PausedStartsThenPausesWhenStopped(t *testing.T) {
	ctl := &fakeServiceStateCtl{current: ServiceStopped}
	log := zerolog.New(io.Discard)

	if err := restoreServiceState(context.Background(), log, ctl, &svcctl.Handle{}, ServicePaused); err != nil {
		t.Fatalf("restoreServiceState returned error: %v", err)
	}
	if !ctl.startCalled {
		t.Fatalf("expected StartServiceW to be called")
	}
	if !ctl.pauseCalled {
		t.Fatalf("expected ControlService(pause) to be called")
	}
	if ctl.lastControl != ServiceControlPause {
		t.Fatalf("control=%d, want %d", ctl.lastControl, ServiceControlPause)
	}
}

func TestRestoreServiceState_UnsupportedStateIsNoop(t *testing.T) {
	ctl := &fakeServiceStateCtl{current: ServiceRunning}
	log := zerolog.New(io.Discard)

	if err := restoreServiceState(context.Background(), log, ctl, &svcctl.Handle{}, ServiceStartPending); err != nil {
		t.Fatalf("restoreServiceState returned error: %v", err)
	}
	if ctl.startCalled || ctl.stopCalled || ctl.pauseCalled {
		t.Fatalf("expected no state change calls; start=%v stop=%v pause=%v", ctl.startCalled, ctl.stopCalled, ctl.pauseCalled)
	}
}
