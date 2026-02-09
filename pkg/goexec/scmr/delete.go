package scmrexec

import (
	"context"
	"errors"
	"github.com/atoz-project/go-exec/pkg/goexec"
)

const (
	MethodDelete = "Delete"
)

type ScmrDelete struct {
	Scmr
	goexec.Cleaner

	IO goexec.ExecutionIO

	ServiceName string
}

func (m *ScmrDelete) Clean(ctx context.Context) error {
	// Clean method-level resources first, then base SCMR connection resources.
	return errors.Join(m.Cleaner.Clean(ctx), m.Scmr.Clean(ctx))
}

func (m *ScmrDelete) Call(ctx context.Context) (err error) {

	svc, err := m.openService(ctx, m.ServiceName, ServiceDeleteAccess)
	if err != nil {
		return err
	}
	// Register early so it runs last (Cleaner is LIFO).
	m.AddCleaners(func(ctxInner context.Context) error { return m.closeService(ctxInner, svc) })

	return m.deleteService(ctx, svc)
}
