package antivirus

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeClamd accepts one connection, reads a z-command (and the INSTREAM chunks), then
// answers with reply(received bytes).
func fakeClamd(t *testing.T, reply func(cmd string, data []byte) string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				br := bufio.NewReader(conn)
				cmd, err := br.ReadString(0)
				if err != nil {
					return
				}
				cmd = strings.TrimSuffix(strings.TrimPrefix(cmd, "z"), "\x00")
				var data bytes.Buffer
				if cmd == "INSTREAM" {
					for {
						var n uint32
						if err := binary.Read(br, binary.BigEndian, &n); err != nil {
							return
						}
						if n == 0 {
							break
						}
						if _, err := io.CopyN(&data, br, int64(n)); err != nil {
							return
						}
					}
				}
				r := reply(cmd, data.Bytes())
				if r == "" {
					return // hang up without answering
				}
				_, _ = conn.Write([]byte(r + "\x00"))
			}(conn)
		}
	}()
	return ln.Addr().String()
}

func TestScanClean(t *testing.T) {
	var got []byte
	addr := fakeClamd(t, func(cmd string, data []byte) string {
		got = data
		return "stream: OK"
	})
	c := &Clamd{Address: addr, Timeout: 5 * time.Second}
	payload := bytes.Repeat([]byte("x"), 3*chunkSize+17)
	sig, err := c.Scan(context.Background(), bytes.NewReader(payload))
	if err != nil || sig != "" {
		t.Fatalf("Scan = %q, %v; want clean", sig, err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("clamd got %d bytes, want %d", len(got), len(payload))
	}
}

func TestScanFound(t *testing.T) {
	addr := fakeClamd(t, func(string, []byte) string { return "stream: Win.Test.EICAR_HDB-1 FOUND" })
	c := &Clamd{Address: addr, Timeout: 5 * time.Second}
	sig, err := c.Scan(context.Background(), strings.NewReader("X5O!P%@AP"))
	if err != nil || sig != "Win.Test.EICAR_HDB-1" {
		t.Fatalf("Scan = %q, %v; want the EICAR signature", sig, err)
	}
}

func TestScanErrorReply(t *testing.T) {
	addr := fakeClamd(t, func(string, []byte) string { return "INSTREAM size limit exceeded. ERROR" })
	c := &Clamd{Address: addr, Timeout: 5 * time.Second}
	if _, err := c.Scan(context.Background(), strings.NewReader("data")); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Scan error = %v, want ErrUnavailable", err)
	}
}

func TestScanHangUp(t *testing.T) {
	addr := fakeClamd(t, func(string, []byte) string { return "" })
	c := &Clamd{Address: addr, Timeout: 5 * time.Second}
	if _, err := c.Scan(context.Background(), strings.NewReader("data")); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Scan error = %v, want ErrUnavailable", err)
	}
}

func TestScanTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, err := ln.Accept()
		if err == nil {
			defer conn.Close()
			time.Sleep(2 * time.Second) // never answers
		}
	}()
	c := &Clamd{Address: ln.Addr().String(), Timeout: 200 * time.Millisecond}
	start := time.Now()
	if _, err := c.Scan(context.Background(), strings.NewReader("data")); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Scan error = %v, want ErrUnavailable", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("Scan took %v, want the 200ms timeout", time.Since(start))
	}
}

func TestScanNoServer(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close()
	c := &Clamd{Address: addr, Timeout: time.Second}
	if _, err := c.Scan(context.Background(), strings.NewReader("data")); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Scan error = %v, want ErrUnavailable", err)
	}
}

func TestPing(t *testing.T) {
	addr := fakeClamd(t, func(cmd string, _ []byte) string {
		if cmd == "PING" {
			return "PONG"
		}
		return "UNKNOWN COMMAND"
	})
	c := &Clamd{Address: addr, Timeout: time.Second}
	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestNewWithoutAddress(t *testing.T) {
	if New(Config{}) != nil {
		t.Fatal("New without an address should turn scanning off")
	}
}
