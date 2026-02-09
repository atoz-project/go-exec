package goexec

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/rs/zerolog"
)

func TestCleaner_Clean_LIFOOrder(t *testing.T) {
	var calls []string
	c := &Cleaner{}
	c.AddCleaners(
		func(context.Context) error { calls = append(calls, "a"); return nil },
		func(context.Context) error { calls = append(calls, "b"); return nil },
		func(context.Context) error { calls = append(calls, "c"); return nil },
	)

	ctx := zerolog.New(io.Discard).WithContext(context.Background())
	if err := c.Clean(ctx); err != nil {
		t.Fatalf("Clean returned error: %v", err)
	}

	want := []string{"c", "b", "a"}
	if len(calls) != len(want) {
		t.Fatalf("calls=%v, want %v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("calls=%v, want %v", calls, want)
		}
	}
}

func TestCleaner_Clean_JoinsErrors(t *testing.T) {
	e1 := errors.New("e1")
	e2 := errors.New("e2")

	c := &Cleaner{}
	c.AddCleaners(
		func(context.Context) error { return e1 },
		func(context.Context) error { return nil },
		func(context.Context) error { return e2 },
	)

	ctx := zerolog.New(io.Discard).WithContext(context.Background())
	err := c.Clean(ctx)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, e1) || !errors.Is(err, e2) {
		t.Fatalf("err=%v, want joined errors containing %v and %v", err, e1, e2)
	}
}
