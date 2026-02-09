package smb

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/oiweiwei/go-smb2.fork"
	"github.com/rs/zerolog"
)

type recordingDialer struct {
	dialCalled        bool
	dialContextCalled bool

	network string
	address string

	hasDeadline bool

	retErr error
}

func (d *recordingDialer) Dial(network string, address string) (net.Conn, error) {
	d.dialCalled = true
	d.network = network
	d.address = address
	return nil, d.retErr
}

func (d *recordingDialer) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	d.dialContextCalled = true
	d.network = network
	d.address = address
	_, d.hasDeadline = ctx.Deadline()
	return nil, d.retErr
}

func TestClient_Connect_DoesNotPanicWhenNotParsed(t *testing.T) {
	c := &Client{}
	ctx := zerolog.New(io.Discard).WithContext(context.Background())

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Connect panicked: %v", r)
		}
	}()

	if err := c.Connect(ctx); err == nil {
		t.Fatalf("expected error")
	}
}

func TestClient_Connect_UsesPortAndTimeoutWhenAvailable(t *testing.T) {
	d := &recordingDialer{retErr: errors.New("dial failed")}
	c := &Client{
		ClientOptions: ClientOptions{
			ConnectTimeout: 50 * time.Millisecond,
		},
	}
	c.Host = "127.0.0.1"
	c.Port = 1445
	c.netDialer = d
	c.dialer = &smb2.Dialer{} // must be non-nil; should not be used in this test

	ctx := zerolog.New(io.Discard).WithContext(context.Background())
	if err := c.Connect(ctx); err == nil {
		t.Fatalf("expected error")
	}

	if !d.dialContextCalled || d.dialCalled {
		t.Fatalf("expected DialContext to be used when available; dialCalled=%v dialContextCalled=%v", d.dialCalled, d.dialContextCalled)
	}
	if d.network != "tcp" {
		t.Fatalf("network=%q, want %q", d.network, "tcp")
	}
	if d.address != "127.0.0.1:1445" {
		t.Fatalf("address=%q, want %q", d.address, "127.0.0.1:1445")
	}
	if !d.hasDeadline {
		t.Fatalf("expected dial context to have a deadline when ConnectTimeout is set")
	}
}

type dummyAddr string

func (d dummyAddr) Network() string { return "tcp" }
func (d dummyAddr) String() string  { return string(d) }

type fakeConn struct {
	closeErr error
	order    *[]string
}

func (c *fakeConn) Read([]byte) (int, error)  { return 0, io.EOF }
func (c *fakeConn) Write([]byte) (int, error) { return 0, nil }
func (c *fakeConn) Close() error {
	if c.order != nil {
		*c.order = append(*c.order, "conn")
	}
	return c.closeErr
}
func (c *fakeConn) LocalAddr() net.Addr              { return dummyAddr("l") }
func (c *fakeConn) RemoteAddr() net.Addr             { return dummyAddr("r") }
func (c *fakeConn) SetDeadline(time.Time) error      { return nil }
func (c *fakeConn) SetReadDeadline(time.Time) error  { return nil }
func (c *fakeConn) SetWriteDeadline(time.Time) error { return nil }

func TestClient_Close_JoinsErrorsAndKeepsOrder(t *testing.T) {
	prevUmount := umountShare
	prevLogoff := logoffSession
	t.Cleanup(func() {
		umountShare = prevUmount
		logoffSession = prevLogoff
	})

	order := []string{}
	eUmount := errors.New("umount")
	eLogoff := errors.New("logoff")
	eConn := errors.New("conn close")

	umountShare = func(*smb2.Share) error {
		order = append(order, "umount")
		return eUmount
	}
	logoffSession = func(*smb2.Session) error {
		order = append(order, "logoff")
		return eLogoff
	}

	c := &Client{
		mount: &smb2.Share{},
		sess:  &smb2.Session{},
		conn:  &fakeConn{closeErr: eConn, order: &order},
		share: "ADMIN$",
	}

	ctx := zerolog.New(io.Discard).WithContext(context.Background())
	err := c.Close(ctx)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, eUmount) || !errors.Is(err, eLogoff) || !errors.Is(err, eConn) {
		t.Fatalf("err=%v, want joined errors containing %v, %v, %v", err, eUmount, eLogoff, eConn)
	}

	wantOrder := []string{"umount", "logoff", "conn"}
	if len(order) != len(wantOrder) {
		t.Fatalf("order=%v, want %v", order, wantOrder)
	}
	for i := range wantOrder {
		if order[i] != wantOrder[i] {
			t.Fatalf("order=%v, want %v", order, wantOrder)
		}
	}

	if c.mount != nil || c.sess != nil || c.conn != nil || c.share != "" {
		t.Fatalf("client not fully reset after Close: mount=%v sess=%v conn=%v share=%q", c.mount, c.sess, c.conn, c.share)
	}
}
