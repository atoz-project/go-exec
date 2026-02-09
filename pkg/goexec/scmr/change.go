package scmrexec

import (
	"context"
	"errors"
	"fmt"
	"github.com/atoz-project/go-exec/pkg/goexec"
	"github.com/oiweiwei/go-msrpc/dcerpc"
	"github.com/oiweiwei/go-msrpc/msrpc/erref/win32"
	"github.com/oiweiwei/go-msrpc/msrpc/scmr/svcctl/v2"
	"github.com/rs/zerolog"
)

const (
	MethodChange = "Change"
)

type ScmrChange struct {
	Scmr
	goexec.Cleaner
	goexec.Executor

	IO goexec.ExecutionIO

	NoStart     bool
	NoRevert    bool
	ServiceName string
}

func (m *ScmrChange) Clean(ctx context.Context) error {
	// Clean method-level resources first, then base SCMR connection resources.
	return errors.Join(m.Cleaner.Clean(ctx), m.Scmr.Clean(ctx))
}

type serviceStateController interface {
	QueryServiceStatus(context.Context, *svcctl.QueryServiceStatusRequest, ...dcerpc.CallOption) (*svcctl.QueryServiceStatusResponse, error)
	StartServiceW(context.Context, *svcctl.StartServiceWRequest, ...dcerpc.CallOption) (*svcctl.StartServiceWResponse, error)
	ControlService(context.Context, *svcctl.ControlServiceRequest, ...dcerpc.CallOption) (*svcctl.ControlServiceResponse, error)
}

func restoreServiceState(ctx context.Context, log zerolog.Logger, ctl serviceStateController, handle *svcctl.Handle, originalState uint32) error {
	// Best-effort restoration for common states (stopped, running, paused). Other
	// states (pending transitions) are skipped to avoid making things worse.
	switch originalState {
	case ServiceStopped, ServiceRunning, ServicePaused:
	default:
		return nil
	}
	if handle == nil {
		return errors.New("service handle is nil")
	}

	qr, err := ctl.QueryServiceStatus(ctx, &svcctl.QueryServiceStatusRequest{Service: handle})
	if err != nil {
		return fmt.Errorf("query service status: %w", err)
	}
	if qr.Return != 0 {
		return fmt.Errorf("query service status: %w", win32.FromCode(qr.Return))
	}
	if qr.ServiceStatus == nil {
		return errors.New("query service status: missing ServiceStatus")
	}

	currentState := qr.ServiceStatus.CurrentState
	switch originalState {
	case ServiceRunning:
		if currentState == ServiceRunning {
			return nil
		}
		sr, err := ctl.StartServiceW(ctx, &svcctl.StartServiceWRequest{Service: handle})
		if err != nil {
			// Some targets time out even when the start request was accepted.
			if errors.Is(err, context.DeadlineExceeded) {
				log.Warn().Err(err).Msg("Service start deadline exceeded; assuming start was accepted")
				return nil
			}
			return fmt.Errorf("start service: %w", err)
		}
		if sr.Return != 0 && sr.Return != ErrorServiceRequestTimeout {
			return fmt.Errorf("start service: %w", win32.FromCode(sr.Return))
		}
		return nil

	case ServiceStopped:
		if currentState == ServiceStopped {
			return nil
		}
		cr, err := ctl.ControlService(ctx, &svcctl.ControlServiceRequest{Service: handle, Control: ServiceControlStop})
		if err != nil {
			// Stopping an already-stopped service is fine.
			if cr != nil && cr.Return == ErrorServiceNotActive {
				return nil
			}
			return fmt.Errorf("stop service: %w", err)
		}
		if cr.Return != 0 && cr.Return != ErrorServiceNotActive {
			return fmt.Errorf("stop service: %w", win32.FromCode(cr.Return))
		}
		return nil

	case ServicePaused:
		if currentState == ServicePaused {
			return nil
		}
		if currentState == ServiceStopped {
			sr, err := ctl.StartServiceW(ctx, &svcctl.StartServiceWRequest{Service: handle})
			if err != nil && !errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("start service for pause: %w", err)
			}
			if sr != nil && sr.Return != 0 && sr.Return != ErrorServiceRequestTimeout {
				return fmt.Errorf("start service for pause: %w", win32.FromCode(sr.Return))
			}
		}
		cr, err := ctl.ControlService(ctx, &svcctl.ControlServiceRequest{Service: handle, Control: ServiceControlPause})
		if err != nil {
			return fmt.Errorf("pause service: %w", err)
		}
		if cr.Return != 0 {
			return fmt.Errorf("pause service: %w", win32.FromCode(cr.Return))
		}
		return nil
	}

	return nil
}

func (m *ScmrChange) Execute(ctx context.Context, in *goexec.ExecutionIO) (err error) {

	log := zerolog.Ctx(ctx).With().
		Str("service", m.ServiceName).
		Logger()

	svc := &service{name: m.ServiceName}

	openResponse, err := m.ctl.OpenServiceW(ctx, &svcctl.OpenServiceWRequest{
		ServiceManager: m.scm,
		ServiceName:    svc.name,
		DesiredAccess:  ServiceModifyAccess,
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to open service handle")
		return fmt.Errorf("open service request: %w", err)
	}
	if openResponse.Return != 0 {
		codeErr := win32.FromCode(openResponse.Return)
		log.Error().Err(codeErr).Msg("Failed to open service handle")
		return fmt.Errorf("open service: %w", codeErr)
	}

	svc.handle = openResponse.Service
	log.Info().Msg("Opened service handle")

	// Register early so it runs last (Cleaner is LIFO).
	m.AddCleaners(func(ctxInner context.Context) error {
		return m.closeService(ctxInner, svc)
	})

	statusResponse, err := m.ctl.QueryServiceStatus(ctx, &svcctl.QueryServiceStatusRequest{Service: svc.handle})
	if err != nil {
		log.Debug().Err(err).Msg("Failed to fetch service status")
	} else if statusResponse.Return != 0 {
		log.Debug().Err(win32.FromCode(statusResponse.Return)).Msg("Failed to fetch service status")
	} else {
		svc.originalStatus = statusResponse.ServiceStatus
		if svc.originalStatus != nil {
			log.Debug().Str("state", fmt.Sprintf("0x%08x", svc.originalStatus.CurrentState)).Msg("Fetched original service status")
		}
	}

	// Note the original service configuration
	queryResponse, err := m.ctl.QueryServiceConfigW(ctx, &svcctl.QueryServiceConfigWRequest{
		Service:      svc.handle,
		BufferLength: 8 * 1024,
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch service configuration")
		return fmt.Errorf("get service config: %w", err)
	}

	log.Info().Str("binaryPath", queryResponse.ServiceConfig.BinaryPathName).Msg("Fetched original service configuration")
	svc.originalConfig = queryResponse.ServiceConfig

	originalState := uint32(0)
	if svc.originalStatus != nil {
		originalState = svc.originalStatus.CurrentState
	}
	originalStartType := svc.originalConfig.StartType

	stopResponse, err := m.ctl.ControlService(ctx, &svcctl.ControlServiceRequest{
		Service: svc.handle,
		Control: ServiceControlStop,
	})

	if err != nil {
		if stopResponse == nil || stopResponse.Return != ErrorServiceNotActive {

			log.Error().Err(err).Msg("Failed to stop existing service")
			return fmt.Errorf("stop service: %w", err)
		}

		log.Debug().Msg("Service is not running")

	} else {
		log.Info().Msg("Stopped existing service")
	}

	req := &svcctl.ChangeServiceConfigWRequest{
		Service:          svc.handle,
		BinaryPathName:   in.String(),
		DisplayName:      svc.originalConfig.DisplayName,
		ServiceType:      svc.originalConfig.ServiceType,
		StartType:        ServiceDemandStart,
		ErrorControl:     svc.originalConfig.ErrorControl,
		LoadOrderGroup:   svc.originalConfig.LoadOrderGroup,
		ServiceStartName: svc.originalConfig.ServiceStartName,
		TagID:            svc.originalConfig.TagID,
		Dependencies:     parseDependencies(svc.originalConfig.Dependencies),
	}

	bpn := svc.originalConfig.BinaryPathName

	_, err = m.ctl.ChangeServiceConfigW(ctx, req)

	if err != nil {
		log.Error().Err(err).Msg("Failed to request service configuration change")
		return fmt.Errorf("change service config request: %w", err)
	}

	if !m.NoStart {
		err = m.startService(ctx, svc)
		if err != nil {
			log.Error().Err(err).Msg("Failed to start service")
		}
	}

	if !m.NoRevert {
		if svc.handle == nil {

			if err = m.Reconnect(ctx); err != nil {
				return err
			}
			svc, err = m.openService(ctx, svc.name, ServiceModifyAccess)

			if err != nil {
				log.Error().Err(err).Msg("Failed to reopen service handle")
				return fmt.Errorf("reopen service: %w", err)
			}
		}

		// Restore original configuration. When the service was running and its
		// original start type was disabled, restore the binary path first using
		// a startable type, then restore the start type after state restoration.
		restoreStartTypeLater := false
		req.BinaryPathName = bpn
		req.Service = svc.handle
		req.StartType = originalStartType
		if (originalState == ServiceRunning || originalState == ServicePaused) && originalStartType == ServiceDisabled {
			restoreStartTypeLater = true
			req.StartType = ServiceDemandStart
		}

		_, err := m.ctl.ChangeServiceConfigW(ctx, req)

		if err != nil {
			log.Error().Err(err).Msg("Failed to restore original service configuration")
			return fmt.Errorf("restore service config: %w", err)
		}
		log.Info().Msg("Restored original service configuration")

		if originalState == ServiceRunning || originalState == ServiceStopped || originalState == ServicePaused {
			if restoreErr := restoreServiceState(ctx, log, m.ctl, svc.handle, originalState); restoreErr != nil {
				log.Warn().Err(restoreErr).Msg("Failed to restore original service state")
			} else {
				log.Info().Msg("Restored original service state")
			}
		}

		if restoreStartTypeLater {
			req.StartType = originalStartType
			if _, err := m.ctl.ChangeServiceConfigW(ctx, req); err != nil {
				log.Warn().Err(err).Msg("Failed to restore original service start type")
			} else {
				log.Info().Msg("Restored original service start type")
			}
		}
	}

	return
}
