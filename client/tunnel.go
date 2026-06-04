package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"

	"soroush-relay/soroushlib"

	"github.com/hashicorp/yamux"
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
	yamuxSession   *yamux.Session
	socksListener  net.Listener
	latencyMs      int64
	lastPingAt     time.Time
	startedAt      time.Time
	errorMsg       string

	// Soroush session for the client account
	clientSession   *soroushlib.MTProtoSession
	clientTransport *soroushlib.ObfuscatedTransport
	clientRouter    *soroushlib.MessageRouter

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
	if tunnel.socksListener != nil {
		tunnel.socksListener.Close()
		tunnel.socksListener = nil
	}
	if tunnel.yamuxSession != nil {
		tunnel.yamuxSession.Close()
		tunnel.yamuxSession = nil
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
	tunnel.clientRouter = nil
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
	session.Logger = func(msg string, level string) {
		recordSystemLog(msg, level)
	}

	connCtx, connCancel := context.WithTimeout(ctx, 15*time.Second)
	if err := transport.Connect(connCtx); err != nil {
		connCancel()
		setTunnelError(fmt.Sprintf("Transport connect failed: %v", err))
		return
	}
	connCancel()
	recordSystemLog("[Tunnel] MTProto transport connected to wss://im-server.splus.ir/apiws", "success")

	// Start MessageRouter
	router := soroushlib.NewMessageRouter(session)
	go func() {
		if err := router.Run(ctx); err != nil {
			if ctx.Err() == nil {
				recordSystemLog(fmt.Sprintf("[Tunnel] MessageRouter error: %v", err), "error")
				setTunnelError(fmt.Sprintf("MessageRouter error: %v", err))
			}
		}
	}()

	// Start keepalive ping ticker for client Soroush WebSocket
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pingBody := soroushlib.BuildPingDelayDisconnectRequest(time.Now().UnixNano(), 75)
				session.Send(ctx, pingBody, true)
			}
		}
	}()

	tunnel.mu.Lock()
	tunnel.clientSession = session
	tunnel.clientTransport = transport
	tunnel.clientRouter = router
	tunnel.mu.Unlock()

	// ── Step 4+5: Discovery — Group Bus or Legacy Dispatcher ──
	if config.GroupChatID != 0 {
		// === GROUP PUB/SUB DISCOVERY ===
		psk := soroushlib.DefaultPSK
		if config.PSK != "" {
			psk = []byte(config.PSK)
		}
		clientID := clientAcc.ID

		// Send DISCOVER to group (wrapped in initConnection, uses SendAndWait for salt handling)
		recordSystemLog("[Tunnel] Broadcasting DISCOVER to group...", "info")
		discover := soroushlib.NewDiscover(clientID)
		encoded, err := soroushlib.EncodeGroupCommand(discover, psk)
		if err != nil {
			setTunnelError(fmt.Sprintf("Encode DISCOVER: %v", err))
			return
		}
		discBody := soroushlib.BuildSendChannelMessage(config.GroupChatID, config.GroupAccessHash, encoded, time.Now().UnixNano())
		wrappedDisc := soroushlib.WrapInitConnection(soroushlib.SoroushAppID, discBody)

		offerDone := make(chan bool)
		var offer *soroushlib.GroupCommand

		// Start a background sender that retries DISCOVER every 5 seconds until OFFER is received
		discoverCtx, discoverCancel := context.WithCancel(ctx)
		defer discoverCancel()
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				discCtx, discCancel := context.WithTimeout(discoverCtx, 10*time.Second)
				_, _, err := session.SendAndWait(discCtx, wrappedDisc, true)
				discCancel()
				if err != nil {
					recordSystemLog(fmt.Sprintf("[Tunnel] Retrying DISCOVER failed: %v", err), "warn")
				} else {
					recordSystemLog("[Tunnel] DISCOVER sent ✅", "success")
				}

				select {
				case <-discoverCtx.Done():
					return
				case <-ticker.C:
				}
			}
		}()

		// Wait for OFFER using text subscription
		offerCtx, offerCancel := context.WithTimeout(ctx, 30*time.Second)
		defer offerCancel()
		textSub := router.SubscribeText()
		defer router.UnsubscribeText(textSub)

		go func() {
			for msg := range textSub {
				if !msg.IsGroup || msg.ChatID != config.GroupChatID || msg.FromUserID == clientAcc.SoroushUserID {
					continue
				}
				cmd, err := soroushlib.DecodeGroupCommand(msg.Text, psk)
				if err != nil {
					continue
				}
				if cmd.Cmd == soroushlib.CmdOffer && cmd.CID == clientID {
					offer = cmd
					discoverCancel() // Stop broadcasting DISCOVER
					select {
					case offerDone <- true:
					default:
					}
					return
				}
			}
		}()

		select {
		case <-ctx.Done():
			setTunnelError("Cancelled during discovery")
			return
		case <-offerCtx.Done():
			setTunnelError("No server OFFER received within 30s")
			return
		case <-offerDone:
			// Offer received
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
		defer listenCancel()
		textSub := router.SubscribeText()
		defer router.UnsubscribeText(textSub)

		go func() {
			for msg := range textSub {
				if msg.FromUserID == config.DispatcherUserID {
					uid, ah, ok := soroushlib.ParseDispatcherResponse(msg.Text)
					if ok {
						select {
						case workerCh <- workerAssignment{userID: uid, accessHash: ah}:
						default:
						}
						return
					} else if msg.Text == soroushlib.DispatcherNoWorkers {
						select {
						case workerCh <- workerAssignment{err: fmt.Errorf("no idle workers")}:
						default:
						}
						return
					}
				}
			}
		}()

		select {
		case <-ctx.Done():
			setTunnelError("Cancelled during dispatch")
			return
		case <-listenCtx.Done():
			setTunnelError("Dispatch timeout (30s)")
			return
		case wa := <-workerCh:
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
	if err := establishWebRTC(ctx, session, router, &config); err != nil {
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

func establishWebRTC(ctx context.Context, session *soroushlib.MTProtoSession, router *soroushlib.MessageRouter, config *DBTunnelConfig) error {
	tunnel.mu.Lock()
	clientID := tunnel.clientAccountID
	serverID := tunnel.serverAccountID
	groupChatID := tunnel.groupChatID
	psk := tunnel.groupPSK
	workerUID := tunnel.workerUserID
	workerAH := tunnel.workerAccessHash
	tunnel.mu.Unlock()

	// ── Step 7: Complete Soroush Call Setup ──
	a, gA, gAHash := generateClientDH()

	randID := make([]byte, 4)
	rand.Read(randID)
	randomID := int32(binary.LittleEndian.Uint32(randID))

	callBody := soroushlib.BuildPhoneRequestCall(workerUID, workerAH, randomID, gAHash)
	recordSystemLog("[Soroush] Sending phone.requestCall to worker...", "info")

	// Subscribe to raw updates BEFORE sending the call request to avoid missing
	// fast-pushed phoneCallAccepted/confirmed updates (fixes race condition)
	updateSub := router.SubscribeUpdate()
	defer router.UnsubscribeUpdate(updateSub)

	// Call request via SendAndWait to get initial CallID and AccessHash
	callReqCtx, callReqCancel := context.WithTimeout(ctx, 15*time.Second)
	cid, r, err := session.SendAndWait(callReqCtx, callBody, true)
	callReqCancel()
	if err != nil {
		return fmt.Errorf("send phone.requestCall: %w", err)
	}

	innerCID, innerReader := unwrapResponse(cid, r, 0)
	initialCallEvent, err := soroushlib.ParsePhoneCallResult(innerCID, innerReader)
	if err != nil {
		return fmt.Errorf("parse requestCall result: %w", err)
	}
	if initialCallEvent == nil || initialCallEvent.CallID == 0 {
		return fmt.Errorf("invalid call event returned from requestCall")
	}

	callID := initialCallEvent.CallID
	callAccessHash := initialCallEvent.AccessHash
	recordSystemLog(fmt.Sprintf("[Soroush] Call requested. CallID=%d, AccessHash=%d", callID, callAccessHash), "info")

	recordSystemLog("[Soroush] Waiting for call to be accepted by worker...", "info")
	acceptCtx, acceptCancel := context.WithTimeout(ctx, 30*time.Second)
	defer acceptCancel()

	var gb []byte
	acceptedDone := make(chan bool, 1)

	go func() {
		for {
			select {
			case <-acceptCtx.Done():
				return
			case msg, ok := <-updateSub:
				if !ok {
					return
				}
				innerCID, innerReader := unwrapResponse(msg.CID, soroushlib.NewTLReader(msg.Data), 0)
				if innerCID == soroushlib.IDUpdatePhoneCall {
					callEvent, err := soroushlib.ParseCallUpdate(innerReader)
					if err != nil || callEvent == nil {
						continue
					}
					if callEvent.CallID == callID && callEvent.Type == "accepted" {
						gb = callEvent.GB
						select {
						case acceptedDone <- true:
						default:
						}
						return
					}
				}
			}
		}
	}()

	select {
	case <-acceptedDone:
		recordSystemLog("[Soroush] Call accepted by worker. Computing E2E key...", "success")
	case <-acceptCtx.Done():
		return fmt.Errorf("timeout waiting for call acceptance")
	}

	// Compute key fingerprint and confirm the call
	fingerprint := computeFingerprint(gb, a)
	confirmBody := soroushlib.BuildPhoneConfirmCall(callID, callAccessHash, gA, fingerprint)
	recordSystemLog("[Soroush] Sending phone.confirmCall...", "info")

	confirmCtx, confirmCancel := context.WithTimeout(ctx, 15*time.Second)
	_, _, err = session.SendAndWait(confirmCtx, confirmBody, true)
	confirmCancel()
	if err != nil {
		return fmt.Errorf("send phone.confirmCall: %w", err)
	}

	// Wait for call confirmation and TURN servers
	recordSystemLog("[Soroush] Waiting for call confirmation and TURN servers...", "info")
	confirmWaitCtx, confirmWaitCancel := context.WithTimeout(ctx, 30*time.Second)
	defer confirmWaitCancel()

	var connections []soroushlib.PhoneConnectionInfo
	confirmedDone := make(chan bool, 1)

	go func() {
		for {
			select {
			case <-confirmWaitCtx.Done():
				return
			case msg, ok := <-updateSub:
				if !ok {
					return
				}
				innerCID, innerReader := unwrapResponse(msg.CID, soroushlib.NewTLReader(msg.Data), 0)
				if innerCID == soroushlib.IDUpdatePhoneCall {
					callEvent, err := soroushlib.ParseCallUpdate(innerReader)
					if err != nil || callEvent == nil {
						continue
					}
					if callEvent.CallID == callID && (callEvent.Type == "confirmed" || len(callEvent.Connections) > 0) {
						connections = callEvent.Connections
						select {
						case confirmedDone <- true:
						default:
						}
						return
					}
				}
			}
		}
	}()

	select {
	case <-confirmedDone:
		recordSystemLog(fmt.Sprintf("[Soroush] Call confirmed. Received %d connection endpoints.", len(connections)), "success")
	case <-confirmWaitCtx.Done():
		return fmt.Errorf("timeout waiting for call confirmation")
	}

	// Build ICE server config from Soroush's dynamic TURN servers
	var iceServers []webrtc.ICEServer
	for _, conn := range connections {
		if conn.Turn && conn.Username != "" {
			turnTCP := fmt.Sprintf("turn:%s:%d?transport=tcp", conn.IP, conn.Port)
			iceServers = append(iceServers, webrtc.ICEServer{
				URLs:           []string{turnTCP},
				Username:       conn.Username,
				Credential:     conn.Password,
				CredentialType: webrtc.ICECredentialTypePassword,
			})
			turnUDP := fmt.Sprintf("turn:%s:%d", conn.IP, conn.Port)
			iceServers = append(iceServers, webrtc.ICEServer{
				URLs:           []string{turnUDP},
				Username:       conn.Username,
				Credential:     conn.Password,
				CredentialType: webrtc.ICECredentialTypePassword,
			})
			recordSystemLog(fmt.Sprintf("[WebRTC] Dynamic TURN: %s:%d (user: %s)", conn.IP, conn.Port, conn.Username), "info")
		} else if conn.Stun {
			stunURL := fmt.Sprintf("stun:%s:%d", conn.IP, conn.Port)
			iceServers = append(iceServers, webrtc.ICEServer{
				URLs: []string{stunURL},
			})
		}
	}

	// Fallback to static SoroushTURNServers if no TURN server with credentials was found
	if len(iceServers) == 0 {
		recordSystemLog("[WebRTC] No dynamic TURN credentials received. Falling back to static configuration.", "warn")
		for _, srv := range soroushlib.SoroushTURNServers {
			ice := webrtc.ICEServer{URLs: srv.URLs}
			if srv.Username != "" {
				ice.Username = srv.Username
				ice.Credential = srv.Credential
				ice.CredentialType = webrtc.ICECredentialTypePassword
			}
			iceServers = append(iceServers, ice)
		}
	}

	configPC := webrtc.Configuration{
		ICEServers:         iceServers,
		BundlePolicy:       webrtc.BundlePolicyMaxBundle,
		RTCPMuxPolicy:      webrtc.RTCPMuxPolicyRequire,
		ICETransportPolicy: webrtc.ICETransportPolicyRelay, // Force TURN relay for censor bypass
	}

	// Create PeerConnection
	pc, err := webrtc.NewPeerConnection(configPC)
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
	negotiated := true
	var dcID uint16 = 0
	dc, err := pc.CreateDataChannel("data", &webrtc.DataChannelInit{
		Ordered:    &ordered,
		Negotiated: &negotiated,
		ID:         &dcID,
	})
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
		recordSystemLog("[WebRTC] Data channel OPEN!", "success")

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
			soroushlib.SendGroupCommand(connCtx2, csess, gcID, connCmd, gPSK, config.GroupAccessHash)
			connCancel2()
		}

		// Start local SOCKS5 proxy over yamux
		go startLocalSOCKS5Proxy(ctx, dc)
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

	// ── Create SDP Offer ──
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		pc.Close()
		return fmt.Errorf("create offer: %w", err)
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		pc.Close()
		return fmt.Errorf("set local description: %w", err)
	}

	// Wait for ICE gathering to complete to embed relay candidates in SDP Offer
	recordSystemLog("[WebRTC] Waiting for ICE gathering to complete...", "info")
	gatherDone := webrtc.GatheringCompletePromise(pc)
	select {
	case <-gatherDone:
		recordSystemLog("[WebRTC] ICE gathering complete. Relay candidates embedded.", "success")
	case <-time.After(10 * time.Second):
		recordSystemLog("[WebRTC] ICE gathering timeout (10s), using partial SDP...", "warn")
	}

	localDesc := pc.LocalDescription()
	if localDesc == nil {
		pc.Close()
		return fmt.Errorf("no local SDP description after gathering")
	}

	// ── Step 8: Send SDP Offer to worker ──
	if groupChatID != 0 {
		// Send chunked SDP offer
		chunks := soroushlib.ChunkString(localDesc.SDP, 1500)
		for i, chunk := range chunks {
			var cmd *soroushlib.GroupCommand
			if i < len(chunks)-1 {
				cmd = &soroushlib.GroupCommand{
					Version:    1,
					Cmd:        soroushlib.CmdSDPOfferChunk,
					CID:        clientID,
					SID:        serverID,
					Data:       chunk,
					ChunkIdx:   i,
					ChunkTotal: len(chunks),
					Timestamp:  time.Now().UnixMilli(),
				}
			} else {
				cmd = soroushlib.NewSDPOffer(clientID, serverID, chunk)
				cmd.ChunkIdx = i
				cmd.ChunkTotal = len(chunks)
			}
			sdpSendCtx, sdpSendCancel := context.WithTimeout(ctx, 10*time.Second)
			err = soroushlib.SendGroupCommand(sdpSendCtx, session, groupChatID, cmd, psk, config.GroupAccessHash)
			sdpSendCancel()
			if err != nil {
				pc.Close()
				return fmt.Errorf("send SDP offer chunk %d/%d: %w", i+1, len(chunks), err)
			}
			time.Sleep(150 * time.Millisecond) // rate limiting
		}
		recordSystemLog(fmt.Sprintf("[WebRTC] Chunked SDP Offer sent to Group Bus (%d bytes, %d chunks)", len(localDesc.SDP), len(chunks)), "success")
	} else {
		// Fallback for legacy dispatcher (DMs)
		offerMsg := soroushlib.FormatSDPOffer(localDesc.SDP)
		sdpSendCtx, sdpSendCancel := context.WithTimeout(ctx, 10*time.Second)
		err = soroushlib.SendTextMessage(sdpSendCtx, session, workerUID, workerAH, offerMsg)
		sdpSendCancel()
		if err != nil {
			pc.Close()
			return fmt.Errorf("send SDP offer: %w", err)
		}
		recordSystemLog(fmt.Sprintf("[WebRTC] SDP Offer sent to worker via DM (%d bytes)", len(localDesc.SDP)), "success")
	}

	// ── Step 9: Listen for SDP Answer ──
	sdpAnswerCtx, sdpAnswerCancel := context.WithTimeout(ctx, 45*time.Second)
	defer sdpAnswerCancel()
	answerDone := make(chan string, 1)

	textSub := router.SubscribeText()
	defer router.UnsubscribeText(textSub)

	assembler := soroushlib.NewSDPAssembler()

	go func() {
		for {
			select {
			case <-sdpAnswerCtx.Done():
				return
			case msg, ok := <-textSub:
				if !ok {
					return
				}
				if groupChatID != 0 {
					if !msg.IsGroup || msg.ChatID != groupChatID {
						continue
					}
					cmd, err := soroushlib.DecodeGroupCommand(msg.Text, psk)
					if err != nil {
						continue
					}
					if cmd.CID != clientID || cmd.SID != serverID {
						continue
					}

					if cmd.Cmd == soroushlib.CmdSDPAnswerChunk {
						if fullSDP, complete := assembler.AddChunk(cmd.ChunkIdx, cmd.ChunkTotal, cmd.Data); complete {
							select {
							case answerDone <- fullSDP:
							default:
							}
						}
					} else if cmd.Cmd == soroushlib.CmdSDPAnswer {
						var complete bool
						var fullSDP string
						if cmd.ChunkTotal > 1 {
							fullSDP, complete = assembler.AddChunk(cmd.ChunkIdx, cmd.ChunkTotal, cmd.Data)
						} else {
							fullSDP = cmd.Data
							complete = true
						}
						if complete {
							select {
							case answerDone <- fullSDP:
							default:
							}
						}
					}
				} else {
					// Legacy DM mode
					if msg.IsGroup || msg.FromUserID != workerUID {
						continue
					}
					if soroushlib.IsSDPAnswer(msg.Text) {
						sdpStr := soroushlib.ExtractSDP(msg.Text)
						select {
						case answerDone <- sdpStr:
						default:
						}
					}
				}
			}
		}
	}()

	var finalSDPAnswer string
	select {
	case finalSDPAnswer = <-answerDone:
		recordSystemLog("[WebRTC] SDP Answer fully received and assembled!", "success")
	case <-sdpAnswerCtx.Done():
		pc.Close()
		return fmt.Errorf("SDP answer timeout (45s)")
	}

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  finalSDPAnswer,
	}
	if err := pc.SetRemoteDescription(answer); err != nil {
		pc.Close()
		return fmt.Errorf("set remote description: %w", err)
	}
	recordSystemLog("[WebRTC] Remote description set successfully", "success")

	// ── Step 10: Asynchronously Send and Receive ICE candidates ──
	go func() {
		for {
			select {
			case candidate := <-pendingICE:
				if groupChatID != 0 {
					cmdICE := soroushlib.NewICE(clientID, serverID, candidate)
					iceCtx, iceCancel := context.WithTimeout(ctx, 5*time.Second)
					soroushlib.SendGroupCommand(iceCtx, session, groupChatID, cmdICE, psk, config.GroupAccessHash)
					iceCancel()
				} else {
					iceMsg := soroushlib.FormatICECandidate(candidate)
					iceCtx, iceCancel := context.WithTimeout(ctx, 5*time.Second)
					soroushlib.SendTextMessage(iceCtx, session, workerUID, workerAH, iceMsg)
					iceCancel()
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Listen for incoming ICE candidates from worker using a SEPARATE text subscription
	// (avoids competing with the SDP answer listener on the same channel)
	iceTextSub := router.SubscribeText()
	go func() {
		defer router.UnsubscribeText(iceTextSub)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-iceTextSub:
				if !ok {
					return
				}
				if groupChatID != 0 {
					if !msg.IsGroup || msg.ChatID != groupChatID {
						continue
					}
					cmd, err := soroushlib.DecodeGroupCommand(msg.Text, psk)
					if err != nil {
						continue
					}
					if cmd.CID != clientID || cmd.SID != serverID {
						continue
					}
					if cmd.Cmd == soroushlib.CmdICE {
						if err := pc.AddICECandidate(webrtc.ICECandidateInit{Candidate: cmd.Data}); err != nil {
							recordSystemLog(fmt.Sprintf("[WebRTC] AddICECandidate failed: %v", err), "warn")
						}
					}
				} else {
					if msg.IsGroup || msg.FromUserID != workerUID {
						continue
					}
					if soroushlib.IsICECandidate(msg.Text) {
						candidateStr := soroushlib.ExtractICECandidate(msg.Text)
						if err := pc.AddICECandidate(webrtc.ICECandidateInit{Candidate: candidateStr}); err != nil {
							recordSystemLog(fmt.Sprintf("[WebRTC] AddICECandidate failed: %v", err), "warn")
						}
					}
				}
			}
		}
	}()

	recordSystemLog("[WebRTC] Waiting for data channel to open...", "info")

	// Wait for connection or timeout
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(35 * time.Second):
		tunnel.mu.Lock()
		isConnected := tunnel.phase == "connected"
		tunnel.mu.Unlock()
		if !isConnected {
			pc.Close()
			return fmt.Errorf("data channel open timeout (35s)")
		}
	}

	return nil
}

type callRecvResult struct {
	cid    uint32
	reader *soroushlib.TLReader
	err    error
}

// Local SOCKS5 proxy — yamux client over DataChannel
// ──────────────────────────────────────────────────────────────────────────────

func startLocalSOCKS5Proxy(ctx context.Context, dc *webrtc.DataChannel) {
	// Wrap the DataChannel in a stream-oriented adapter
	dcConn := soroushlib.NewDataChannelConn(dc)

	// Create yamux client session (we initiate streams → server accepts them)
	yamuxCfg := yamux.DefaultConfig()
	yamuxCfg.EnableKeepAlive = true
	yamuxCfg.KeepAliveInterval = 30 * time.Second
	yamuxCfg.ConnectionWriteTimeout = 10 * time.Second

	yamuxSession, err := yamux.Client(dcConn, yamuxCfg)
	if err != nil {
		recordSystemLog(fmt.Sprintf("[SOCKS5] Yamux client init failed: %v", err), "error")
		return
	}

	tunnel.mu.Lock()
	tunnel.yamuxSession = yamuxSession
	tunnel.mu.Unlock()

	// Start Yamux-level keepalive and active latency/reconnection monitoring
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if yamuxSession.IsClosed() {
					recordSystemLog("[SOCKS5] Yamux session closed unexpectedly, triggering reconnect...", "warn")
					dcConn.Close()
					return
				}

				// Measure RTT and verify connection health via active ping
				rttChan := make(chan time.Duration, 1)
				errChan := make(chan error, 1)
				go func() {
					rtt, err := yamuxSession.Ping()
					if err != nil {
						errChan <- err
					} else {
						rttChan <- rtt
					}
				}()

				select {
				case rtt := <-rttChan:
					tunnel.mu.Lock()
					tunnel.latencyMs = rtt.Milliseconds()
					tunnel.lastPingAt = time.Now()
					tunnel.mu.Unlock()
				case err := <-errChan:
					recordSystemLog(fmt.Sprintf("[SOCKS5] Yamux keepalive ping failed: %v. Reconnecting...", err), "error")
					dcConn.Close()
					return
				case <-time.After(5 * time.Second):
					recordSystemLog("[SOCKS5] Yamux keepalive ping timeout (5s). Reconnecting...", "error")
					dcConn.Close()
					return
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Start local TCP listener on state.socksPort dynamically
	state.mu.RLock()
	socksPort := state.socksPort
	state.mu.RUnlock()

	addr := fmt.Sprintf("127.0.0.1:%d", socksPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		recordSystemLog(fmt.Sprintf("[SOCKS5] Failed to listen on %s: %v", addr, err), "error")
		return
	}

	tunnel.mu.Lock()
	tunnel.socksListener = listener
	tunnel.mu.Unlock()

	recordSystemLog(fmt.Sprintf("[SOCKS5] Local proxy listening on %s", addr), "success")
	addLog(fmt.Sprintf("🌐 SOCKS5 proxy ready on %s — configure your browser to use it!", addr), "success")

	// Accept incoming local connections and pipe them through yamux
	for {
		localConn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if yamuxSession.IsClosed() {
				recordSystemLog("[SOCKS5] Yamux session closed, stopping proxy", "info")
				return
			}
			log.Printf("[SOCKS5] Accept error: %v", err)
			return
		}

		go func(conn net.Conn) {
			defer conn.Close()

			// Open a new yamux stream for this connection
			stream, err := yamuxSession.Open()
			if err != nil {
				log.Printf("[SOCKS5] Yamux stream open failed: %v", err)
				return
			}
			defer stream.Close()

			// Bidirectional pipe: local ↔ yamux stream ↔ DataChannel ↔ server SOCKS5
			done := make(chan struct{}, 2)
			go func() {
				io.Copy(stream, conn)
				done <- struct{}{}
			}()
			go func() {
				io.Copy(conn, stream)
				done <- struct{}{}
			}()
			<-done
		}(localConn)
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
// DH helper (for call encryption — generates client DH keys and computes fingerprint)
// ──────────────────────────────────────────────────────────────────────────────

func generateClientDH() (a *big.Int, gA []byte, gAHash []byte) {
	aBytes := make([]byte, 256)
	rand.Read(aBytes)
	a = new(big.Int).SetBytes(aBytes)

	g := big.NewInt(3)
	p, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF", 16)

	gABig := new(big.Int).Exp(g, a, p)
	gA = make([]byte, 256)
	gABigBytes := gABig.Bytes()
	copy(gA[256-len(gABigBytes):], gABigBytes)

	gAHash = soroushlib.Sha256Sum(gA)
	return
}

func computeFingerprint(gBBytes []byte, a *big.Int) int64 {
	gB := new(big.Int).SetBytes(gBBytes)
	p, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD129024E088A67CC74020BBEA63B139B22514A08798E3404DDEF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7EDEE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3DC2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F83655D23DCA3AD961C62F356208552BB9ED529077096966D670C354E4ABC9804F1746C08CA18217C32905E462E36CE3BE39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9DE2BCBF6955817183995497CEA956AE515D2261898FA051015728E5A8AACAA68FFFFFFFFFFFFFFFF", 16)
	s := new(big.Int).Exp(gB, a, p)
	sBytes := make([]byte, 256)
	sBigBytes := s.Bytes()
	copy(sBytes[256-len(sBigBytes):], sBigBytes)
	hash := soroushlib.Sha256Sum(sBytes)
	return int64(binary.LittleEndian.Uint64(hash[0:8]))
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

	socksReady := tunnel.socksListener != nil
	socksAddr := ""
	if socksReady {
		socksAddr = tunnel.socksListener.Addr().String()
	}

	resp := map[string]interface{}{
		"phase":      tunnel.phase,
		"latencyMs":  tunnel.latencyMs,
		"uptime":     uptime,
		"error":      tunnel.errorMsg,
		"socksReady": socksReady,
		"socksAddr":  socksAddr,
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
	wrappedDisc := soroushlib.WrapInitConnection(soroushlib.SoroushAppID, discBody)
	discoverCtx, discoverCancel := context.WithTimeout(context.Background(), 30*time.Second)
	_, _, discErr := session.SendAndWait(discoverCtx, wrappedDisc, true)
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

// Proxies logs request from the client frontend to the exit node server using GORM database PSK
func handleGetServerLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	serverURL := r.URL.Query().Get("serverUrl")
	if serverURL == "" {
		http.Error(w, `{"error":"serverUrl query parameter is required"}`, http.StatusBadRequest)
		return
	}

	// Fetch PSK from DB
	var tunnelCfg DBTunnelConfig
	if err := db.First(&tunnelCfg).Error; err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to retrieve tunnel configuration: %v"}`, err), http.StatusInternalServerError)
		return
	}

	// Make request to server panel
	logsURL := fmt.Sprintf("%s/api/logs/raw?psk=%s&format=json", serverURL, tunnelCfg.PSK)
	
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(logsURL)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to fetch logs from server: %v"}`, err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
