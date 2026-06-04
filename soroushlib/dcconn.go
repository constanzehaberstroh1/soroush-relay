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
	dc          *webrtc.DataChannel
	buf         chan []byte
	rem         []byte
	closed      chan struct{}
	once        sync.Once
	lastRxMutex sync.Mutex
	lastRxTime  time.Time
}

// NewDataChannelConn creates a new DataChannelConn adapter.
// It registers an OnMessage handler on the DataChannel to buffer incoming data.
func NewDataChannelConn(dc *webrtc.DataChannel) *DataChannelConn {
	c := &DataChannelConn{
		dc:         dc,
		buf:        make(chan []byte, 512),
		closed:     make(chan struct{}),
		lastRxTime: time.Now(),
	}
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		c.lastRxMutex.Lock()
		c.lastRxTime = time.Now()
		c.lastRxMutex.Unlock()

		// Intercept stealth ping-pong frames to prevent polluting yamux buffer
		if len(msg.Data) == 4 && string(msg.Data) == "PING" {
			_ = dc.Send([]byte("PONG"))
			return
		}
		if len(msg.Data) == 4 && string(msg.Data) == "PONG" {
			return
		}

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

	// Start DataChannel active keep-alive and dead-link detection loop
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.lastRxMutex.Lock()
				lastRx := c.lastRxTime
				c.lastRxMutex.Unlock()

				// If no message or keepalive pong received for 45 seconds, assume dead
				if time.Since(lastRx) > 45*time.Second {
					c.Close()
					return
				}

				// Proactively send a ping to keep NAT bindings open and verify path health
				if err := dc.Send([]byte("PING")); err != nil {
					c.Close()
					return
				}
			case <-c.closed:
				return
			}
		}
	}()

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
