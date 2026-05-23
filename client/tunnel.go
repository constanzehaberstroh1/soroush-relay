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
	tunnel.mu.Lock()
	defer tunnel.mu.Unlock()

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

	// ── Step 2: Get dispatcher account ID from config ──
	var config DBTunnelConfig
	if err := db.First(&config).Error; err != nil || config.DispatcherUserID == 0 {
		setTunnelError("Dispatcher account not configured. Set it in tunnel settings.")
		return
	}
	recordSystemLog(fmt.Sprintf("[Tunnel] Dispatcher target: UserID=%d", config.DispatcherUserID), "info")

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

	// ── Step 4: Smart Dispatch — send SYN to dispatcher ──
	recordSystemLog("[Tunnel] Sending dispatch request (SYN_REQ_V1)...", "info")
	sendCtx, sendCancel := context.WithTimeout(ctx, 10*time.Second)
	err := soroushlib.SendTextMessage(sendCtx, session, config.DispatcherUserID, config.DispatcherAccessHash, soroushlib.DispatcherSynRequest)
	sendCancel()
	if err != nil {
		setTunnelError(fmt.Sprintf("Failed to send dispatch request: %v", err))
		return
	}
	recordSystemLog("[Tunnel] Dispatch SYN sent. Waiting for worker assignment...", "info")

	// ── Step 5: Wait for ACK_ROUTE response ──
	workerCh := make(chan workerAssignment, 1)
	listenCtx, listenCancel := context.WithTimeout(ctx, 30*time.Second)
	go func() {
		soroushlib.ListenForMessages(listenCtx, session, func(msg soroushlib.IncomingMessage) {
			if msg.FromUserID == config.DispatcherUserID {
				uid, ah, ok := soroushlib.ParseDispatcherResponse(msg.Text)
				if ok {
					workerCh <- workerAssignment{userID: uid, accessHash: ah}
				} else if msg.Text == soroushlib.DispatcherNoWorkers {
					workerCh <- workerAssignment{err: fmt.Errorf("no idle workers available on server")}
				}
			}
		})
	}()

	select {
	case <-ctx.Done():
		listenCancel()
		setTunnelError("Tunnel cancelled during dispatch")
		return
	case <-listenCtx.Done():
		listenCancel()
		setTunnelError("Timeout waiting for dispatcher response (30s)")
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
		recordSystemLog(fmt.Sprintf("[Tunnel] Worker assigned: UserID=%d. Initiating WebRTC call...", wa.userID), "success")
	}

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
		tunnel.phase = "idle"
		tunnel.mu.Unlock()

		state.mu.Lock()
		state.tunnelActive = false
		state.mu.Unlock()
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

	// ── Generate SDP Offer ──
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("create offer: %w", err)
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("set local description: %w", err)
	}

	// Wait for ICE gathering to complete
	gatherComplete := webrtc.GatheringCompletePromise(pc)
	select {
	case <-gatherComplete:
	case <-time.After(15 * time.Second):
		recordSystemLog("[WebRTC] ICE gathering timeout, proceeding with partial candidates", "warn")
	case <-ctx.Done():
		return ctx.Err()
	}

	localDesc := pc.LocalDescription()
	recordSystemLog(fmt.Sprintf("[WebRTC] SDP Offer generated (%d bytes)", len(localDesc.SDP)), "success")

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

			// If we got TURN credentials from the call, update ICE
			if len(callEvent.Connections) > 0 {
				for _, conn := range callEvent.Connections {
					if conn.Turn && conn.Username != "" {
						recordSystemLog(fmt.Sprintf("[WebRTC] Received TURN: %s:%d (user: %s)", conn.IP, conn.Port, conn.Username), "info")
					}
				}
			}
		}
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

// handleTunnelConfig manages the dispatcher account configuration
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
			DispatcherUserID    int64 `json:"dispatcherUserId"`
			DispatcherAccessHash int64 `json:"dispatcherAccessHash"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid request"}`, http.StatusBadRequest)
			return
		}

		config := DBTunnelConfig{
			ID:                   1,
			DispatcherUserID:     req.DispatcherUserID,
			DispatcherAccessHash: req.DispatcherAccessHash,
		}
		db.Save(&config)

		addLog(fmt.Sprintf("Dispatcher configured: UserID=%d", req.DispatcherUserID), "success")
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
