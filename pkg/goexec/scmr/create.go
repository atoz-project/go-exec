package scmrexec

import (
	"context"
	"errors"
	"fmt"
	"github.com/atoz-project/go-exec/internal/util"
	"github.com/atoz-project/go-exec/pkg/goexec"
	"github.com/oiweiwei/go-msrpc/msrpc/erref/win32"
	"github.com/oiweiwei/go-msrpc/msrpc/scmr/svcctl/v2"
	"github.com/rs/zerolog"
)

const (
	MethodCreate = "Create"
)

type ScmrCreate struct {
	Scmr
	goexec.Cleaner
	goexec.Executor

	IO goexec.ExecutionIO

	NoDelete    bool
	NoStart     bool
	ServiceName string
	DisplayName string
}

func (m *ScmrCreate) Clean(ctx context.Context) error {
	// Clean method-level resources first, then base SCMR connection resources.
	return errors.Join(m.Cleaner.Clean(ctx), m.Scmr.Clean(ctx))
}

func (m *ScmrCreate) ensure() {
	if m.ServiceName == "" {
		m.ServiceName = util.RandomString()
	}
	if m.DisplayName == "" {
		m.DisplayName = m.ServiceName
	}
}

func (m *ScmrCreate) Execute(ctx context.Context, in *goexec.ExecutionIO) (err error) {
	m.ensure()

	log := zerolog.Ctx(ctx).With().
		Str("service", m.ServiceName).Logger()

	svc := &service{name: m.ServiceName}

	resp, err := m.ctl.CreateServiceW(ctx, &svcctl.CreateServiceWRequest{
		ServiceManager: m.scm,
		ServiceName:    m.ServiceName,
		DisplayName:    m.DisplayName,
		BinaryPathName: in.String(),
		ServiceType:    ServiceWin32OwnProcess,
		StartType:      ServiceDemandStart,
		DesiredAccess:  ServiceAllAccess, // TODO: Replace
	})

	if err != nil {
		log.Error().Err(err).Msg("Create service request failed")
		return fmt.Errorf("create service request: %w", err)
	}

	if resp.Return != 0 {
		codeErr := win32.FromCode(resp.Return)
		log.Error().Err(codeErr).Msg("Failed to create service")
		return fmt.Errorf("create service: %w", codeErr)
	}

	svc.handle = resp.Service

	// Register close first so it runs last (Cleaner is LIFO).
	m.AddCleaners(func(ctxInner context.Context) error {

		r, errInner := m.ctl.CloseService(ctxInner, &svcctl.CloseServiceRequest{
			ServiceObject: svc.handle,
		})
		if errInner != nil {
			return fmt.Errorf("close service: %w", errInner)
		}
		if r.Return != 0 {
			return fmt.Errorf("close service: %w", win32.FromCode(r.Return))
		}
		log.Info().Msg("Closed service handle")

		return nil
	})

	if !m.NoDelete {
		// Register delete after close so it runs before close (Cleaner is LIFO).
		m.AddCleaners(func(ctxInner context.Context) error {

			r, errInner := m.ctl.DeleteService(ctxInner, &svcctl.DeleteServiceRequest{
				Service: svc.handle,
			})
			if errInner != nil {
				return fmt.Errorf("delete service: %w", errInner)
			}
			if r.Return != 0 {
				return fmt.Errorf("delete service: %w", win32.FromCode(r.Return))
			}
			log.Info().Msg("Deleted service")

			return nil
		})
	}

	log.Info().Msg("Created service")

	if !m.NoStart {

		err = m.startService(ctx, svc)

		if err != nil {
			log.Error().Err(err).Msg("Failed to start service")
			return fmt.Errorf("start service: %w", err)
		}
	}
	if svc.handle == nil {

		if err = m.Reconnect(ctx); err != nil {
			return err
		}
		svc, err = m.openService(ctx, svc.name)

		if err != nil {
			log.Error().Err(err).Msg("Failed to reopen service handle")
			return fmt.Errorf("reopen service: %w", err)
		}
	}

	return
}
