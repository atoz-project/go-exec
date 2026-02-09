package smb

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/atoz-project/go-exec/pkg/goexec"
	"github.com/oiweiwei/go-smb2.fork"
	"github.com/rs/zerolog"
)

type trackedRWC struct {
	r     *bytes.Reader
	order *[]string
}

func (f *trackedRWC) Read(p []byte) (int, error)  { return f.r.Read(p) }
func (f *trackedRWC) Write(p []byte) (int, error) { return len(p), nil }
func (f *trackedRWC) Close() error {
	*f.order = append(*f.order, "close")
	return nil
}

func TestOutputFileFetcher_DeletesAfterFetch(t *testing.T) {
	// Stub SMB share operations.
	prevOpen := shareOpenFile
	prevRemove := shareRemoveFile
	t.Cleanup(func() {
		shareOpenFile = prevOpen
		shareRemoveFile = prevRemove
	})

	order := []string{}
	wantRel := filepath.ToSlash(filepath.Join("temp", "out.txt"))

	shareOpenFile = func(_ *smb2.Share, name string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
		if filepath.ToSlash(name) != wantRel {
			t.Fatalf("OpenFile name=%q, want %q", filepath.ToSlash(name), wantRel)
		}
		if flag != os.O_RDWR {
			t.Fatalf("OpenFile flag=%d, want %d", flag, os.O_RDWR)
		}
		order = append(order, "open")
		return &trackedRWC{r: bytes.NewReader([]byte("hello")), order: &order}, nil
	}
	shareRemoveFile = func(_ *smb2.Share, name string) error {
		if filepath.ToSlash(name) != wantRel {
			t.Fatalf("Remove name=%q, want %q", filepath.ToSlash(name), wantRel)
		}
		order = append(order, "remove")
		return nil
	}

	// Prepare fetcher without network calls.
	c := &Client{connected: true, share: "ADMIN$"}
	o := &OutputFileFetcher{
		Client:           c,
		Share:            "ADMIN$",
		SharePath:        `C:\Windows`,
		File:             `C:\Windows\Temp\out.txt`,
		DeleteOutputFile: true,
	}

	ctx := zerolog.New(io.Discard).WithContext(context.Background())
	ctx = context.WithValue(ctx, goexec.ContextOptionOutputTimeout, 50*time.Millisecond)
	ctx = context.WithValue(ctx, goexec.ContextOptionOutputPollInterval, 1*time.Millisecond)

	buf := new(bytes.Buffer)
	if err := o.GetOutput(ctx, buf); err != nil {
		t.Fatalf("GetOutput returned error: %v", err)
	}
	if got := buf.String(); got != "hello" {
		t.Fatalf("output=%q, want %q", got, "hello")
	}

	if err := o.Clean(ctx); err != nil {
		t.Fatalf("Clean returned error: %v", err)
	}

	wantOrder := []string{"open", "close", "remove"}
	if len(order) != len(wantOrder) {
		t.Fatalf("order=%v, want %v", order, wantOrder)
	}
	for i := range wantOrder {
		if order[i] != wantOrder[i] {
			t.Fatalf("order=%v, want %v", order, wantOrder)
		}
	}
}
