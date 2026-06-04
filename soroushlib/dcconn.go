package soroushlib

import (
	"net"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

// DataChannelConn wraps a WebRTC DataChannel to implement net.Conn.
// This allows stream-oriented protocols (like yamux) to multiplex over the
// message-oriented DataChannel.
type DataChannelConn struct {
	dc     *webrtc.DataChannel
	buf    chan []byte
	rem    []byte
	closed chan struct{}
	once   sync.Once
}

// NewDataChannelConn creates a new DataChannelConn adapter.
// It registers an OnMessage handler on the DataChannel to buffer incoming data.
func NewDataChannelConn(dc *webrtc.DataChannel) *DataChannelConn {
	c := &DataChannelConn{
		dc:     dc,
		buf:    make(chan []byte, 512),
		closed: make(chan struct{}),
	}
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		data := make([]byte, len(msg.Data))
		copy(data, msg.Data)
		select {
		case c.buf <- data:
		case <-c.closed:
		}
	})
	dc.OnClose(func() {
		c.Close()
	})
	return c
}

// Read implements net.Conn. Blocks until data is available.
func (c *DataChannelConn) Read(b []byte) (int, error) {
	if len(c.rem) > 0 {
		n := copy(b, c.rem)
		c.rem = c.rem[n:]
		return n, nil
	}
	select {
	case data, ok := <-c.buf:
		if !ok {
			return 0, net.ErrClosed
		}
		n := copy(b, data)
		if n < len(data) {
			c.rem = data[n:]
		}
		return n, nil
	case <-c.closed:
		return 0, net.ErrClosed
	}
}

// Write implements net.Conn. Sends data as a DataChannel message.
func (c *DataChannelConn) Write(b []byte) (int, error) {
	chunk := make([]byte, len(b))
	copy(chunk, b)
	if err := c.dc.Send(chunk); err != nil {
		return 0, err
	}
	return len(b), nil
}

// Close implements net.Conn.
func (c *DataChannelConn) Close() error {
	c.once.Do(func() {
		close(c.closed)
	})
	return c.dc.Close()
}

// LocalAddr implements net.Conn.
func (c *DataChannelConn) LocalAddr() net.Addr {
	return dcAddr("local")
}

// RemoteAddr implements net.Conn.
func (c *DataChannelConn) RemoteAddr() net.Addr {
	return dcAddr("remote")
}

// SetDeadline implements net.Conn (no-op).
func (c *DataChannelConn) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline implements net.Conn (no-op).
func (c *DataChannelConn) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline implements net.Conn (no-op).
func (c *DataChannelConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type dcAddr string

func (a dcAddr) Network() string {
	return "datachannel"
}

func (a dcAddr) String() string {
	return string(a)
}

var _ net.Conn = (*DataChannelConn)(nil)
