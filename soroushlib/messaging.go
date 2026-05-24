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

	// messages.getDialogs — 0xA0F4CB4F (Soroush layer 182)
	IDGetDialogs uint32 = 0xA0F4CB4F

	// InputPeerUser — 0xDDE8A54C (Soroush layer 182)
	IDInputPeerUser uint32 = 0xDDE8A54C

	// InputPeerChat — 0x35A95CB9 (for group chats)
	IDInputPeerChat uint32 = 0x35A95CB9

	// PeerUser — 0x59511722
	IDPeerUser uint32 = 0x59511722

	// PeerChat — 0x36C6019A (group chat peer identifier)
	IDPeerChat uint32 = 0x36C6019A

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

	// InputPeerEmpty — 0x7F3B18EA
	IDInputPeerEmpty uint32 = 0x7F3B18EA

	// messages.dialogs — 0x15BA6C40
	IDDialogs uint32 = 0x15BA6C40

	// messages.dialogsSlice — 0x71E094F3
	IDDialogsSlice uint32 = 0x71E094F3

	// dialog — 0xD58A08C6
	IDDialog uint32 = 0xD58A08C6

	// PeerChannel — 0xA2A5371E
	IDPeerChannel uint32 = 0xA2A5371E

	// chat — 0x41CBF256
	IDChat uint32 = 0x41CBF256

	// chatForbidden — 0x6592A1A7
	IDChatForbidden uint32 = 0x6592A1A7

	// channel — 0x8E87CCD8 (Soroush layer 182)
	IDChannel uint32 = 0x8E87CCD8

	// channelForbidden — 0x17D493D5
	IDChannelForbidden uint32 = 0x17D493D5
)

// ──────────────────────────────────────────────────────────────────────────────
// Build messages.getDialogs TL payload
// ──────────────────────────────────────────────────────────────────────────────

// DialogInfo represents a chat/group from the user's dialog list
type DialogInfo struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	Type         string `json:"type"` // "group", "channel", "supergroup"
	MembersCount int32  `json:"membersCount"`
}

// BuildGetDialogsRequest builds a messages.getDialogs request
// This fetches the user's chat list (groups, channels, DMs)
func BuildGetDialogsRequest() []byte {
	w := NewTLWriter()
	w.WriteUint32(IDGetDialogs)

	// flags = 0 (no exclude_pinned, folder_id, etc.)
	w.WriteInt32(0)

	// offset_date = 0
	w.WriteInt32(0)

	// offset_id = 0
	w.WriteInt32(0)

	// offset_peer = InputPeerEmpty
	w.WriteUint32(IDInputPeerEmpty)

	// limit = 100
	w.WriteInt32(100)

	// hash = 0
	w.WriteInt64(0)

	return w.GetBytes()
}

// ParseDialogsForGroups extracts group chats from a messages.getDialogs response
// It scans the chats vector within the response for Chat and Channel objects
func ParseDialogsForGroups(cid uint32, r *TLReader) ([]DialogInfo, error) {
	if cid == IDRPCError {
		return nil, ParseRPCError(r)
	}

	// Both messages.dialogs and messages.dialogsSlice have similar structures
	// messages.dialogsSlice has an extra count field at the start
	if cid == IDDialogsSlice {
		r.ReadInt32() // count
	} else if cid != IDDialogs {
		return nil, fmt.Errorf("unexpected response cid=0x%08X for getDialogs", cid)
	}

	// Skip dialogs vector
	skipTLVector(r)

	// Skip messages vector
	skipTLVector(r)

	// Parse chats vector — this is where groups/channels live
	groups := parseChatVector(r)

	return groups, nil
}

// skipTLVector skips over a TL vector (reads and discards all elements)
func skipTLVector(r *TLReader) {
	r.ReadUint32() // vector constructor ID
	count, _ := r.ReadInt32()
	for i := int32(0); i < count; i++ {
		// We don't know the exact size of each element, so we skip by reading
		// the constructor and trying to parse minimally
		// For dialogs and messages, we'll just skip a reasonable amount
		startPos := r.Remaining()
		skipTLObject(r)
		// If we didn't consume anything, break to avoid infinite loop
		if r.Remaining() == startPos {
			break
		}
	}
}

// skipTLObject tries to skip a single TL object (best-effort for dialogs/messages)
func skipTLObject(r *TLReader) {
	cid, err := r.ReadUint32()
	if err != nil {
		return
	}

	switch cid {
	case IDDialog: // dialog#d58a08c6
		r.ReadInt32()  // flags
		r.ReadUint32() // peer constructor
		r.ReadInt64()  // peer id
		r.ReadInt32()  // top_message
		r.ReadInt32()  // read_inbox_max_id
		r.ReadInt32()  // read_outbox_max_id
		r.ReadInt32()  // unread_count
		r.ReadInt32()  // unread_mentions_count
		r.ReadInt32()  // unread_reactions_count
		// notify_settings (peerNotifySettings)
		skipNotifySettings(r)
	case IDMessage: // message
		flags, _ := r.ReadInt32()
		r.ReadInt32() // id
		if flags&(1<<8) != 0 {
			r.ReadUint32() // from_id peer constructor
			r.ReadInt64()  // from_id value
		}
		r.ReadUint32() // peer_id constructor
		r.ReadInt64()  // peer_id value
		// We can't reliably skip the rest, so stop here
	default:
		// Unknown — try to skip a few fields
		for j := 0; j < 8; j++ {
			if r.Remaining() <= 0 {
				break
			}
			r.ReadInt32()
		}
	}
}

// skipNotifySettings skips a peerNotifySettings object
func skipNotifySettings(r *TLReader) {
	r.ReadUint32() // constructor
	flags, _ := r.ReadInt32()
	if flags&(1<<0) != 0 {
		r.ReadInt32() // show_previews (Bool)
	}
	if flags&(1<<1) != 0 {
		r.ReadInt32() // silent (Bool)
	}
	if flags&(1<<2) != 0 {
		r.ReadInt32() // mute_until
	}
	if flags&(1<<3) != 0 {
		r.ReadString() // sound (NotificationSound — skip as string)
	}
}

// parseChatVector parses the chats vector and extracts group/channel info
func parseChatVector(r *TLReader) []DialogInfo {
	var groups []DialogInfo

	r.ReadUint32() // vector constructor ID (0x1cb5c415)
	count, err := r.ReadInt32()
	if err != nil || count <= 0 {
		return groups
	}

	for i := int32(0); i < count; i++ {
		cid, err := r.ReadUint32()
		if err != nil {
			break
		}

		switch cid {
		case IDChat, 0x29562865: // chat or chatEmpty variants
			info := parseChatObject(r, cid)
			if info != nil {
				groups = append(groups, *info)
			}
		case IDChannel, 0x94F592DB, 0x8261AC61: // channel variants across layers
			info := parseChannelObject(r, cid)
			if info != nil {
				groups = append(groups, *info)
			}
		case IDChatForbidden:
			// chat_id + title
			id, _ := r.ReadInt64()
			title, _ := r.ReadString()
			groups = append(groups, DialogInfo{
				ID:    id,
				Title: title + " (forbidden)",
				Type:  "group",
			})
		case IDChannelForbidden:
			flags, _ := r.ReadInt32()
			id, _ := r.ReadInt64()
			_ = flags
			r.ReadInt64() // access_hash
			title, _ := r.ReadString()
			groups = append(groups, DialogInfo{
				ID:    id,
				Title: title + " (forbidden)",
				Type:  "channel",
			})
		default:
			// Unknown chat type — try to skip
			log.Printf("[Dialogs] Unknown chat constructor: 0x%08X", cid)
			// Read some fields to try to advance
			r.ReadInt64()
			r.ReadString()
		}
	}

	return groups
}

// parseChatObject parses a chat (group) TL object
func parseChatObject(r *TLReader, cid uint32) *DialogInfo {
	flags, _ := r.ReadInt32()
	id, _ := r.ReadInt64()
	title, _ := r.ReadString()

	// photo — skip
	photoCID, _ := r.ReadUint32()
	if photoCID != 0x37C1011C { // chatPhotoEmpty
		// chatPhoto — skip fields
		r.ReadInt32()  // flags
		r.ReadInt64()  // photo_id
		if r.Remaining() > 8 {
			r.ReadBytes() // stripped_thumb (optional based on flags)
		}
		r.ReadInt32() // dc_id
	}

	participantsCount, _ := r.ReadInt32()
	r.ReadInt32() // date
	r.ReadInt32() // version

	_ = flags

	return &DialogInfo{
		ID:           id,
		Title:        title,
		Type:         "group",
		MembersCount: participantsCount,
	}
}

// parseChannelObject parses a channel/supergroup TL object
func parseChannelObject(r *TLReader, cid uint32) *DialogInfo {
	flags, _ := r.ReadInt32()
	_ = flags
	id, _ := r.ReadInt64()

	// access_hash (if flags bit 13)
	if flags&(1<<13) != 0 {
		r.ReadInt64()
	}

	title, _ := r.ReadString()

	chatType := "channel"
	if flags&(1<<8) != 0 { // megagroup flag
		chatType = "supergroup"
	}

	return &DialogInfo{
		ID:    id,
		Title: title,
		Type:  chatType,
	}
}

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

// BuildSendGroupMessage builds a messages.sendMessage request to a group chat
// chatID: the Soroush group chat ID
// text: message body
func BuildSendGroupMessage(chatID int64, text string, randomID int64) []byte {
	w := NewTLWriter()
	w.WriteUint32(IDSendMessage)

	// flags = 0
	w.WriteInt32(0)

	// peer = InputPeerChat(chat_id)
	w.WriteUint32(IDInputPeerChat)
	w.WriteInt64(chatID)

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
	ChatID     int64 // non-zero if this is a group message
	IsGroup    bool  // true if message came from a group chat
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

	// peer_id — can be PeerUser or PeerChat (group)
	var chatID int64
	var isGroup bool
	peerCID, _ := r.ReadUint32()
	switch peerCID {
	case IDPeerChat:
		chatID, _ = r.ReadInt64()
		isGroup = true
	case IDPeerUser:
		r.ReadInt64() // peer user_id (not needed, we have from_id)
	default:
		// Unknown peer type — try to read int64 and continue
		r.ReadInt64()
	}

	// message text
	text, _ := r.ReadString()

	if fromUserID != 0 {
		handler(IncomingMessage{
			FromUserID: fromUserID,
			Text:       text,
			MessageID:  msgID,
			ChatID:     chatID,
			IsGroup:    isGroup,
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

// SendGroupMessage sends a text message to a Soroush group chat via MTProto
func SendGroupMessage(ctx context.Context, session *MTProtoSession, chatID int64, text string) error {
	randomID := time.Now().UnixNano()
	body := BuildSendGroupMessage(chatID, text, randomID)

	_, err := session.Send(ctx, body, true)
	if err != nil {
		return fmt.Errorf("send group message: %w", err)
	}
	log.Printf("[Messaging] Sent group message to chat %d: %s", chatID, truncate(text, 50))
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
// SDP Exchange Protocol — direct messages for WebRTC signaling
// ──────────────────────────────────────────────────────────────────────────────

const (
	// SDPOfferPrefix is sent by client to worker with the SDP offer
	SDPOfferPrefix = "SDP_OFFER:"

	// SDPAnswerPrefix is sent by worker to client with the SDP answer
	SDPAnswerPrefix = "SDP_ANSWER:"

	// ICECandidatePrefix is sent by both peers to exchange ICE candidates
	ICECandidatePrefix = "ICE:"
)

// IsSDPOffer checks if a message is an SDP offer
func IsSDPOffer(text string) bool {
	return strings.HasPrefix(text, SDPOfferPrefix)
}

// IsSDPAnswer checks if a message is an SDP answer
func IsSDPAnswer(text string) bool {
	return strings.HasPrefix(text, SDPAnswerPrefix)
}

// IsICECandidate checks if a message is an ICE candidate
func IsICECandidate(text string) bool {
	return strings.HasPrefix(text, ICECandidatePrefix)
}

// ExtractSDP extracts the SDP payload from an SDP_OFFER: or SDP_ANSWER: message
func ExtractSDP(text string) string {
	if strings.HasPrefix(text, SDPOfferPrefix) {
		return text[len(SDPOfferPrefix):]
	}
	if strings.HasPrefix(text, SDPAnswerPrefix) {
		return text[len(SDPAnswerPrefix):]
	}
	return ""
}

// ExtractICECandidate extracts the ICE candidate JSON from an ICE: message
func ExtractICECandidate(text string) string {
	if strings.HasPrefix(text, ICECandidatePrefix) {
		return text[len(ICECandidatePrefix):]
	}
	return ""
}

// FormatSDPOffer wraps an SDP string as an offer message
func FormatSDPOffer(sdp string) string {
	return SDPOfferPrefix + sdp
}

// FormatSDPAnswer wraps an SDP string as an answer message
func FormatSDPAnswer(sdp string) string {
	return SDPAnswerPrefix + sdp
}

// FormatICECandidate wraps an ICE candidate JSON string
func FormatICECandidate(candidateJSON string) string {
	return ICECandidatePrefix + candidateJSON
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
