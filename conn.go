package sower

import (
	"errors"
	"io"
	"net"
)

// Conn is ...
type Conn struct {
	net.Conn
	r io.Reader
}

// Read is ...
func (c *Conn) Read(b []byte) (int, error) {
	if c.r == nil {
		return c.Conn.Read(b)
	}

	n, err := c.r.Read(b)
	if err != nil && errors.Is(err, io.EOF) {
		c.r = nil
		return n, nil
	}

	return n, err
}
