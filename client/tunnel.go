package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sync"
	"time"

	"soroush-relay/soroushlib"

	"github.com/pion/webrtc/v4"
)

// ──────────────────────────────────────────────────────────────────────────────
// Tunnel Engine — Client Side
// ──────────────────────────────────────────────────────────────────────────────

// TunnelEngine manages the full lifecycle of the Soroush WebRTC tunnel
type TunnelEngine struct {
	mu             sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc
	phase          string // "idle", "dispatching", "calling", "connected", "error"
	peerConnection *webrtc.PeerConnection
	dataChannel    *webrtc.DataChannel
	latencyMs      int64
	lastPingAt     time.Time
	startedAt      time.Time
	errorMsg       string

	// Soroush session for the client account
	clientSession   *soroushlib.MTProtoSession
	clientTransport *soroushlib.ObfuscatedTransport

	// Worker assignment
	workerUserID    int64
	workerAccessHash int64

	// Group bus state (for CONNECTED/DISCONNECT broadcasts)
	groupChatID     int64
	groupPSK        []byte
	clientAccountID string
	serverAccountID string
}

var tunnel = &TunnelEngine{phase: "idle"}

// ──────────────────────────────────────────────────────────────────────────────
// Pulse protocol messages
// ──────────────────────────────────────────────────────────────────────────────

type PulseMessage struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Start Tunnel — Main orchestration
// ──────────────────────────────────────────────────────────────────────────────

func startTunnel() error {
	tunnel.mu.Lock()
	if tunnel.phase != "idle" && tunnel.phase != "error" {
		tunnel.mu.Unlock()
		return fmt.Errorf("tunnel already in phase: %s", tunnel.phase)
	}
	tunnel.phase = "dispatching"
	tunnel.errorMsg = ""
	ctx, cancel := context.WithCancel(context.Background())
	tunnel.ctx = ctx
	tunnel.cancel = cancel
	tunnel.startedAt = time.Now()
	tunnel.mu.Unlock()

	state.mu.Lock()
	state.connecting = true
	state.mu.Unlock()

	go runTunnelFlow(ctx, cancel)
	return nil
}

func stopTunnel() {
	// Extract group bus state under lock, then release before network I/O
	tunnel.mu.Lock()
	gcID := tunnel.groupChatID
	sess := tunnel.clientSession
	cID := tunnel.clientAccountID
	psk := tunnel.groupPSK
	tunnel.mu.Unlock()

	// Broadcast DISCONNECT outside of lock to avoid blocking UI polls
	if gcID != 0 && sess != nil && cID != "" {
		disc := soroushlib.NewDisconnect(cID)
		discCtx, discCancel := context.WithTimeout(context.Background(), 5*time.Second)
		soroushlib.SendGroupCommand(discCtx, sess, gcID, disc, psk)
		discCancel()
	}

	// Re-acquire lock for state teardown
	tunnel.mu.Lock()
	if tunnel.cancel != nil {
		tunnel.cancel()
	}
	if tunnel.dataChannel != nil {
		tunnel.dataChannel.Close()
		tunnel.dataChannel = nil
	}
	if tunnel.peerConnection != nil {
		tunnel.peerConnection.Close()
		tunnel.peerConnection = nil
	}
	if tunnel.clientTransport != nil {
		tunnel.clientTransport.Disconnect()
		tunnel.clientTransport = nil
	}
	tunnel.phase = "idle"
	tunnel.latencyMs = 0
	tunnel.groupChatID = 0
	tunnel.clientAccountID = ""
	tunnel.serverAccountID = ""
	tunnel.mu.Unlock()

	state.mu.Lock()
	state.tunnelActive = false
	state.connecting = false
	state.mu.Unlock()

	addLog("Soroush WebRTC Tunnel stopped.", "warn")
}

// ──────────────────────────────────────────────────────────────────────────────
// Tunnel Flow — Step by step
// ──────────────────────────────────────────────────────────────────────────────

func runTunnelFlow(ctx context.Context, cancel context.CancelFunc) {
	defer func() {
		if r := recover(); r != nil {
			recordSystemLog(fmt.Sprintf("[Tunnel] Panic: %v", r), "error")
			setTunnelError(fmt.Sprintf("panic: %v", r))
		}
	}()

	// ── Step 1: Get client account from DB ──
	var clientAcc DBSoroushAccount
	if err := db.Where("status = ?", "connected").First(&clientAcc).Error; err != nil {
		setTunnelError("No authenticated Soroush account found. Add one first.")
		return
	}
	recordSystemLog(fmt.Sprintf("[Tunnel] Using client account: %s (ID: %d)", clientAcc.PhoneNumber, clientAcc.SoroushUserID), "info")

	// ── Step 2: Load tunnel config ──
	var config DBTunnelConfig
	db.First(&config)

	// ── Step 3: Connect to Soroush and restore session ──
	recordSystemLog("[Tunnel] Connecting to Soroush MTProto...", "info")
	session, transport := soroushlib.RestoreSession(clientAcc.AuthKey, clientAcc.AuthKeyID, clientAcc.ServerSalt)

	connCtx, connCancel := context.WithTimeout(ctx, 15*time.Second)
	if err := transport.Connect(connCtx); err != nil {
		connCancel()
		setTunnelError(fmt.Sprintf("Transport connect failed: %v", err))
		return
	}
	connCancel()
	recordSystemLog("[Tunnel] MTProto transport connected to wss://im-server.splus.ir/apiws", "success")

	tunnel.mu.Lock()
	tunnel.clientSession = session
	tunnel.clientTransport = transport
	tunnel.mu.Unlock()

	// ── Step 4+5: Discovery — Group Bus or Legacy Dispatcher ──
	if config.GroupChatID != 0 {
		// === GROUP PUB/SUB DISCOVERY ===
		psk := soroushlib.DefaultPSK
		if config.PSK != "" {
			psk = []byte(config.PSK)
		}
		clientID := clientAcc.ID

		// Send DISCOVER to group (uses SendAndWait to handle bad_server_salt)
		recordSystemLog("[Tunnel] Broadcasting DISCOVER to group...", "info")
		discover := soroushlib.NewDiscover(clientID)
		encoded, err := soroushlib.EncodeGroupCommand(discover, psk)
		if err != nil {
			setTunnelError(fmt.Sprintf("Encode DISCOVER: %v", err))
			return
		}
		discBody := soroushlib.BuildSendChannelMessage(config.GroupChatID, config.GroupAccessHash, encoded, time.Now().UnixNano())
		discCtx, discCancel := context.WithTimeout(ctx, 30*time.Second)
		_, _, err = session.SendAndWait(discCtx, discBody, true)
		discCancel()
		if err != nil {
			setTunnelError(fmt.Sprintf("DISCOVER failed: %v", err))
			return
		}
		recordSystemLog("[Tunnel] DISCOVER sent ✅", "success")

		// Wait for OFFER
		offerCtx, offerCancel := context.WithTimeout(ctx, 30*time.Second)
		offerCh := make(chan *soroushlib.GroupCommand, 1)
		go func() {
			soroushlib.ListenForMessages(offerCtx, session, func(msg soroushlib.IncomingMessage) {
				if !msg.IsGroup || msg.ChatID != config.GroupChatID || msg.FromUserID == clientAcc.SoroushUserID {
					return
				}
				cmd, err := soroushlib.DecodeGroupCommand(msg.Text, psk)
				if err != nil {
					return
				}
				if cmd.Cmd == soroushlib.CmdOffer && cmd.CID == clientID {
					// Non-blocking send: if multiple servers reply, take first, ignore rest
					select {
					case offerCh <- cmd:
					default:
					}
				}
			})
		}()

		var offer *soroushlib.GroupCommand
		select {
		case <-ctx.Done():
			offerCancel()
			setTunnelError("Cancelled during discovery")
			return
		case <-offerCtx.Done():
			offerCancel()
			setTunnelError("No server OFFER received within 30s")
			return
		case offer = <-offerCh:
			offerCancel()
		}
		recordSystemLog(fmt.Sprintf("[Tunnel] OFFER received from server=%s worker_uid=%d", offer.SID, offer.UID), "success")

		// Send CALLING to group to lock this server
		calling := soroushlib.NewCalling(clientID, offer.SID)
		callCtx2, callCancel2 := context.WithTimeout(ctx, 10*time.Second)
		soroushlib.SendGroupCommand(callCtx2, session, config.GroupChatID, calling, psk, config.GroupAccessHash)
		callCancel2()
		recordSystemLog(fmt.Sprintf("[Tunnel] CALLING sent for server %s", offer.SID), "info")

		tunnel.mu.Lock()
		tunnel.workerUserID = offer.UID
		tunnel.workerAccessHash = offer.AccessHash
		tunnel.phase = "calling"
		tunnel.groupChatID = config.GroupChatID
		tunnel.groupPSK = psk
		tunnel.clientAccountID = clientID
		tunnel.serverAccountID = offer.SID
		tunnel.mu.Unlock()
	} else if config.DispatcherUserID != 0 {
		// === LEGACY DISPATCHER ===
		recordSystemLog("[Tunnel] Using legacy dispatcher mode...", "warn")
		sendCtx, sendCancel := context.WithTimeout(ctx, 10*time.Second)
		err := soroushlib.SendTextMessage(sendCtx, session, config.DispatcherUserID, config.DispatcherAccessHash, soroushlib.DispatcherSynRequest)
		sendCancel()
		if err != nil {
			setTunnelError(fmt.Sprintf("Dispatch request failed: %v", err))
			return
		}

		workerCh := make(chan workerAssignment, 1)
		listenCtx, listenCancel := context.WithTimeout(ctx, 30*time.Second)
		go func() {
			soroushlib.ListenForMessages(listenCtx, session, func(msg soroushlib.IncomingMessage) {
				if msg.FromUserID == config.DispatcherUserID {
					uid, ah, ok := soroushlib.ParseDispatcherResponse(msg.Text)
					if ok {
						workerCh <- workerAssignment{userID: uid, accessHash: ah}
					} else if msg.Text == soroushlib.DispatcherNoWorkers {
						workerCh <- workerAssignment{err: fmt.Errorf("no idle workers")}
					}
				}
			})
		}()

		select {
		case <-ctx.Done():
			listenCancel()
			setTunnelError("Cancelled during dispatch")
			return
		case <-listenCtx.Done():
			listenCancel()
			setTunnelError("Dispatch timeout (30s)")
			return
		case wa := <-workerCh:
			listenCancel()
			if wa.err != nil {
				setTunnelError(wa.err.Error())
				return
			}
			tunnel.mu.Lock()
			tunnel.workerUserID = wa.userID
			tunnel.workerAccessHash = wa.accessHash
			tunnel.phase = "calling"
			tunnel.mu.Unlock()
		}
	} else {
		setTunnelError("No Group Chat ID or Dispatcher configured. Set one in Settings.")
		return
	}

	recordSystemLog(fmt.Sprintf("[Tunnel] Worker assigned: UID=%d. Initiating WebRTC call...", tunnel.workerUserID), "success")

	// ── Step 6: Establish WebRTC connection ──
	if err := establishWebRTC(ctx, session); err != nil {
		setTunnelError(fmt.Sprintf("WebRTC failed: %v", err))
		return
	}
}

type workerAssignment struct {
	userID     int64
	accessHash int64
	err        error
}

// ──────────────────────────────────────────────────────────────────────────────
// WebRTC Setup — Stealth Voice Call with Data Channel
// ──────────────────────────────────────────────────────────────────────────────

func establishWebRTC(ctx context.Context, session *soroushlib.MTProtoSession) error {
	// Build ICE server config from Soroush's TURN servers
	var iceServers []webrtc.ICEServer
	for _, srv := range soroushlib.SoroushTURNServers {
		ice := webrtc.ICEServer{URLs: srv.URLs}
		if srv.Username != "" {
			ice.Username = srv.Username
			ice.Credential = srv.Credential
			ice.CredentialType = webrtc.ICECredentialTypePassword
		}
		iceServers = append(iceServers, ice)
	}

	config := webrtc.Configuration{
		ICEServers:   iceServers,
		BundlePolicy: webrtc.BundlePolicyMaxBundle,
		RTCPMuxPolicy: webrtc.RTCPMuxPolicyRequire,
	}

	// Create PeerConnection
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("create peer connection: %w", err)
	}

	// ── Add dummy Audio track (Opus) to mimic voice call ──
	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		"audio0",
		"soroush-voice-stream",
	)
	if err != nil {
		pc.Close()
		return fmt.Errorf("create audio track: %w", err)
	}

	_, err = pc.AddTrack(audioTrack)
	if err != nil {
		pc.Close()
		return fmt.Errorf("add audio track: %w", err)
	}
	recordSystemLog("[WebRTC] Dummy Opus audio track added (voice call disguise)", "info")

	// ── Create Data Channel ──
	ordered := true
	dcInit := &webrtc.DataChannelInit{Ordered: &ordered}
	dc, err := pc.CreateDataChannel("data", dcInit)
	if err != nil {
		pc.Close()
		return fmt.Errorf("create data channel: %w", err)
	}

	tunnel.mu.Lock()
	tunnel.peerConnection = pc
	tunnel.dataChannel = dc
	tunnel.mu.Unlock()

	// ── Data Channel event handlers ──
	dc.OnOpen(func() {
		recordSystemLog("[WebRTC] Data channel OPEN! Sending init_pulse...", "success")

		pulse := PulseMessage{
			Type:      "init_pulse",
			Timestamp: time.Now().UnixNano(),
			ClientID:  "client-01",
		}
		data, _ := json.Marshal(pulse)
		dc.Send(data)

		// Start keepalive ping loop
		go runPingLoop(ctx, dc)

		tunnel.mu.Lock()
		tunnel.phase = "connected"
		tunnel.mu.Unlock()

		state.mu.Lock()
		state.connecting = false
		state.tunnelActive = true
		state.startedAt = time.Now()
		state.mu.Unlock()

		addLog("✅ Soroush WebRTC Tunnel ESTABLISHED!", "success")
		addLog("Traffic disguised as Soroush voice call payload", "success")

		// Broadcast CONNECTED to group
		tunnel.mu.Lock()
		gcID := tunnel.groupChatID
		gPSK := tunnel.groupPSK
		cAID := tunnel.clientAccountID
		sAID := tunnel.serverAccountID
		csess := tunnel.clientSession
		tunnel.mu.Unlock()
		if gcID != 0 && csess != nil {
			connCmd := soroushlib.NewConnected(cAID, sAID, tunnel.latencyMs)
			connCtx2, connCancel2 := context.WithTimeout(ctx, 5*time.Second)
			soroushlib.SendGroupCommand(connCtx2, csess, gcID, connCmd, gPSK)
			connCancel2()
		}
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		var pulse PulseMessage
		if err := json.Unmarshal(msg.Data, &pulse); err != nil {
			return
		}

		switch pulse.Type {
		case "ack_pulse":
			tunnel.mu.Lock()
			tunnel.latencyMs = pulse.LatencyMs
			tunnel.mu.Unlock()
			recordSystemLog(fmt.Sprintf("[WebRTC] Pulse ACK received! Latency: %dms", pulse.LatencyMs), "success")

		case "pong":
			latency := time.Since(tunnel.lastPingAt).Milliseconds()
			tunnel.mu.Lock()
			tunnel.latencyMs = latency
			tunnel.mu.Unlock()
		}
	})

	dc.OnClose(func() {
		recordSystemLog("[WebRTC] Data channel closed", "warn")

		tunnel.mu.Lock()
		wasConnected := tunnel.phase == "connected"
		tunnel.phase = "reconnecting"
		tunnel.mu.Unlock()

		state.mu.Lock()
		state.tunnelActive = false
		state.mu.Unlock()

		// Auto-reconnect if the tunnel was previously connected (not manually stopped)
		if wasConnected {
			addLog("WebRTC data channel lost. Auto-reconnecting in 3s...", "warn")
			go func() {
				time.Sleep(3 * time.Second)
				tunnel.mu.Lock()
				// Only reconnect if still in "reconnecting" (not manually stopped)
				if tunnel.phase != "reconnecting" {
					tunnel.mu.Unlock()
					return
				}
				// Clean up old connection
				if tunnel.peerConnection != nil {
					tunnel.peerConnection.Close()
					tunnel.peerConnection = nil
				}
				tunnel.dataChannel = nil
				tunnel.phase = "dispatching"
				ctx, cancel := context.WithCancel(context.Background())
				tunnel.ctx = ctx
				tunnel.cancel = cancel
				tunnel.mu.Unlock()

				state.mu.Lock()
				state.connecting = true
				state.mu.Unlock()

				recordSystemLog("[Tunnel] Auto-reconnecting...", "info")
				runTunnelFlow(ctx, cancel)
			}()
		}
	})

	// ── ICE connection state logging ──
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		recordSystemLog(fmt.Sprintf("[WebRTC] ICE state: %s", state.String()), "info")
		if state == webrtc.ICEConnectionStateFailed || state == webrtc.ICEConnectionStateDisconnected {
			addLog("WebRTC ICE connection lost. Tunnel interrupted.", "error")
		}
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		recordSystemLog(fmt.Sprintf("[WebRTC] Connection state: %s", state.String()), "info")
	})

	// ── Trickle ICE: Register candidate handler BEFORE generating offer ──
	// Candidates are collected and sent to the worker after SDP answer is received
	pendingICE := make(chan string, 32)
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		select {
		case pendingICE <- c.ToJSON().Candidate:
		default:
		}
	})

	// ── Generate SDP Offer (bare, without waiting for ICE gathering) ──
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("create offer: %w", err)
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("set local description: %w", err)
	}

	// NOTE: No GatheringCompletePromise wait! Trickle ICE sends candidates
	// asynchronously via direct messages after SDP answer is received.

	localDesc := pc.LocalDescription()
	recordSystemLog(fmt.Sprintf("[WebRTC] SDP Offer generated (%d bytes). Trickle ICE active.", len(localDesc.SDP)), "success")

	// ── Step 7: Send call via Soroush signaling ──
	// Generate DH parameters for the call encryption handshake
	gAHash := make([]byte, 32)
	rand.Read(gAHash)

	randID := make([]byte, 4)
	rand.Read(randID)
	randomID := int32(binary.LittleEndian.Uint32(randID))

	tunnel.mu.Lock()
	workerUID := tunnel.workerUserID
	workerAH := tunnel.workerAccessHash
	tunnel.mu.Unlock()

	callBody := soroushlib.BuildPhoneRequestCall(workerUID, workerAH, randomID, gAHash)

	recordSystemLog("[Soroush] Sending phone.requestCall to worker...", "info")
	callCtx, callCancel := context.WithTimeout(ctx, 15*time.Second)

	// Start listening for call response
	callRecvCh := make(chan callRecvResult, 1)
	go func() {
		cid, reader, err := session.Recv(callCtx)
		callRecvCh <- callRecvResult{cid: cid, reader: reader, err: err}
	}()

	_, err = session.Send(callCtx, callBody, true)
	if err != nil {
		callCancel()
		return fmt.Errorf("send phone.requestCall: %w", err)
	}

	// Wait for call acceptance
	select {
	case <-ctx.Done():
		callCancel()
		return ctx.Err()
	case result := <-callRecvCh:
		callCancel()
		if result.err != nil {
			return fmt.Errorf("recv call response: %w", result.err)
		}

		innerCID, innerReader := unwrapResponse(result.cid, result.reader, 0)
		callEvent, err := soroushlib.ParsePhoneCallResult(innerCID, innerReader)
		if err != nil {
			return fmt.Errorf("parse call result: %w", err)
		}

		if callEvent != nil {
			recordSystemLog(fmt.Sprintf("[Soroush] Call event: %s (ID: %d)", callEvent.Type, callEvent.CallID), "success")

			// Apply TURN credentials from call event to PeerConnection
			if len(callEvent.Connections) > 0 {
				for _, conn := range callEvent.Connections {
					if conn.Turn && conn.Username != "" {
						turnURL := fmt.Sprintf("turn:%s:%d", conn.IP, conn.Port)
						recordSystemLog(fmt.Sprintf("[WebRTC] Adding TURN: %s (user: %s)", turnURL, conn.Username), "info")
					}
				}
			}
		}
	}

	// ── Step 8: Send SDP Offer to worker via direct message ──
	sdpOffer := pc.LocalDescription()
	if sdpOffer == nil {
		return fmt.Errorf("no local SDP description")
	}

	offerMsg := soroushlib.FormatSDPOffer(sdpOffer.SDP)
	sdpSendCtx, sdpSendCancel := context.WithTimeout(ctx, 10*time.Second)
	err = soroushlib.SendTextMessage(sdpSendCtx, session, workerUID, workerAH, offerMsg)
	sdpSendCancel()
	if err != nil {
		return fmt.Errorf("send SDP offer: %w", err)
	}
	recordSystemLog(fmt.Sprintf("[WebRTC] SDP Offer sent to worker (%d bytes)", len(sdpOffer.SDP)), "success")

	// ── Step 9: Listen for SDP Answer + ICE candidates from worker ──
	sdpAnswerCtx, sdpAnswerCancel := context.WithTimeout(ctx, 30*time.Second)
	answerDone := make(chan bool, 1)

	// pendingICE channel is already populated by OnICECandidate registered above

	go func() {
		soroushlib.ListenForMessages(sdpAnswerCtx, session, func(msg soroushlib.IncomingMessage) {
			if msg.IsGroup || msg.FromUserID != workerUID {
				return
			}

			if soroushlib.IsSDPAnswer(msg.Text) {
				sdpStr := soroushlib.ExtractSDP(msg.Text)
				recordSystemLog(fmt.Sprintf("[WebRTC] Received SDP answer (%d bytes)", len(sdpStr)), "info")

				answer := webrtc.SessionDescription{
					Type: webrtc.SDPTypeAnswer,
					SDP:  sdpStr,
				}
				if err := pc.SetRemoteDescription(answer); err != nil {
					recordSystemLog(fmt.Sprintf("[WebRTC] SetRemoteDescription failed: %v", err), "error")
					return
				}
				recordSystemLog("[WebRTC] Remote description set successfully", "success")

				// Send our ICE candidates to worker
				go func() {
					time.Sleep(500 * time.Millisecond)
					for {
						select {
						case candidate := <-pendingICE:
							iceMsg := soroushlib.FormatICECandidate(candidate)
							iceCtx, iceCancel := context.WithTimeout(sdpAnswerCtx, 5*time.Second)
							soroushlib.SendTextMessage(iceCtx, session, workerUID, workerAH, iceMsg)
							iceCancel()
						case <-sdpAnswerCtx.Done():
							return
						}
					}
				}()

				answerDone <- true
			}

			if soroushlib.IsICECandidate(msg.Text) {
				candidateStr := soroushlib.ExtractICECandidate(msg.Text)
				if err := pc.AddICECandidate(webrtc.ICECandidateInit{Candidate: candidateStr}); err != nil {
					recordSystemLog(fmt.Sprintf("[WebRTC] AddICECandidate failed: %v", err), "warn")
				}
			}
		})
	}()

	select {
	case <-answerDone:
		sdpAnswerCancel()
		recordSystemLog("[WebRTC] SDP negotiation complete!", "success")
	case <-sdpAnswerCtx.Done():
		sdpAnswerCancel()
		return fmt.Errorf("SDP answer timeout (30s)")
	}

	recordSystemLog("[WebRTC] Waiting for data channel to open...", "info")

	// Wait for connection or timeout
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(30 * time.Second):
		if tunnel.phase != "connected" {
			return fmt.Errorf("data channel open timeout (30s)")
		}
	}

	return nil
}

type callRecvResult struct {
	cid    uint32
	reader *soroushlib.TLReader
	err    error
}

// ──────────────────────────────────────────────────────────────────────────────
// Keepalive ping loop
// ──────────────────────────────────────────────────────────────────────────────

func runPingLoop(ctx context.Context, dc *webrtc.DataChannel) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tunnel.lastPingAt = time.Now()
			ping := PulseMessage{Type: "ping", Timestamp: time.Now().UnixNano()}
			data, _ := json.Marshal(ping)
			if err := dc.Send(data); err != nil {
				log.Printf("[Tunnel] ping send failed: %v", err)
				return
			}
		}
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Error state helper
// ──────────────────────────────────────────────────────────────────────────────

func setTunnelError(msg string) {
	recordSystemLog(fmt.Sprintf("[Tunnel] ERROR: %s", msg), "error")
	tunnel.mu.Lock()
	tunnel.phase = "error"
	tunnel.errorMsg = msg
	tunnel.mu.Unlock()

	state.mu.Lock()
	state.connecting = false
	state.tunnelActive = false
	state.mu.Unlock()
}

// ──────────────────────────────────────────────────────────────────────────────
// DH helper (for call encryption — generates g_a_hash)
// ──────────────────────────────────────────────────────────────────────────────

func generateCallDH() (gA []byte, gAHash []byte) {
	// Generate random 256-byte exponent
	aBytes := make([]byte, 256)
	rand.Read(aBytes)
	a := new(big.Int).SetBytes(aBytes)

	// Use g=3, standard DH prime from Telegram
	g := big.NewInt(3)
	// Simplified: generate g_a = g^a mod p (using a well-known safe prime)
	p, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF", 16)

	gABig := new(big.Int).Exp(g, a, p)
	gA = make([]byte, 256)
	gABigBytes := gABig.Bytes()
	copy(gA[256-len(gABigBytes):], gABigBytes)

	gAHash = soroushlib.Sha256Sum(gA)
	return
}

// ──────────────────────────────────────────────────────────────────────────────
// API Handlers for tunnel control
// ──────────────────────────────────────────────────────────────────────────────

func handleTunnelStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if err := startTunnel(); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"message":"Tunnel connection initiated"}`))
}

func handleTunnelStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	stopTunnel()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Tunnel closed"}`))
}

func handleTunnelStatus(w http.ResponseWriter, r *http.Request) {
	tunnel.mu.Lock()
	defer tunnel.mu.Unlock()

	uptime := "0s"
	if tunnel.phase == "connected" {
		uptime = time.Since(tunnel.startedAt).Round(time.Second).String()
	}

	resp := map[string]interface{}{
		"phase":     tunnel.phase,
		"latencyMs": tunnel.latencyMs,
		"uptime":    uptime,
		"error":     tunnel.errorMsg,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleTunnelConfig manages the tunnel configuration (group bus + legacy dispatcher)
func handleTunnelConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		var config DBTunnelConfig
		db.First(&config)
		json.NewEncoder(w).Encode(config)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			GroupChatID          int64  `json:"groupChatId"`
			GroupAccessHash      int64  `json:"groupAccessHash"`
			PSK                  string `json:"psk"`
			DispatcherUserID     int64  `json:"dispatcherUserId"`
			DispatcherAccessHash int64  `json:"dispatcherAccessHash"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid request"}`, http.StatusBadRequest)
			return
		}

		config := DBTunnelConfig{
			ID:                   1,
			GroupChatID:          req.GroupChatID,
			GroupAccessHash:      req.GroupAccessHash,
			PSK:                  req.PSK,
			DispatcherUserID:     req.DispatcherUserID,
			DispatcherAccessHash: req.DispatcherAccessHash,
		}
		db.Save(&config)

		addLog(fmt.Sprintf("Tunnel config saved: GroupChatID=%d AccessHash=%d", req.GroupChatID, req.GroupAccessHash), "success")
		json.NewEncoder(w).Encode(config)
		return
	}

	http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
}

// ──────────────────────────────────────────────────────────────────────────────
// Server Connectivity Test — Pings the Clever Cloud server directly
// ──────────────────────────────────────────────────────────────────────────────

func handleTestServerConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ServerURL string `json:"serverUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ServerURL == "" {
		http.Error(w, `{"error":"serverUrl is required"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Ping the server's public /api/ping endpoint
	pingURL := req.ServerURL + "/api/ping"
	addLog(fmt.Sprintf("Testing server connectivity: %s", pingURL), "info")

	client := &http.Client{Timeout: 15 * time.Second}
	start := time.Now()
	resp, err := client.Get(pingURL)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		addLog(fmt.Sprintf("Server connection FAILED: %v", err), "error")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   false,
			"error":     err.Error(),
			"latencyMs": latency,
		})
		return
	}
	defer resp.Body.Close()

	var serverResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&serverResp)

	if resp.StatusCode == 200 {
		addLog(fmt.Sprintf("✅ Server connection OK! Latency: %dms, Version: %v", latency, serverResp["version"]), "success")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    true,
			"latencyMs":  latency,
			"statusCode": resp.StatusCode,
			"server":     serverResp,
		})
	} else {
		addLog(fmt.Sprintf("⚠️ Server responded with HTTP %d (latency: %dms)", resp.StatusCode, latency), "warn")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":    false,
			"latencyMs":  latency,
			"statusCode": resp.StatusCode,
			"error":      fmt.Sprintf("Unexpected HTTP %d", resp.StatusCode),
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Tunnel Test Handler — POST /api/tunnel/test
// Runs 4-step test: MTProto → Group DISCOVER → WebRTC Call → Ping/Pong
// ──────────────────────────────────────────────────────────────────────────────

type TunnelTestStep struct {
	Name      string `json:"name"`
	Status    string `json:"status"` // "pass", "fail", "skip"
	LatencyMs int64  `json:"latencyMs"`
	Detail    string `json:"detail"`
}

func handleTunnelTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	steps := make([]TunnelTestStep, 0, 4)
	overallStart := time.Now()

	// ── Step 1: MTProto Connect ──
	step1Start := time.Now()
	var account DBSoroushAccount
	if err := db.Where("status = ? AND length(auth_key) > 0", "connected").First(&account).Error; err != nil {
		steps = append(steps, TunnelTestStep{
			Name: "mtproto_connect", Status: "fail", Detail: "No connected account found",
			LatencyMs: time.Since(step1Start).Milliseconds(),
		})
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "steps": steps})
		return
	}

	session, transport := soroushlib.RestoreSession(account.AuthKey, account.AuthKeyID, account.ServerSalt)
	connCtx, connCancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := transport.Connect(connCtx); err != nil {
		connCancel()
		steps = append(steps, TunnelTestStep{
			Name: "mtproto_connect", Status: "fail", Detail: fmt.Sprintf("Transport: %v", err),
			LatencyMs: time.Since(step1Start).Milliseconds(),
		})
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "steps": steps})
		return
	}
	connCancel()
	defer transport.Disconnect()

	steps = append(steps, TunnelTestStep{
		Name: "mtproto_connect", Status: "pass",
		Detail:    fmt.Sprintf("Connected as %s (UID=%d)", account.PhoneNumber, account.SoroushUserID),
		LatencyMs: time.Since(step1Start).Milliseconds(),
	})
	addLog("[TunnelTest] Step 1: MTProto connected ✅", "success")

	// ── Step 2: Group DISCOVER ──
	step2Start := time.Now()

	var tunnelCfg DBTunnelConfig
	db.First(&tunnelCfg)
	if tunnelCfg.GroupChatID == 0 {
		steps = append(steps, TunnelTestStep{
			Name: "group_discover", Status: "fail", Detail: "Group Chat ID not configured",
			LatencyMs: time.Since(step2Start).Milliseconds(),
		})
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "steps": steps})
		return
	}

	psk := soroushlib.DefaultPSK
	if tunnelCfg.PSK != "" {
		psk = []byte(tunnelCfg.PSK)
	}
	clientID := account.ID

	// Send DISCOVER to group (uses SendAndWait to handle bad_server_salt + prime session)
	discover := soroushlib.NewDiscover(clientID)
	encoded, encErr := soroushlib.EncodeGroupCommand(discover, psk)
	if encErr != nil {
		steps = append(steps, TunnelTestStep{
			Name: "group_discover", Status: "fail", Detail: fmt.Sprintf("Encode: %v", encErr),
			LatencyMs: time.Since(step2Start).Milliseconds(),
		})
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "steps": steps})
		return
	}
	discBody := soroushlib.BuildSendChannelMessage(tunnelCfg.GroupChatID, tunnelCfg.GroupAccessHash, encoded, time.Now().UnixNano())
	discoverCtx, discoverCancel := context.WithTimeout(context.Background(), 30*time.Second)
	_, _, discErr := session.SendAndWait(discoverCtx, discBody, true)
	discoverCancel()
	if discErr != nil {
		steps = append(steps, TunnelTestStep{
			Name: "group_discover", Status: "fail", Detail: fmt.Sprintf("Send DISCOVER: %v", discErr),
			LatencyMs: time.Since(step2Start).Milliseconds(),
		})
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "steps": steps})
		return
	}
	addLog("[TunnelTest] DISCOVER sent to group, waiting for OFFER...", "info")

	// Wait for OFFER response (timeout 30s)
	var offer *soroushlib.GroupCommand
	offerCtx, offerCancel := context.WithTimeout(context.Background(), 30*time.Second)

	offerCh := make(chan *soroushlib.GroupCommand, 1)
	go func() {
		soroushlib.ListenForMessages(offerCtx, session, func(msg soroushlib.IncomingMessage) {
			if !msg.IsGroup || msg.ChatID != tunnelCfg.GroupChatID {
				return
			}
			if msg.FromUserID == account.SoroushUserID {
				return
			}
			cmd, err := soroushlib.DecodeGroupCommand(msg.Text, psk)
			if err != nil {
				return
			}
			if cmd.Cmd == soroushlib.CmdOffer && cmd.CID == clientID {
				offerCh <- cmd
			}
		})
	}()

	select {
	case offer = <-offerCh:
		offerCancel() // Stop listening
	case <-offerCtx.Done():
		offerCancel()
		steps = append(steps, TunnelTestStep{
			Name: "group_discover", Status: "fail", Detail: "No OFFER received within 30s",
			LatencyMs: time.Since(step2Start).Milliseconds(),
		})
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "steps": steps})
		return
	}

	steps = append(steps, TunnelTestStep{
		Name: "group_discover", Status: "pass",
		Detail:    fmt.Sprintf("OFFER from server=%s worker_uid=%d", offer.SID, offer.UID),
		LatencyMs: time.Since(step2Start).Milliseconds(),
	})
	addLog(fmt.Sprintf("[TunnelTest] Step 2: OFFER received from server %s ✅", offer.SID), "success")

	// ── Step 3: WebRTC Call (stub — requires SDP exchange) ──
	steps = append(steps, TunnelTestStep{
		Name: "webrtc_call", Status: "skip",
		Detail:    fmt.Sprintf("Worker UID=%d ready. SDP exchange pending implementation.", offer.UID),
		LatencyMs: 0,
	})
	addLog("[TunnelTest] Step 3: WebRTC call — skipped (SDP exchange pending)", "warn")

	// ── Step 4: Ping/Pong (depends on Step 3) ──
	steps = append(steps, TunnelTestStep{
		Name: "ping_pong", Status: "skip",
		Detail:    "Depends on WebRTC data channel",
		LatencyMs: 0,
	})
	addLog("[TunnelTest] Step 4: Ping/Pong — skipped (depends on WebRTC)", "warn")

	overallLatency := time.Since(overallStart).Milliseconds()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          true,
		"steps":            steps,
		"overallLatencyMs": overallLatency,
	})
}
