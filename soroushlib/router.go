package soroushlib

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// UpdateMessage represents a raw MTProto update packet
type UpdateMessage struct {
	CID  uint32
	Data []byte
}

// MessageRouter coordinates reading from one MTProto session and broadcasting to subscribers
type MessageRouter struct {
	session    *MTProtoSession
	mu         sync.Mutex
	subsUpdate []chan UpdateMessage
	subsText   []chan IncomingMessage
	running    bool
	done       chan struct{}
}

// NewMessageRouter creates a new MessageRouter
func NewMessageRouter(session *MTProtoSession) *MessageRouter {
	return &MessageRouter{
		session: session,
		done:    make(chan struct{}),
	}
}

// SubscribeUpdate returns a channel receiving raw MTProto updates
func (mr *MessageRouter) SubscribeUpdate() chan UpdateMessage {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	ch := make(chan UpdateMessage, 500)
	mr.subsUpdate = append(mr.subsUpdate, ch)
	return ch
}

// UnsubscribeUpdate removes an update subscription
func (mr *MessageRouter) UnsubscribeUpdate(ch chan UpdateMessage) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	for i, c := range mr.subsUpdate {
		if c == ch {
			mr.subsUpdate = append(mr.subsUpdate[:i], mr.subsUpdate[i+1:]...)
			// Don't close here — Run()'s defer handles cleanup to avoid double-close panic
			break
		}
	}
}

// SubscribeText returns a channel receiving parsed text messages
func (mr *MessageRouter) SubscribeText() chan IncomingMessage {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	ch := make(chan IncomingMessage, 500)
	mr.subsText = append(mr.subsText, ch)
	return ch
}

// UnsubscribeText removes a text subscription
func (mr *MessageRouter) UnsubscribeText(ch chan IncomingMessage) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	for i, c := range mr.subsText {
		if c == ch {
			mr.subsText = append(mr.subsText[:i], mr.subsText[i+1:]...)
			// Don't close here — Run()'s defer handles cleanup to avoid double-close panic
			break
		}
	}
}

// Run starts the read loop from the MTProto session and broadcasts to subscribers
func (mr *MessageRouter) Run(ctx context.Context) error {
	mr.mu.Lock()
	if mr.running {
		mr.mu.Unlock()
		return fmt.Errorf("router already running")
	}
	mr.running = true
	mr.mu.Unlock()

	defer func() {
		mr.mu.Lock()
		mr.running = false
		// Close all subscribers
		for _, ch := range mr.subsUpdate {
			// Drain and close
			select {
			case <-ch:
			default:
			}
			close(ch)
		}
		mr.subsUpdate = nil
		for _, ch := range mr.subsText {
			select {
			case <-ch:
			default:
			}
			close(ch)
		}
		mr.subsText = nil
		mr.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		cid, reader, err := mr.session.Recv(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			errStr := err.Error()
			if strings.Contains(errStr, "closed network connection") ||
				strings.Contains(errStr, "broken pipe") ||
				strings.Contains(errStr, "connection reset") ||
				strings.Contains(errStr, "EOF") {
				return fmt.Errorf("connection lost: %w", err)
			}
			return fmt.Errorf("recv error: %w", err)
		}

		// Save the raw reader bytes before reading from it
		var rawBytes []byte
		if reader != nil {
			rem := reader.Remaining()
			rawBytes, _ = reader.ReadRaw(rem)
			// Recreate the reader for our own processing
			reader = NewTLReader(rawBytes)
		}

		// Dispatch raw update
		mr.mu.Lock()
		for _, ch := range mr.subsUpdate {
			select {
			case ch <- UpdateMessage{CID: cid, Data: rawBytes}:
			default:
				// Avoid blocking on slow channel
			}
		}
		mr.mu.Unlock()

		// Also check if we can process it as text message updates
		if reader != nil {
			processUpdate(cid, reader, mr.session, func(msg IncomingMessage) {
				mr.mu.Lock()
				for _, ch := range mr.subsText {
					select {
					case ch <- msg:
					default:
					}
				}
				mr.mu.Unlock()
			})
		}
	}
}
