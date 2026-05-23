package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"soroush-relay/soroushlib"
)

// ──────────────────────────────────────────────────────────────────────────────
// Pending OTP sessions — tracks live MTProto sessions during OTP flow
// ──────────────────────────────────────────────────────────────────────────────

type PendingOTPSession struct {
	PhoneNumber   string
	Name          string
	Transport     *soroushlib.ObfuscatedTransport
	Session       *soroushlib.MTProtoSession
	PhoneCodeHash []byte
	SessionID     string
	Cancel        context.CancelFunc
	ExpiresAt     time.Time
}

var (
	pendingMu       sync.Mutex
	pendingSessions = make(map[string]*PendingOTPSession)
)

// Request OTP JSON payloads
type OTPRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	Name        string `json:"name"`
}

type OTPVerifyRequest struct {
	PhoneNumber string `json:"phoneNumber"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	SessionID   string `json:"sessionId"`
}

func recordSystemLog(message string, logType string) {
	addLog(message, logType)
	fmt.Printf("[%s] %s\n", strings.ToUpper(logType), message)
}

// ──────────────────────────────────────────────────────────────────────────────
// Request OTP Handler — Real MTProto authentication
// ──────────────────────────────────────────────────────────────────────────────

func handleRequestOTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req OTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PhoneNumber == "" || req.Name == "" {
		http.Error(w, `{"error":"Invalid request payload (phoneNumber and name are required)"}`, http.StatusBadRequest)
		return
	}

	recordSystemLog(fmt.Sprintf("[Soroush MTProto] Starting real OTP flow for %s (%s)", soroushlib.MaskPhone(req.PhoneNumber), req.Name), "info")

	// Step 1: Connect transport
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	transport := soroushlib.NewTransport()
	if err := transport.Connect(ctx); err != nil {
		cancel()
		recordSystemLog(fmt.Sprintf("[Soroush MTProto] Transport connect failed: %v", err), "error")
		http.Error(w, fmt.Sprintf(`{"error":"Transport connect failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	recordSystemLog("[Soroush MTProto] WebSocket transport connected to wss://im-server.splus.ir/apiws", "success")

	// Step 2: DH key exchange
	session := soroushlib.NewSession(transport)
	if err := session.CreateAuthKey(ctx); err != nil {
		cancel()
		transport.Disconnect()
		recordSystemLog(fmt.Sprintf("[Soroush MTProto] Auth key exchange failed: %v", err), "error")
		http.Error(w, fmt.Sprintf(`{"error":"Auth key exchange failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	recordSystemLog(fmt.Sprintf("[Soroush MTProto] Auth key generated (auth_key_id=%d)", session.AuthKeyID), "success")
	cancel()

	// Step 3: Send auth.sendCode
	sendCodeBody := soroushlib.BuildSendCodeRequest(req.PhoneNumber, soroushlib.SoroushAppID, soroushlib.SoroushAppHash)
	wrappedBody := soroushlib.WrapInitConnection(soroushlib.SoroushAppID, sendCodeBody)

	recordSystemLog("[Soroush MTProto] Sending auth.sendCode via MTProto...", "info")

	bgCtx, bgCancel := context.WithCancel(context.Background())
	sendCtx, sendCancel := context.WithTimeout(bgCtx, 30*time.Second)

	recvCh := make(chan recvResult, 1)
	go func() {
		cid, reader, err := session.Recv(sendCtx)
		recvCh <- recvResult{cid: cid, reader: reader, err: err}
	}()

	msgID, err := session.Send(sendCtx, wrappedBody, true)
	if err != nil {
		sendCancel()
		bgCancel()
		transport.Disconnect()
		recordSystemLog(fmt.Sprintf("[Soroush MTProto] Send failed: %v", err), "error")
		http.Error(w, fmt.Sprintf(`{"error":"Send failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	var phoneCodeHash []byte
	for attempt := 0; attempt < 3; attempt++ {
		result := <-recvCh
		if result.err != nil {
			sendCancel()
			bgCancel()
			transport.Disconnect()
			recordSystemLog(fmt.Sprintf("[Soroush MTProto] Recv failed: %v", result.err), "error")
			http.Error(w, fmt.Sprintf(`{"error":"Recv failed: %s"}`, result.err.Error()), http.StatusInternalServerError)
			return
		}

		innerCID, innerReader := unwrapResponse(result.cid, result.reader, msgID)

		if innerCID == soroushlib.IDBadServerSalt {
			innerReader.ReadInt64()
			innerReader.ReadInt32()
			innerReader.ReadInt32()
			newSalt, _ := innerReader.ReadInt64()
			session.ServerSalt = newSalt
			recordSystemLog(fmt.Sprintf("[Soroush MTProto] Bad server salt, retrying (salt=%d)", newSalt), "info")
			go func() {
				cid, reader, err := session.Recv(sendCtx)
				recvCh <- recvResult{cid: cid, reader: reader, err: err}
			}()
			msgID, _ = session.Send(sendCtx, wrappedBody, true)
			continue
		}

		if innerCID == soroushlib.IDNewSession {
			innerReader.ReadInt64()
			innerReader.ReadInt64()
			newSalt, _ := innerReader.ReadInt64()
			session.ServerSalt = newSalt
			recordSystemLog(fmt.Sprintf("[Soroush MTProto] New session (salt=%d), waiting for RPC result...", newSalt), "info")
			go func() {
				cid, reader, err := session.Recv(sendCtx)
				recvCh <- recvResult{cid: cid, reader: reader, err: err}
			}()
			continue
		}

		var timeout int32
		phoneCodeHash, timeout, err = soroushlib.ParseSentCodeResponse(innerCID, innerReader)
		if err != nil {
			sendCancel()
			bgCancel()
			transport.Disconnect()
			recordSystemLog(fmt.Sprintf("[Soroush MTProto] sendCode failed: %v", err), "error")
			http.Error(w, fmt.Sprintf(`{"error":"sendCode failed: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		recordSystemLog(fmt.Sprintf("[Soroush MTProto] OTP sent! hash_len=%d, timeout=%d", len(phoneCodeHash), timeout), "success")
		recordSystemLog(fmt.Sprintf("[Soroush SMS Gateway] Verification code sent to %s", req.PhoneNumber), "success")
		break
	}

	sendCancel()

	sessionID := fmt.Sprintf("sess-%d", time.Now().UnixNano())
	pendingMu.Lock()
	pendingSessions[sessionID] = &PendingOTPSession{
		PhoneNumber:   req.PhoneNumber,
		Name:          req.Name,
		Transport:     transport,
		Session:       session,
		PhoneCodeHash: phoneCodeHash,
		SessionID:     sessionID,
		Cancel:        bgCancel,
		ExpiresAt:     time.Now().Add(5 * time.Minute),
	}
	pendingMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"phoneNumber": req.PhoneNumber,
		"sessionId":   sessionID,
		"message":     "OTP sent via Soroush. Enter the verification code you received.",
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Verify OTP Handler — Real MTProto sign-in
// ──────────────────────────────────────────────────────────────────────────────

func handleVerifyOTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req OTPVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PhoneNumber == "" || req.Code == "" || req.SessionID == "" {
		http.Error(w, `{"error":"Invalid request payload (phoneNumber, code, sessionId required)"}`, http.StatusBadRequest)
		return
	}

	pendingMu.Lock()
	pending, found := pendingSessions[req.SessionID]
	if found {
		delete(pendingSessions, req.SessionID)
	}
	pendingMu.Unlock()

	if !found {
		http.Error(w, `{"error":"Session not found or expired. Please request a new OTP."}`, http.StatusUnauthorized)
		return
	}

	recordSystemLog(fmt.Sprintf("[Soroush MTProto] Verifying OTP for %s", soroushlib.MaskPhone(req.PhoneNumber)), "info")

	signInBody := soroushlib.BuildSignInRequest(req.PhoneNumber, pending.PhoneCodeHash, req.Code)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	recvCh := make(chan recvResult, 1)
	go func() {
		cid, reader, err := pending.Session.Recv(ctx)
		recvCh <- recvResult{cid: cid, reader: reader, err: err}
	}()

	msgID, err := pending.Session.Send(ctx, signInBody, true)
	if err != nil {
		pending.Cancel()
		pending.Transport.Disconnect()
		recordSystemLog(fmt.Sprintf("[Soroush MTProto] signIn send failed: %v", err), "error")
		http.Error(w, fmt.Sprintf(`{"error":"signIn failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	result := <-recvCh
	if result.err != nil {
		pending.Cancel()
		pending.Transport.Disconnect()
		recordSystemLog(fmt.Sprintf("[Soroush MTProto] signIn recv failed: %v", result.err), "error")
		http.Error(w, fmt.Sprintf(`{"error":"signIn recv failed: %s"}`, result.err.Error()), http.StatusInternalServerError)
		return
	}

	innerCID, innerReader := unwrapResponse(result.cid, result.reader, msgID)

	userID, firstName, lastName, accessHash, err := soroushlib.ParseAuthorizationResponse(innerCID, innerReader)
	if err != nil {
		pending.Cancel()
		pending.Transport.Disconnect()
		recordSystemLog(fmt.Sprintf("[Soroush MTProto] Auth failed: %v", err), "error")
		http.Error(w, fmt.Sprintf(`{"error":"Authentication failed: %s"}`, err.Error()), http.StatusUnauthorized)
		return
	}

	displayName := strings.TrimSpace(firstName + " " + lastName)
	recordSystemLog(fmt.Sprintf("[Soroush MTProto] Verified! User: %s (ID: %d)", displayName, userID), "success")

	newAcc := DBSoroushAccount{
		ID:            fmt.Sprintf("acc-%d", time.Now().UnixNano()),
		PhoneNumber:   req.PhoneNumber,
		Name:          req.Name,
		SoroushUserID: userID,
		AccessHash:    accessHash,
		DisplayName:   displayName,
		AuthKey:       pending.Session.AuthKey,
		AuthKeyID:     soroushlib.Int64ToBytes(pending.Session.AuthKeyID),
		ServerSalt:    soroushlib.Int64ToBytes(pending.Session.ServerSalt),
		SessionData:   fmt.Sprintf(`{"auth_key_id":%d,"dc_id":2}`, pending.Session.AuthKeyID),
		DcID:          2,
		Status:        "connected",
		LastActive:    "Just authenticated",
		CreatedAt:     time.Now(),
	}

	if err := db.Save(&newAcc).Error; err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Database write failure: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	recordSystemLog(fmt.Sprintf("[Soroush Account Pool] Session activated. Phone: %s | UserID: %d | Name: %s", newAcc.PhoneNumber, newAcc.SoroushUserID, newAcc.DisplayName), "success")

	pending.Cancel()
	pending.Transport.Disconnect()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":            newAcc.ID,
		"phoneNumber":   newAcc.PhoneNumber,
		"name":          newAcc.Name,
		"soroushUserId": newAcc.SoroushUserID,
		"displayName":   newAcc.DisplayName,
		"status":        newAcc.Status,
		"lastActive":    newAcc.LastActive,
		"createdAt":     newAcc.CreatedAt,
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────────────────────

type recvResult struct {
	cid    uint32
	reader *soroushlib.TLReader
	err    error
}

func unwrapResponse(cid uint32, r *soroushlib.TLReader, expectedMsgID int64) (uint32, *soroushlib.TLReader) {
	switch cid {
	case soroushlib.IDRPCResult:
		r.ReadInt64()
		innerCID, _ := r.ReadUint32()
		rem := r.Remaining()
		data, _ := r.ReadRaw(rem)
		return innerCID, soroushlib.NewTLReader(data)

	case soroushlib.IDMsgContainer:
		count, _ := r.ReadInt32()
		for i := int32(0); i < count; i++ {
			r.ReadInt64()
			r.ReadInt32()
			bodyLen, _ := r.ReadInt32()
			body, err := r.ReadRaw(int(bodyLen))
			if err != nil {
				continue
			}
			subReader := soroushlib.NewTLReader(body)
			subCID, _ := subReader.ReadUint32()
			if subCID == soroushlib.IDRPCResult {
				subReader.ReadInt64()
				innerCID, _ := subReader.ReadUint32()
				rem := subReader.Remaining()
				data, _ := subReader.ReadRaw(rem)
				return innerCID, soroushlib.NewTLReader(data)
			}
			if subCID == soroushlib.IDBadServerSalt || subCID == soroushlib.IDNewSession {
				return subCID, subReader
			}
		}
		return cid, r

	default:
		return cid, r
	}
}
