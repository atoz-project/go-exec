package goexec

import (
	"context"
	"errors"
	"github.com/rs/zerolog"
)

type Clean interface {
	Clean(ctx context.Context) error
}

type Cleaner struct {
	workers []func(ctx context.Context) error
}

func (c *Cleaner) AddCleaners(workers ...func(ctx context.Context) error) {
	c.workers = append(c.workers, workers...)
}

func (c *Cleaner) Clean(ctx context.Context) (err error) {
	log := zerolog.Ctx(ctx).With().
		Str("component", "cleaner").Logger()

	var errs []error
	// Execute cleaners in LIFO order to match defer semantics.
	for i := len(c.workers) - 1; i >= 0; i-- {
		if cleanErr := c.workers[i](log.WithContext(ctx)); cleanErr != nil {
			log.Warn().Err(cleanErr).Msg("Clean worker failed")
			errs = append(errs, cleanErr)
		}
	}
	return errors.Join(errs...)
}
