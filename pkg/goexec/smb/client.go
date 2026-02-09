package smb

import (
	"context"
	"errors"
	"fmt"
	"github.com/oiweiwei/go-smb2.fork"
	"github.com/rs/zerolog"
	"net"
	"strconv"
)

type Client struct {
	ClientOptions

	conn  net.Conn
	sess  *smb2.Session
	mount *smb2.Share

	connected bool
	share     string
}

var (
	umountShare   = func(sh *smb2.Share) error { return sh.Umount() }
	logoffSession = func(s *smb2.Session) error { return s.Logoff() }
)

func (c *Client) Session() (sess *smb2.Session) {
	return c.sess
}

func (c *Client) String() string {
	return ClientName
}

func (c *Client) Logger(ctx context.Context) zerolog.Logger {
	return zerolog.Ctx(ctx).With().Str("client", c.String()).Logger()
}

func (c *Client) Mount(ctx context.Context, share string) (err error) {

	if c.sess == nil {
		return errors.New("SMB session not initialized")
	}

	c.mount, err = c.sess.Mount(share)
	zerolog.Ctx(ctx).Debug().Str("share", share).Msg("Mounted SMB share")
	c.share = share

	return
}

func (c *Client) Connect(ctx context.Context) (err error) {

	log := c.Logger(ctx)
	if c.netDialer == nil {
		return fmt.Errorf("TCP dialer not initialized: call Parse() first")
	}
	if c.dialer == nil {
		return fmt.Errorf("%s dialer not initialized: call Parse() first", c.String())
	}
	if c.Port == 0 {
		return fmt.Errorf("%s port not initialized: call Parse() first", c.String())
	}

	// Establish TCP connection
	address := net.JoinHostPort(c.Host, strconv.FormatUint(uint64(c.Port), 10))
	dialCtx := ctx
	if c.ConnectTimeout > 0 {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, c.ConnectTimeout)
		defer cancel()
	}
	if cd, ok := c.netDialer.(interface {
		DialContext(context.Context, string, string) (net.Conn, error)
	}); ok {
		c.conn, err = cd.DialContext(dialCtx, "tcp", address)
	} else {
		c.conn, err = c.netDialer.Dial("tcp", address)
	}

	if err != nil {
		return err
	}

	log = log.With().Str("address", c.conn.RemoteAddr().String()).Logger()
	log.Debug().Msgf("Connected to %s server", c.String())

	// Open SMB session
	c.sess, err = c.dialer.DialContext(ctx, c.conn)

	if err != nil {
		log.Error().Err(err).Msgf("Failed to open %s session", c.String())
		_ = c.conn.Close()
		c.conn = nil
		return fmt.Errorf("dial %s: %w", c.String(), err)
	}
	log.Debug().Msgf("Opened %s session", c.String())

	c.connected = true

	return
}

func (c *Client) Close(ctx context.Context) (err error) {

	log := c.Logger(ctx)

	c.connected = false

	var errs []error

	// Unmount SMB share
	if c.mount != nil {
		if cleanErr := umountShare(c.mount); cleanErr != nil {
			log.Debug().Err(cleanErr).Msg("Failed to unmount share")
			errs = append(errs, cleanErr)
		} else {
			log.Debug().Msg("Unmounted file share")
		}
		c.mount = nil
		c.share = ""
	}

	// Close SMB session
	if c.sess != nil {
		if cleanErr := logoffSession(c.sess); cleanErr != nil {
			log.Debug().Err(cleanErr).Msgf("Failed to discard SMB session")
			errs = append(errs, cleanErr)
		} else {
			log.Debug().Msg("Discarded SMB session")
		}
		c.sess = nil
	}

	// Close underlying connection (if any)
	if c.conn != nil {
		if cleanErr := c.conn.Close(); cleanErr != nil {
			log.Debug().Err(cleanErr).Msgf("Failed to disconnect SMB client")
			errs = append(errs, cleanErr)
		} else {
			log.Debug().Msg("Disconnected SMB client")
		}
		c.conn = nil
	}

	return errors.Join(errs...)
}
