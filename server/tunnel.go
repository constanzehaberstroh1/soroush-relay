package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"soroush-relay/soroushlib"

	socks5 "github.com/armon/go-socks5"
	"github.com/hashicorp/yamux"
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
	groupChatID     int64
	groupAccessHash int64
	psk             []byte
}

type WorkerConnection struct {
	AccountID      string
	PhoneNumber    string
	PeerConnection *webrtc.PeerConnection
	DataChannel    *webrtc.DataChannel
	YamuxSession   *yamux.Session
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

	// Load group config from DB
	var groupCfg DBGroupConfig
	if err := db.First(&groupCfg).Error; err != nil || groupCfg.GroupChatID == 0 {
		serverTunnel.mu.Unlock()
		return fmt.Errorf("no group config set — configure the 'My lovely family' group chat ID first")
	}

	ctx, cancel := context.WithCancel(context.Background())
	serverTunnel.ctx = ctx
	serverTunnel.cancel = cancel
	serverTunnel.running = true
	serverTunnel.groupChatID = groupCfg.GroupChatID
	serverTunnel.groupAccessHash = groupCfg.GroupAccessHash
	if groupCfg.PSK != "" {
		serverTunnel.psk = []byte(groupCfg.PSK)
	} else {
		serverTunnel.psk = soroushlib.DefaultPSK
	}
	serverTunnel.mu.Unlock()

	go runGroupObserver(ctx)
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
// Group Observer — monitors group, sends heartbeats, handles DISCOVER
// Auto-reconnects on connection loss.
// ──────────────────────────────────────────────────────────────────────────────

func runGroupObserver(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			recordSystemLog(fmt.Sprintf("[GroupObserver] Panic: %v", r), "error")
		}
		serverTunnel.mu.Lock()
		serverTunnel.running = false
		serverTunnel.dispatcherReady = false
		serverTunnel.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		err := runGroupObserverOnce(ctx)
		if ctx.Err() != nil {
			return
		}

		recordSystemLog(fmt.Sprintf("[GroupObserver] Connection lost: %v. Reconnecting in 10s...", err), "warn")
		serverTunnel.mu.Lock()
		serverTunnel.dispatcherReady = false
		serverTunnel.mu.Unlock()

		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
	}
}

func runGroupObserverOnce(ctx context.Context) error {
	var account DBSoroushAccount
	if err := db.Where("status = ? AND length(auth_key) > 0", "connected").First(&account).Error; err != nil {
		recordSystemLog("[GroupObserver] No connected account available.", "error")
		return fmt.Errorf("no account: %w", err)
	}

	recordSystemLog(fmt.Sprintf("[GroupObserver] Starting with account: %s (UID: %d)", account.PhoneNumber, account.SoroushUserID), "info")

	session, transport := soroushlib.RestoreSession(account.AuthKey, account.AuthKeyID, account.ServerSalt)

	connCtx, connCancel := context.WithTimeout(ctx, 15*time.Second)
	if err := transport.Connect(connCtx); err != nil {
		connCancel()
		recordSystemLog(fmt.Sprintf("[GroupObserver] Transport connect failed: %v", err), "error")
		return fmt.Errorf("connect: %w", err)
	}
	connCancel()
	defer transport.Disconnect()

	recordSystemLog("[GroupObserver] Connected to Soroush ✅", "success")

	serverTunnel.mu.Lock()
	serverTunnel.dispatcherReady = true
	chatID := serverTunnel.groupChatID
	chatAH := serverTunnel.groupAccessHash
	psk := serverTunnel.psk
	serverTunnel.mu.Unlock()

	serverID := account.ID

	// Send initial heartbeat wrapped in initConnection (required by Soroush for new sessions).
	// Without initConnection, Soroush processes the RPC then closes the WebSocket.
	hb := soroushlib.NewHeartbeat(serverID, account.SoroushUserID, account.AccessHash, 0)
	encoded, err := soroushlib.EncodeGroupCommand(hb, psk)
	if err != nil {
		recordSystemLog(fmt.Sprintf("[GroupObserver] Encode heartbeat failed: %v", err), "error")
		return fmt.Errorf("encode heartbeat: %w", err)
	}
	hbBody := soroushlib.BuildSendChannelMessage(chatID, chatAH, encoded, time.Now().UnixNano())
	wrappedBody := soroushlib.WrapInitConnection(soroushlib.SoroushAppID, hbBody)

	hbCtx, hbCancel := context.WithTimeout(ctx, 30*time.Second)
	_, _, err = session.SendAndWait(hbCtx, wrappedBody, true)
	hbCancel()
	if err != nil {
		recordSystemLog(fmt.Sprintf("[GroupObserver] Initial heartbeat failed: %v", err), "warn")
	} else {
		recordSystemLog("[GroupObserver] Initial heartbeat sent and confirmed ✅", "success")
	}

	// Send ping_delay_disconnect to register keep-alive (75s disconnect delay)
	pingBody := soroushlib.BuildPingDelayDisconnectRequest(time.Now().UnixNano(), 75)
	session.Send(ctx, pingBody, true)

	repliedDiscovers := make(map[string]time.Time)

	heartbeatTicker := time.NewTicker(5 * time.Minute)
	defer heartbeatTicker.Stop()

	// Ping ticker to keep the Soroush WebSocket alive
	pingTicker := time.NewTicker(60 * time.Second)
	defer pingTicker.Stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- soroushlib.ListenForMessages(ctx, session, func(msg soroushlib.IncomingMessage) {
			if !msg.IsGroup || msg.ChatID != chatID {
				return
			}
			if msg.FromUserID == account.SoroushUserID {
				return
			}

			cmd, err := soroushlib.DecodeGroupCommand(msg.Text, psk)
			if err != nil {
				return
			}

			recordSystemLog(fmt.Sprintf("[GroupObserver] Received %s from UID=%d", cmd.Cmd, msg.FromUserID), "info")

			switch cmd.Cmd {
			case soroushlib.CmdDiscover:
				if _, ok := repliedDiscovers[cmd.CID]; ok {
					return
				}
				repliedDiscovers[cmd.CID] = time.Now()

				go func(targetCID string) {
					delay := time.Duration(2000+rand.Intn(2000)) * time.Millisecond
					time.Sleep(delay)

					offer := soroushlib.NewOffer(targetCID, serverID, account.SoroushUserID, account.AccessHash)
					offerCtx, offerCancel := context.WithTimeout(ctx, 10*time.Second)
					defer offerCancel()
					if err := soroushlib.SendGroupCommand(offerCtx, session, chatID, offer, psk, chatAH); err != nil {
						recordSystemLog(fmt.Sprintf("[GroupObserver] Failed to send OFFER: %v", err), "error")
					} else {
						recordSystemLog(fmt.Sprintf("[GroupObserver] Sent OFFER to client %s", targetCID), "success")
					}
				}(cmd.CID)

			case soroushlib.CmdCalling:
				if cmd.SID == serverID {
					recordSystemLog(fmt.Sprintf("[GroupObserver] Client %s is CALLING us!", cmd.CID), "success")
					go startWorkerListener(ctx, &account, msg.FromUserID)
				}

			case soroushlib.CmdDisconnect:
				recordSystemLog(fmt.Sprintf("[GroupObserver] DISCONNECT from %s", cmd.SID), "info")
			}
		})
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-pingTicker.C:
			// Keep the connection alive with periodic pings
			pingBody := soroushlib.BuildPingDelayDisconnectRequest(time.Now().UnixNano(), 75)
			session.Send(ctx, pingBody, true)
		case <-heartbeatTicker.C:
			hb := soroushlib.NewHeartbeat(serverID, account.SoroushUserID, account.AccessHash, len(serverTunnel.activeWorkers))
			hbCtx, hbCancel := context.WithTimeout(ctx, 10*time.Second)
			soroushlib.SendGroupCommand(hbCtx, session, chatID, hb, psk, chatAH)
			hbCancel()

			now := time.Now()
			for k, v := range repliedDiscovers {
				if now.Sub(v) > 5*time.Minute {
					delete(repliedDiscovers, k)
				}
			}
		case err := <-errCh:
			if err != nil && ctx.Err() == nil {
				recordSystemLog(fmt.Sprintf("[GroupObserver] Listen error: %v", err), "error")
				return err
			}
			return nil
		}
	}
}

// runDispatcher is the legacy dispatcher (kept for backward compatibility)
func runDispatcher(ctx context.Context) {
	recordSystemLog("[Dispatcher] Legacy dispatcher mode. Use Group Bus for new deployments.", "warn")
	// Find the dispatcher account (role = "dispatcher")
	var dispatcherAcc DBSoroushAccount
	if err := db.Where("role = ?", "dispatcher").First(&dispatcherAcc).Error; err != nil {
		recordSystemLog("[Dispatcher] No dispatcher account configured.", "error")
		return
	}

	session, transport := soroushlib.RestoreSession(dispatcherAcc.AuthKey, dispatcherAcc.AuthKeyID, dispatcherAcc.ServerSalt)
	connCtx, connCancel := context.WithTimeout(ctx, 15*time.Second)
	if err := transport.Connect(connCtx); err != nil {
		connCancel()
		return
	}
	connCancel()
	defer transport.Disconnect()

	serverTunnel.mu.Lock()
	serverTunnel.dispatcherReady = true
	serverTunnel.mu.Unlock()

	soroushlib.ListenForMessages(ctx, session, func(msg soroushlib.IncomingMessage) {
		if msg.Text == soroushlib.DispatcherSynRequest {
			handleDispatchRequest(ctx, session, msg.FromUserID, &dispatcherAcc)
		}
	})

	transport.Disconnect()
}

func handleDispatchRequest(ctx context.Context, session *soroushlib.MTProtoSession, clientUserID int64, dispatcherAcc *DBSoroushAccount) {
	var workerAcc DBSoroushAccount
	if err := db.Where("role = ? AND status = ?", "worker", "connected").First(&workerAcc).Error; err != nil {
		sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		soroushlib.SendTextMessage(sendCtx, session, clientUserID, 0, soroushlib.DispatcherNoWorkers)
		cancel()
		return
	}
	db.Model(&workerAcc).Update("status", "busy")
	response := soroushlib.FormatDispatcherResponse(workerAcc.SoroushUserID, workerAcc.AccessHash)
	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	soroushlib.SendTextMessage(sendCtx, session, clientUserID, 0, response)
	cancel()
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

		// Use callCtx directly — coder/websocket kills the socket on context expiry
		cid, reader, err := session.Recv(callCtx)

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

			addLog(fmt.Sprintf("✅ Worker %s: TUNNEL ACTIVE — SOCKS5 proxy ready", account.PhoneNumber), "success")

			// Start the SOCKS5 proxy server over yamux
			go startSOCKS5Server(dc, wc, account)
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

	recordSystemLog(fmt.Sprintf("[Worker %s] Call accepted. Waiting for SDP offer...", account.PhoneNumber), "info")

	// ── SDP Exchange: Listen for SDP_OFFER from client via direct message ──
	sdpCtx, sdpCancel := context.WithTimeout(ctx, 45*time.Second)
	defer sdpCancel()

	// Collect ICE candidates to send after SDP answer
	pendingICE := make(chan string, 32)
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		candidateJSON := c.ToJSON().Candidate
		select {
		case pendingICE <- candidateJSON:
		default:
		}
	})

	// Listen for SDP offer + ICE candidates from client
	sdpDone := make(chan bool, 1)
	go func() {
		soroushlib.ListenForMessages(sdpCtx, session, func(msg soroushlib.IncomingMessage) {
			if msg.IsGroup || msg.FromUserID != callEvent.AdminID {
				return
			}

			if soroushlib.IsSDPOffer(msg.Text) {
				sdpStr := soroushlib.ExtractSDP(msg.Text)
				recordSystemLog(fmt.Sprintf("[Worker %s] Received SDP offer (%d bytes)", account.PhoneNumber, len(sdpStr)), "info")

				offer := webrtc.SessionDescription{
					Type: webrtc.SDPTypeOffer,
					SDP:  sdpStr,
				}
				if err := pc.SetRemoteDescription(offer); err != nil {
					recordSystemLog(fmt.Sprintf("[Worker %s] SetRemoteDescription failed: %v", account.PhoneNumber, err), "error")
					return
				}

				answer, err := pc.CreateAnswer(nil)
				if err != nil {
					recordSystemLog(fmt.Sprintf("[Worker %s] CreateAnswer failed: %v", account.PhoneNumber, err), "error")
					return
				}
				if err := pc.SetLocalDescription(answer); err != nil {
					recordSystemLog(fmt.Sprintf("[Worker %s] SetLocalDescription failed: %v", account.PhoneNumber, err), "error")
					return
				}

				// Send SDP answer back via direct message
				answerMsg := soroushlib.FormatSDPAnswer(answer.SDP)
				sendCtx, sendCancel := context.WithTimeout(sdpCtx, 10*time.Second)
				soroushlib.SendTextMessage(sendCtx, session, callEvent.AdminID, 0, answerMsg)
				sendCancel()
				recordSystemLog(fmt.Sprintf("[Worker %s] SDP answer sent (%d bytes)", account.PhoneNumber, len(answer.SDP)), "success")

				// Send buffered ICE candidates (context-managed, no premature timeout)
				go func() {
					time.Sleep(500 * time.Millisecond)
					for {
						select {
						case candidate := <-pendingICE:
							iceMsg := soroushlib.FormatICECandidate(candidate)
							iceCtx, iceCancel := context.WithTimeout(sdpCtx, 5*time.Second)
							soroushlib.SendTextMessage(iceCtx, session, callEvent.AdminID, 0, iceMsg)
							iceCancel()
						case <-sdpCtx.Done():
							return
						}
					}
				}()

				sdpDone <- true
			}

			if soroushlib.IsICECandidate(msg.Text) {
				candidateStr := soroushlib.ExtractICECandidate(msg.Text)
				if err := pc.AddICECandidate(webrtc.ICECandidateInit{Candidate: candidateStr}); err != nil {
					recordSystemLog(fmt.Sprintf("[Worker %s] AddICECandidate failed: %v", account.PhoneNumber, err), "warn")
				}
			}
		})
	}()

	select {
	case <-sdpDone:
		recordSystemLog(fmt.Sprintf("[Worker %s] SDP negotiation complete!", account.PhoneNumber), "success")
	case <-sdpCtx.Done():
		pc.Close()
		return fmt.Errorf("SDP exchange timed out")
	}

	// Keep the worker alive until context is cancelled
	<-ctx.Done()
	pc.Close()
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Worker message handler — SOCKS5 proxy over yamux
// ──────────────────────────────────────────────────────────────────────────────

func startSOCKS5Server(dc *webrtc.DataChannel, wc *WorkerConnection, account *DBSoroushAccount) {
	// Wrap the DataChannel in a stream-oriented adapter
	dcConn := soroushlib.NewDataChannelConn(dc)

	// Create a yamux server session (client initiates streams → server accepts)
	yamuxCfg := yamux.DefaultConfig()
	yamuxCfg.EnableKeepAlive = true
	yamuxCfg.KeepAliveInterval = 30 * time.Second
	yamuxCfg.ConnectionWriteTimeout = 10 * time.Second

	yamuxSession, err := yamux.Server(dcConn, yamuxCfg)
	if err != nil {
		recordSystemLog(fmt.Sprintf("[Worker %s] Yamux server init failed: %v", account.PhoneNumber, err), "error")
		return
	}

	wc.YamuxSession = yamuxSession

	// Create a SOCKS5 server (no auth required — traffic is already encrypted via WebRTC)
	socks5Conf := &socks5.Config{}
	socks5Server, err := socks5.New(socks5Conf)
	if err != nil {
		recordSystemLog(fmt.Sprintf("[Worker %s] SOCKS5 server init failed: %v", account.PhoneNumber, err), "error")
		return
	}

	recordSystemLog(fmt.Sprintf("[Worker %s] SOCKS5 proxy server started over yamux", account.PhoneNumber), "success")

	// Accept incoming yamux streams and serve SOCKS5
	for {
		stream, err := yamuxSession.Accept()
		if err != nil {
			if yamuxSession.IsClosed() {
				recordSystemLog(fmt.Sprintf("[Worker %s] Yamux session closed", account.PhoneNumber), "info")
				return
			}
			recordSystemLog(fmt.Sprintf("[Worker %s] Yamux accept error: %v", account.PhoneNumber, err), "error")
			return
		}

		// Each stream is a new SOCKS5 connection from the client
		go func() {
			defer stream.Close()
			if err := socks5Server.ServeConn(stream); err != nil {
				log.Printf("[Worker %s] SOCKS5 stream error: %v", account.PhoneNumber, err)
			}
		}()
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Server Tunnel API Handlers
// ──────────────────────────────────────────────────────────────────────────────

func handleServerTunnelStart(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	if err := startServerTunnel(); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}
	addLog("Tunnel engine started — listening for DISCOVER messages", "success")
	json.NewEncoder(w).Encode(map[string]interface{}{"running": true})
}

func handleServerTunnelStop(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	stopServerTunnel()
	addLog("Tunnel engine stopped", "warn")
	json.NewEncoder(w).Encode(map[string]interface{}{"running": false})
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

// ──────────────────────────────────────────────────────────────────────────────
// Group Config API — GET/POST /api/group/config
// ──────────────────────────────────────────────────────────────────────────────

func handleGroupConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		var cfg DBGroupConfig
		db.First(&cfg)
		json.NewEncoder(w).Encode(cfg)

	case http.MethodPost:
		var req struct {
			GroupChatID     int64  `json:"groupChatId"`
			GroupAccessHash int64  `json:"groupAccessHash"`
			PSK             string `json:"psk"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"Invalid request"}`, http.StatusBadRequest)
			return
		}

		var cfg DBGroupConfig
		db.FirstOrCreate(&cfg)
		cfg.GroupChatID = req.GroupChatID
		cfg.GroupAccessHash = req.GroupAccessHash
		if req.PSK != "" {
			cfg.PSK = req.PSK
		}
		if err := db.Save(&cfg).Error; err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		addLog(fmt.Sprintf("Group config updated: ChatID=%d AccessHash=%d", cfg.GroupChatID, cfg.GroupAccessHash), "success")
		json.NewEncoder(w).Encode(cfg)

	default:
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
