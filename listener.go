package sower

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/netip"
	"time"

	"go.uber.org/zap"
)

// Result is ...
type Result struct {
	Conn net.Conn
	Err  error
}

// Listener is ...
type Listener struct {
	ar     AddressRange
	dl     DomainList
	lg     *zap.Logger
	ln     net.Listener
	connCh chan Result
}

func (l *Listener) loop() {
	for {
		conn, err := l.ln.Accept()
		if err != nil {
			return
		}

		go func() {
			buffer := bytes.NewBuffer(nil)
			domain := ""

			if err := tls.Server(&Conn{
				Conn: conn,
				r:    io.TeeReader(conn, buffer),
			}, &tls.Config{
				GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
					domain = hello.ServerName
					return nil, nil
				},
			}).Handshake(); err != nil {
				conn.Close()
				return
			}

			if l.dl != nil && l.dl.Match(domain) {
				select {
				case <-l.connCh:
					return
				default:
					l.connCh <- Result{&Conn{
						Conn: conn,
						r:    bytes.NewReader(buffer.Bytes()),
					}, err}
				}
				return
			}

			addr, err := func() (netip.Addr, error) {
				raddr := conn.RemoteAddr()
				if naddr, ok := raddr.(*net.TCPAddr); ok {
					if ip := naddr.IP.To4(); ip != nil {
						return netip.AddrFrom4([4]byte{ip[0], ip[1], ip[2], ip[3]}), nil
					}
					ip := naddr.IP.To16()
					return netip.AddrFrom16([16]byte{ip[0], ip[1], ip[2], ip[3], ip[4], ip[5], ip[6], ip[7], ip[8], ip[9], ip[10], ip[11], ip[12], ip[13], ip[14], ip[15]}), nil
				}
				return netip.ParseAddr(raddr.String())
			}()
			if err != nil {
				conn.Close()
				return
			}

			if l.ar.Contains(addr) {
				defer conn.Close()

				rc, err := net.Dial("tcp", domain)
				if err != nil {
					return
				}
				defer rc.Close()

				errCh := make(chan error)

				go func() {
					if _, err := io.Copy(conn, rc); err != nil {
						conn.SetReadDeadline(time.Now())
						errCh <- err
						return
					}
					errCh <- nil
				}()

				if _, err := io.Copy(rc, &Conn{
					Conn: conn,
					r:    bytes.NewReader(buffer.Bytes()),
				}); err != nil {
					rc.SetReadDeadline(time.Now())
					l.lg.Error(fmt.Sprintf("io.Copy error: %v", err))
					<-errCh
					return
				}

				if err := <-errCh; err != nil {
					l.lg.Error(fmt.Sprintf("io.Copy error: %v", err))
				}

				return
			}

			select {
			case <-l.connCh:
				return
			default:
				l.connCh <- Result{&Conn{
					Conn: conn,
					r:    bytes.NewReader(buffer.Bytes()),
				}, err}
			}
		}()
	}
}

func (l *Listener) Addr() net.Addr {
	return l.ln.Addr()
}

func (l *Listener) Accept() (net.Conn, error) {
	re, ok := <-l.connCh
	if ok {
		return re.Conn, re.Err
	}
	return nil, net.ErrClosed
}

func (l *Listener) Close() error {
	select {
	case _, ok := <-l.connCh:
		if ok {
			close(l.connCh)
			return nil
		}
		return net.ErrClosed
	default:
		close(l.connCh)
	}
	return nil
}

var _ net.Listener = (*Listener)(nil)
