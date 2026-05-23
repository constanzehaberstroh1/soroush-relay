package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"soroush-relay/soroushlib"

	"github.com/pion/webrtc/v4"
)

// ──────────────────────────────────────────────────────────────────────────────
// Server Tunnel Engine — Dispatcher + Worker Pool
// ──────────────────────────────────────────────────────────────────────────────

type ServerTunnelEngine struct {
	mu              sync.Mutex
	running         bool
	ctx             context.Context
	cancel          context.CancelFunc
	activeWorkers   map[string]*WorkerConnection // keyed by account ID
	dispatcherReady bool
}

type WorkerConnection struct {
	AccountID      string
	PhoneNumber    string
	PeerConnection *webrtc.PeerConnection
	DataChannel    *webrtc.DataChannel
	Phase          string // "idle", "connecting", "active"
	LatencyMs      int64
	ConnectedAt    time.Time
	ClientUserID   int64
}

var serverTunnel = &ServerTunnelEngine{
	activeWorkers: make(map[string]*WorkerConnection),
}

// ──────────────────────────────────────────────────────────────────────────────
// Start Server Tunnel — launches dispatcher listener + worker standby
// ──────────────────────────────────────────────────────────────────────────────

func startServerTunnel() error {
	serverTunnel.mu.Lock()
	if serverTunnel.running {
		serverTunnel.mu.Unlock()
		return fmt.Errorf("server tunnel already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	serverTunnel.ctx = ctx
	serverTunnel.cancel = cancel
	serverTunnel.running = true
	serverTunnel.mu.Unlock()

	go runDispatcher(ctx)
	return nil
}

func stopServerTunnel() {
	serverTunnel.mu.Lock()
	defer serverTunnel.mu.Unlock()

	if serverTunnel.cancel != nil {
		serverTunnel.cancel()
	}

	// Close all worker connections
	for id, wc := range serverTunnel.activeWorkers {
		if wc.DataChannel != nil {
			wc.DataChannel.Close()
		}
		if wc.PeerConnection != nil {
			wc.PeerConnection.Close()
		}
		delete(serverTunnel.activeWorkers, id)
	}

	serverTunnel.running = false
	serverTunnel.dispatcherReady = false
	addLog("Server tunnel engine stopped.", "warn")
}

// ──────────────────────────────────────────────────────────────────────────────
// Dispatcher — listens for SYN_REQ_V1 and assigns workers
// ──────────────────────────────────────────────────────────────────────────────

func runDispatcher(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			recordSystemLog(fmt.Sprintf("[Dispatcher] Panic: %v", r), "error")
		}
	}()

	// Find the dispatcher account (role = "dispatcher")
	var dispatcherAcc DBSoroushAccount
	if err := db.Where("role = ?", "dispatcher").First(&dispatcherAcc).Error; err != nil {
		recordSystemLog("[Dispatcher] No dispatcher account configured. Set an account role to 'dispatcher'.", "error")
		return
	}

	recordSystemLog(fmt.Sprintf("[Dispatcher] Starting with account: %s (ID: %d)", dispatcherAcc.PhoneNumber, dispatcherAcc.SoroushUserID), "info")

	// Connect dispatcher to Soroush
	session, transport := soroushlib.RestoreSession(dispatcherAcc.AuthKey, dispatcherAcc.AuthKeyID, dispatcherAcc.ServerSalt)

	connCtx, connCancel := context.WithTimeout(ctx, 15*time.Second)
	if err := transport.Connect(connCtx); err != nil {
		connCancel()
		recordSystemLog(fmt.Sprintf("[Dispatcher] Transport connect failed: %v", err), "error")
		return
	}
	connCancel()

	recordSystemLog("[Dispatcher] Connected to Soroush. Listening for dispatch requests...", "success")

	serverTunnel.mu.Lock()
	serverTunnel.dispatcherReady = true
	serverTunnel.mu.Unlock()

	// Listen for incoming messages
	err := soroushlib.ListenForMessages(ctx, session, func(msg soroushlib.IncomingMessage) {
		if msg.Text == soroushlib.DispatcherSynRequest {
			recordSystemLog(fmt.Sprintf("[Dispatcher] SYN_REQ_V1 from UserID=%d", msg.FromUserID), "info")
			handleDispatchRequest(ctx, session, msg.FromUserID, &dispatcherAcc)
		}
	})

	if err != nil && ctx.Err() == nil {
		recordSystemLog(fmt.Sprintf("[Dispatcher] Listen error: %v", err), "error")
	}

	transport.Disconnect()
}

func handleDispatchRequest(ctx context.Context, session *soroushlib.MTProtoSession, clientUserID int64, dispatcherAcc *DBSoroushAccount) {
	// Find an idle worker account
	var workerAcc DBSoroushAccount
	if err := db.Where("role = ? AND status = ?", "worker", "connected").First(&workerAcc).Error; err != nil {
		recordSystemLog("[Dispatcher] No idle workers available!", "error")
		// Reply with NACK
		sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		soroushlib.SendTextMessage(sendCtx, session, clientUserID, 0, soroushlib.DispatcherNoWorkers)
		cancel()
		return
	}

	// Mark worker as busy
	db.Model(&workerAcc).Update("status", "busy")

	// Reply with ACK_ROUTE
	response := soroushlib.FormatDispatcherResponse(workerAcc.SoroushUserID, workerAcc.AccessHash)
	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err := soroushlib.SendTextMessage(sendCtx, session, clientUserID, 0, response)
	cancel()

	if err != nil {
		recordSystemLog(fmt.Sprintf("[Dispatcher] Failed to send ACK_ROUTE: %v", err), "error")
		db.Model(&workerAcc).Update("status", "connected")
		return
	}

	recordSystemLog(fmt.Sprintf("[Dispatcher] Assigned worker %s (ID: %d) to client UserID=%d", workerAcc.PhoneNumber, workerAcc.SoroushUserID, clientUserID), "success")

	// Start worker listener for incoming calls
	go startWorkerListener(ctx, &workerAcc, clientUserID)
}

// ──────────────────────────────────────────────────────────────────────────────
// Worker — listens for incoming WebRTC call and sets up data channel
// ──────────────────────────────────────────────────────────────────────────────

func startWorkerListener(ctx context.Context, account *DBSoroushAccount, clientUserID int64) {
	defer func() {
		if r := recover(); r != nil {
			recordSystemLog(fmt.Sprintf("[Worker %s] Panic: %v", account.PhoneNumber, r), "error")
		}
		db.Model(account).Update("status", "connected")
	}()

	recordSystemLog(fmt.Sprintf("[Worker %s] Waiting for incoming call from UserID=%d...", account.PhoneNumber, clientUserID), "info")

	// Connect worker to Soroush
	session, transport := soroushlib.RestoreSession(account.AuthKey, account.AuthKeyID, account.ServerSalt)

	connCtx, connCancel := context.WithTimeout(ctx, 15*time.Second)
	if err := transport.Connect(connCtx); err != nil {
		connCancel()
		recordSystemLog(fmt.Sprintf("[Worker %s] Transport connect failed: %v", account.PhoneNumber, err), "error")
		return
	}
	connCancel()
	defer transport.Disconnect()

	recordSystemLog(fmt.Sprintf("[Worker %s] Connected. Listening for incoming calls...", account.PhoneNumber), "success")

	// Listen for call events
	callCtx, callCancel := context.WithTimeout(ctx, 60*time.Second)
	defer callCancel()

	for {
		select {
		case <-callCtx.Done():
			recordSystemLog(fmt.Sprintf("[Worker %s] Call wait timeout", account.PhoneNumber), "warn")
			return
		default:
		}

		recvCtx, recvCancel := context.WithTimeout(callCtx, 30*time.Second)
		cid, reader, err := session.Recv(recvCtx)
		recvCancel()

		if err != nil {
			if callCtx.Err() != nil {
				return
			}
			continue
		}

		// Unwrap to find call events
		innerCID, innerReader := unwrapResponse(cid, reader, 0)

		if innerCID == soroushlib.IDUpdatePhoneCall {
			callEvent, err := soroushlib.ParseCallUpdate(innerReader)
			if err != nil || callEvent == nil {
				continue
			}

			if callEvent.Type == "requested" && callEvent.AdminID == clientUserID {
				recordSystemLog(fmt.Sprintf("[Worker %s] Incoming call from client! CallID=%d", account.PhoneNumber, callEvent.CallID), "success")

				// Accept the call and set up WebRTC
				if err := handleIncomingCall(ctx, session, account, callEvent); err != nil {
					recordSystemLog(fmt.Sprintf("[Worker %s] Call handling failed: %v", account.PhoneNumber, err), "error")
				}
				return
			}
		}
	}
}

func handleIncomingCall(ctx context.Context, session *soroushlib.MTProtoSession, account *DBSoroushAccount, callEvent *soroushlib.CallEvent) error {
	// Acknowledge the call
	recvBody := soroushlib.BuildPhoneReceivedCall(callEvent.CallID, callEvent.AccessHash)
	ackCtx, ackCancel := context.WithTimeout(ctx, 10*time.Second)
	session.Send(ackCtx, recvBody, true)
	ackCancel()

	// Set up WebRTC PeerConnection (answerer)
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

	// Add TURN servers from call event if available
	for _, conn := range callEvent.Connections {
		if conn.Turn && conn.Username != "" {
			turnURL := fmt.Sprintf("turn:%s:%d", conn.IP, conn.Port)
			iceServers = append(iceServers, webrtc.ICEServer{
				URLs:           []string{turnURL},
				Username:       conn.Username,
				Credential:     conn.Password,
				CredentialType: webrtc.ICECredentialTypePassword,
			})
		}
	}

	config := webrtc.Configuration{
		ICEServers:    iceServers,
		BundlePolicy:  webrtc.BundlePolicyMaxBundle,
		RTCPMuxPolicy: webrtc.RTCPMuxPolicyRequire,
	}

	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("create peer connection: %w", err)
	}

	// Add dummy audio track (to match the offerer)
	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		"audio0",
		"soroush-worker-stream",
	)
	if err != nil {
		pc.Close()
		return fmt.Errorf("create audio track: %w", err)
	}
	pc.AddTrack(audioTrack)

	// Worker connection tracking
	wc := &WorkerConnection{
		AccountID:      account.ID,
		PhoneNumber:    account.PhoneNumber,
		PeerConnection: pc,
		Phase:          "connecting",
		ClientUserID:   callEvent.AdminID,
	}

	serverTunnel.mu.Lock()
	serverTunnel.activeWorkers[account.ID] = wc
	serverTunnel.mu.Unlock()

	// Handle incoming data channel
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		recordSystemLog(fmt.Sprintf("[Worker %s] Data channel '%s' received", account.PhoneNumber, dc.Label()), "success")

		wc.DataChannel = dc

		dc.OnOpen(func() {
			recordSystemLog(fmt.Sprintf("[Worker %s] Data channel OPEN!", account.PhoneNumber), "success")
			wc.Phase = "active"
			wc.ConnectedAt = time.Now()

			db.Model(account).Updates(map[string]interface{}{
				"status":      "tunnel_active",
				"last_active": "Tunnel active",
			})

			addLog(fmt.Sprintf("✅ Worker %s: TUNNEL ACTIVE", account.PhoneNumber), "success")
		})

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			handleWorkerMessage(dc, msg, wc, account)
		})

		dc.OnClose(func() {
			recordSystemLog(fmt.Sprintf("[Worker %s] Data channel closed", account.PhoneNumber), "warn")
			wc.Phase = "idle"
			db.Model(account).Updates(map[string]interface{}{
				"status":      "connected",
				"last_active": "Tunnel disconnected",
			})
		})
	})

	pc.OnICEConnectionStateChange(func(iceState webrtc.ICEConnectionState) {
		recordSystemLog(fmt.Sprintf("[Worker %s] ICE: %s", account.PhoneNumber, iceState.String()), "info")
	})

	// Accept the call via Soroush signaling
	gB := make([]byte, 256)
	_, err = session.Send(ctx, soroushlib.BuildPhoneAcceptCall(callEvent.CallID, callEvent.AccessHash, gB), true)
	if err != nil {
		pc.Close()
		return fmt.Errorf("send phone.acceptCall: %w", err)
	}

	recordSystemLog(fmt.Sprintf("[Worker %s] Call accepted. WebRTC negotiating...", account.PhoneNumber), "info")

	// Keep the worker alive
	<-ctx.Done()
	pc.Close()
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Worker message handler — pulse protocol
// ──────────────────────────────────────────────────────────────────────────────

type PulseMessage struct {
	Type      string `json:"type"`
	Timestamp int64  `json:"timestamp,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

func handleWorkerMessage(dc *webrtc.DataChannel, msg webrtc.DataChannelMessage, wc *WorkerConnection, account *DBSoroushAccount) {
	var pulse PulseMessage
	if err := json.Unmarshal(msg.Data, &pulse); err != nil {
		log.Printf("[Worker %s] Invalid message: %v", account.PhoneNumber, err)
		return
	}

	switch pulse.Type {
	case "init_pulse":
		// Calculate latency
		latency := (time.Now().UnixNano() - pulse.Timestamp) / 1e6
		if latency < 0 {
			latency = 0
		}
		wc.LatencyMs = latency

		recordSystemLog(fmt.Sprintf("[Worker %s] Init pulse from %s. Latency: %dms", account.PhoneNumber, pulse.ClientID, latency), "success")

		// Send ACK
		ack := PulseMessage{Type: "ack_pulse", LatencyMs: latency}
		data, _ := json.Marshal(ack)
		dc.Send(data)

	case "ping":
		// Reply with pong
		pong := PulseMessage{Type: "pong", Timestamp: pulse.Timestamp}
		data, _ := json.Marshal(pong)
		dc.Send(data)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Server Tunnel API Handlers
// ──────────────────────────────────────────────────────────────────────────────

func handleServerTunnelStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	if err := startServerTunnel(); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"message":"Server tunnel engine started"}`))
}

func handleServerTunnelStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	stopServerTunnel()
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Server tunnel engine stopped"}`))
}

func handleServerTunnelStatus(w http.ResponseWriter, r *http.Request) {
	serverTunnel.mu.Lock()
	defer serverTunnel.mu.Unlock()

	workers := make([]map[string]interface{}, 0)
	for _, wc := range serverTunnel.activeWorkers {
		uptime := "0s"
		if wc.Phase == "active" {
			uptime = time.Since(wc.ConnectedAt).Round(time.Second).String()
		}
		workers = append(workers, map[string]interface{}{
			"accountId":   wc.AccountID,
			"phone":       wc.PhoneNumber,
			"phase":       wc.Phase,
			"latencyMs":   wc.LatencyMs,
			"uptime":      uptime,
			"clientUserId": wc.ClientUserID,
		})
	}

	resp := map[string]interface{}{
		"running":         serverTunnel.running,
		"dispatcherReady": serverTunnel.dispatcherReady,
		"activeWorkers":   workers,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleSetAccountRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		AccountID string `json:"accountId"`
		Role      string `json:"role"` // "dispatcher", "worker", ""
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request"}`, http.StatusBadRequest)
		return
	}

	if req.Role != "dispatcher" && req.Role != "worker" && req.Role != "" {
		http.Error(w, `{"error":"Role must be 'dispatcher', 'worker', or empty"}`, http.StatusBadRequest)
		return
	}

	if err := db.Model(&DBSoroushAccount{}).Where("id = ?", req.AccountID).Update("role", req.Role).Error; err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	addLog(fmt.Sprintf("Account %s role set to '%s'", req.AccountID, req.Role), "success")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Role updated"})
}
