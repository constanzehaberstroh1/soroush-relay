package soroushlib

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"strings"
	"time"
)

// ──────────────────────────────────────────────────────────────────────────────
// MTProto constructor IDs for messaging (Soroush / Telegram-derived)
// ──────────────────────────────────────────────────────────────────────────────

const (
	// messages.sendMessage — 0x280D096F (TL schema)
	IDSendMessage uint32 = 0x280D096F

	// messages.getDialogs — 0xA062D3F4
	IDGetDialogs uint32 = 0xA062D3F4

	// InputPeerUser — 0x7B8E7DE6
	IDInputPeerUser uint32 = 0x7B8E7DE6

	// updateShortMessage — 0x313BC7F8
	IDUpdateShortMessage uint32 = 0x313BC7F8

	// updateShort — 0x78D4DEC1
	IDUpdateShort uint32 = 0x78D4DEC1

	// updates — 0x74AE4240
	IDUpdates uint32 = 0x74AE4240

	// updatesTooLong — 0xE317AF7E
	IDUpdatesTooLong uint32 = 0xE317AF7E

	// updateNewMessage — 0x1F2B0AFD
	IDUpdateNewMessage uint32 = 0x1F2B0AFD

	// message — 0x38116EE0
	IDMessage uint32 = 0x38116EE0

	// updateShortSentMessage — 0x9015E101
	IDUpdateShortSentMessage uint32 = 0x9015E101
)

// ──────────────────────────────────────────────────────────────────────────────
// Build messages.sendMessage TL payload
// ──────────────────────────────────────────────────────────────────────────────

// BuildSendTextMessage builds a messages.sendMessage request to a user
// userID: the Soroush user ID of the recipient
// accessHash: the access_hash for the target user
// text: message body
func BuildSendTextMessage(userID int64, accessHash int64, text string, randomID int64) []byte {
	w := NewTLWriter()
	w.WriteUint32(IDSendMessage)

	// flags = 0 (no optional fields like reply_to, entities, etc.)
	w.WriteInt32(0)

	// peer = InputPeerUser(user_id, access_hash)
	w.WriteUint32(IDInputPeerUser)
	w.WriteInt64(userID)
	w.WriteInt64(accessHash)

	// message text
	w.WriteString(text)

	// random_id
	w.WriteInt64(randomID)

	return w.GetBytes()
}

// ──────────────────────────────────────────────────────────────────────────────
// Message listener — runs receive loop and dispatches incoming text messages
// ──────────────────────────────────────────────────────────────────────────────

// IncomingMessage represents a received Soroush text message
type IncomingMessage struct {
	FromUserID int64
	Text       string
	Date       int32
	MessageID  int32
}

// ListenForMessages runs a blocking receive loop on the given session.
// It calls the handler function for each incoming text message.
// Returns when the context is cancelled.
func ListenForMessages(ctx context.Context, session *MTProtoSession, handler func(msg IncomingMessage)) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		recvCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		cid, reader, err := session.Recv(recvCtx)
		cancel()

		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// Timeout or transient error — keep listening
			log.Printf("[Messaging] recv error (will retry): %v", err)
			continue
		}

		// Handle different update wrapper types
		processUpdate(cid, reader, session, handler)
	}
}

// processUpdate handles MTProto update messages and extracts text messages
func processUpdate(cid uint32, r *TLReader, session *MTProtoSession, handler func(msg IncomingMessage)) {
	switch cid {
	case IDMsgsAck:
		// Acknowledgment — ignore
		return

	case IDPong:
		// Pong response — ignore
		return

	case IDBadServerSalt:
		// Update server salt
		r.ReadInt64() // bad_msg_id
		r.ReadInt32() // bad_msg_seqno
		r.ReadInt32() // error_code
		newSalt, _ := r.ReadInt64()
		session.ServerSalt = newSalt
		log.Printf("[Messaging] Updated server salt: %d", newSalt)
		return

	case IDNewSession:
		r.ReadInt64() // first_msg_id
		r.ReadInt64() // unique_id
		newSalt, _ := r.ReadInt64()
		session.ServerSalt = newSalt
		log.Printf("[Messaging] New session, updated salt: %d", newSalt)
		return

	case IDMsgContainer:
		// Parse container
		count, _ := r.ReadInt32()
		for i := int32(0); i < count; i++ {
			r.ReadInt64() // msg_id
			r.ReadInt32() // seq_no
			bodyLen, _ := r.ReadInt32()
			body, err := r.ReadRaw(int(bodyLen))
			if err != nil {
				continue
			}
			subReader := NewTLReader(body)
			subCID, _ := subReader.ReadUint32()
			processUpdate(subCID, subReader, session, handler)
		}
		return

	case IDRPCResult:
		// RPC result (from our sendMessage, etc.) — can contain updates
		r.ReadInt64() // req_msg_id
		innerCID, _ := r.ReadUint32()
		processUpdate(innerCID, r, session, handler)
		return

	case IDUpdateShortMessage:
		// Direct short message from a user
		parseUpdateShortMessage(r, handler)
		return

	case IDUpdates:
		// Full updates container
		parseUpdates(r, handler)
		return

	case IDUpdateShort:
		// Single update wrapper
		innerCID, _ := r.ReadUint32()
		if innerCID == IDUpdateNewMessage {
			parseUpdateNewMessage(r, handler)
		}
		return

	case IDUpdateShortSentMessage:
		// Our sent message was confirmed — ignore
		return
	}
}

// parseUpdateShortMessage extracts a short incoming message
func parseUpdateShortMessage(r *TLReader, handler func(msg IncomingMessage)) {
	flags, _ := r.ReadInt32()
	msgID, _ := r.ReadInt32()
	userID, _ := r.ReadInt64()
	text, _ := r.ReadString()
	_ = flags // flags contain out/mentioned/media_unread etc.

	handler(IncomingMessage{
		FromUserID: userID,
		Text:       text,
		MessageID:  msgID,
	})
}

// parseUpdateNewMessage extracts a message from updateNewMessage
func parseUpdateNewMessage(r *TLReader, handler func(msg IncomingMessage)) {
	// message constructor
	msgCID, _ := r.ReadUint32()
	if msgCID != IDMessage {
		return
	}

	flags, _ := r.ReadInt32()
	_ = flags
	msgID, _ := r.ReadInt32()

	// from_id (PeerUser) if flags bit 8 is set
	var fromUserID int64
	if flags&(1<<8) != 0 {
		r.ReadUint32() // PeerUser constructor
		fromUserID, _ = r.ReadInt64()
	}

	// peer_id (PeerUser)
	r.ReadUint32() // PeerUser constructor
	r.ReadInt64()  // peer user_id

	// message text
	text, _ := r.ReadString()

	if fromUserID != 0 {
		handler(IncomingMessage{
			FromUserID: fromUserID,
			Text:       text,
			MessageID:  msgID,
		})
	}
}

// parseUpdates parses a full updates object
func parseUpdates(r *TLReader, handler func(msg IncomingMessage)) {
	// updates vector
	r.ReadUint32() // vector constructor
	count, _ := r.ReadInt32()
	for i := int32(0); i < count; i++ {
		updateCID, _ := r.ReadUint32()
		if updateCID == IDUpdateNewMessage {
			parseUpdateNewMessage(r, handler)
		}
		// Skip other update types gracefully
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Send message helper — fires and forgets
// ──────────────────────────────────────────────────────────────────────────────

// SendTextMessage sends a text message to a Soroush user via MTProto
func SendTextMessage(ctx context.Context, session *MTProtoSession, userID int64, accessHash int64, text string) error {
	randomID := time.Now().UnixNano()
	body := BuildSendTextMessage(userID, accessHash, text, randomID)

	_, err := session.Send(ctx, body, true)
	if err != nil {
		return fmt.Errorf("send text message: %w", err)
	}
	log.Printf("[Messaging] Sent message to user %d: %s", userID, truncate(text, 50))
	return nil
}

// truncate helper for logging
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ──────────────────────────────────────────────────────────────────────────────
// Dispatcher protocol constants
// ──────────────────────────────────────────────────────────────────────────────

const (
	// DispatcherSynRequest is sent by the client to the dispatcher
	DispatcherSynRequest = "SYN_REQ_V1"

	// DispatcherAckRoutePrefix is the reply from the dispatcher with the worker ID
	DispatcherAckRoutePrefix = "ACK_ROUTE:"

	// DispatcherNoWorkers indicates no idle workers available
	DispatcherNoWorkers = "NACK_NO_WORKERS"
)

// ParseDispatcherResponse parses a dispatcher reply and extracts the worker user ID and access hash
// Format: "ACK_ROUTE:<user_id>:<access_hash>"
func ParseDispatcherResponse(text string) (workerUserID int64, workerAccessHash int64, ok bool) {
	if !strings.HasPrefix(text, DispatcherAckRoutePrefix) {
		return 0, 0, false
	}

	payload := text[len(DispatcherAckRoutePrefix):]
	parts := strings.Split(payload, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}

	userIDBytes := []byte(parts[0])
	accessHashBytes := []byte(parts[1])

	// Parse user ID
	uid, err := parseI64(userIDBytes)
	if err != nil {
		return 0, 0, false
	}

	// Parse access hash
	ah, err := parseI64(accessHashBytes)
	if err != nil {
		return 0, 0, false
	}

	return uid, ah, true
}

// parseI64 parses a decimal string to int64
func parseI64(b []byte) (int64, error) {
	s := string(b)
	var v int64
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}

// FormatDispatcherResponse formats a worker assignment response
func FormatDispatcherResponse(workerUserID int64, workerAccessHash int64) string {
	return fmt.Sprintf("%s%d:%d", DispatcherAckRoutePrefix, workerUserID, workerAccessHash)
}

// ──────────────────────────────────────────────────────────────────────────────
// Session restore from saved credentials
// ──────────────────────────────────────────────────────────────────────────────

// RestoreSession creates a MTProtoSession from saved auth credentials
func RestoreSession(authKey []byte, authKeyID []byte, serverSalt []byte) (*MTProtoSession, *ObfuscatedTransport) {
	transport := NewTransport()
	session := NewSession(transport)

	session.AuthKey = authKey

	if len(authKeyID) >= 8 {
		session.AuthKeyID = int64(binary.LittleEndian.Uint64(authKeyID))
	}
	if len(serverSalt) >= 8 {
		session.ServerSalt = int64(binary.LittleEndian.Uint64(serverSalt))
	}

	return session, transport
}
