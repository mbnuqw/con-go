package con

import (
	"net"
	"strings"
	"time"
)

// Conn - common interface for TcpConn and UnixConn.
type Conn interface {
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	Close() error
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	SetReadBuffer(bytes int) error
	SetWriteBuffer(bytes int) error
}

// Dial checks type of connection and Dial
func Dial(address string) (c Conn, err error) {
	if strings.HasPrefix(address, "/") && strings.HasSuffix(address, ".sock") {
		addr, err := net.ResolveUnixAddr("unix", address)
		c, err = net.DialUnix("unix", nil, addr)
		if err != nil {
			return nil, err
		}
	} else {
		addr, err := net.ResolveTCPAddr("tcp", address)
		c, err = net.DialTCP("tcp", nil, addr)
		if err != nil {
			return nil, err
		}
	}
	return
}
