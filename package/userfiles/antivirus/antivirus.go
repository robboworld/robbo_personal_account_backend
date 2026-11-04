// Package antivirus scans "Мои файлы" uploads with ClamAV (clamd over TCP) before a file
// can reach another account: a teacher reviewing homework, or a class getting a starter file.
package antivirus

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// ErrUnavailable — clamd could not be reached or did not finish the scan.
var ErrUnavailable = errors.New("antivirus unavailable")

const (
	defaultTimeout = 2 * time.Minute
	chunkSize      = 64 << 10
)

// Config is userFiles.antivirus in config.yml (USERFILES_ANTIVIRUS_*).
type Config struct {
	// Address of clamd, host:port; empty turns scanning off.
	Address string
	// Timeout of one scan, connection included.
	Timeout time.Duration
	// Required: an upload is refused when the scan cannot run. Exchanging files between
	// accounts (file assignments in classes) is only on when this is set.
	Required bool
}

func ConfigFromViper() Config {
	c := Config{
		Address:  strings.TrimSpace(viper.GetString("userFiles.antivirus.address")),
		Timeout:  viper.GetDuration("userFiles.antivirus.timeout"),
		Required: viper.GetBool("userFiles.antivirus.required"),
	}
	if c.Timeout <= 0 {
		c.Timeout = defaultTimeout
	}
	return c
}

type Scanner interface {
	// Scan returns the name of the signature the contents match, or "" when they are clean.
	Scan(ctx context.Context, r io.Reader) (signature string, err error)
	Ping(ctx context.Context) error
}

// New returns a clamd client, or nil when no address is configured.
func New(c Config) Scanner {
	if c.Address == "" {
		return nil
	}
	return &Clamd{Address: c.Address, Timeout: c.Timeout}
}

// Clamd speaks the clamd protocol: one connection per command, z-prefixed (NUL-terminated).
type Clamd struct {
	Address string
	Timeout time.Duration
}

// dial connects with a deadline for the whole command: Timeout, or ctx's if it is sooner.
func (c *Clamd) dial(ctx context.Context) (net.Conn, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", c.Address)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	deadline, _ := ctx.Deadline()
	_ = conn.SetDeadline(deadline)
	return conn, nil
}

func (c *Clamd) Ping(ctx context.Context) error {
	conn, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("zPING\x00")); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	reply, err := readReply(conn)
	if err != nil {
		return err
	}
	if reply != "PONG" {
		return fmt.Errorf("%w: unexpected reply %q", ErrUnavailable, reply)
	}
	return nil
}

// Scan streams r with INSTREAM: chunks prefixed by their length (uint32, big-endian),
// then a zero length. clamd answers "stream: OK", "stream: <signature> FOUND" or "... ERROR".
func (c *Clamd) Scan(ctx context.Context, r io.Reader) (string, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	buf := make([]byte, 4+chunkSize)
	for {
		n, readErr := r.Read(buf[4:])
		if n > 0 {
			binary.BigEndian.PutUint32(buf[:4], uint32(n))
			if _, err := conn.Write(buf[:4+n]); err != nil {
				// clamd closes the stream past StreamMaxLength; its reply says so.
				if reply, rerr := readReply(conn); rerr == nil {
					return "", fmt.Errorf("%w: %s", ErrUnavailable, reply)
				}
				return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	reply, err := readReply(conn)
	if err != nil {
		return "", err
	}
	return parseScanReply(reply)
}

func readReply(conn net.Conn) (string, error) {
	b, err := io.ReadAll(io.LimitReader(conn, 4096))
	if err != nil && len(b) == 0 {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return strings.TrimSpace(string(bytes.TrimRight(b, "\x00"))), nil
}

func parseScanReply(reply string) (string, error) {
	// "stream: OK", "stream: Win.Test.EICAR_HDB-1 FOUND"; older clamd answer without "stream:".
	body := strings.TrimSpace(strings.TrimPrefix(reply, "stream:"))
	switch {
	case body == "OK":
		return "", nil
	case strings.HasSuffix(body, " FOUND"):
		sig := strings.TrimSpace(strings.TrimSuffix(body, " FOUND"))
		if sig == "" {
			sig = "unknown"
		}
		return sig, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnavailable, reply)
	}
}
