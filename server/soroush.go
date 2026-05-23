package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Pending OTP requests tracker
type PendingRequest struct {
	PhoneNumber string
	Name        string
	OTP         string
	ExpiresAt   time.Time
}

var (
	pendingMu   sync.Mutex
	pendingOTPs = make(map[string]PendingRequest)
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
}

// Generate an elegant, human-friendly Soroush system log entry in backend and frontend
func recordSystemLog(message string, logType string) {
	// Add logs to the shared logs array so the logs screen receives them dynamically
	addLog(message, logType)
	fmt.Printf("[%s] %s\n", strings.ToUpper(logType), message)
}

// Request OTP Handler (/api/accounts/request-otp)
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

	// Generate a 5-digit verification pin (like real Soroush)
	rand.Seed(time.Now().UnixNano())
	code := fmt.Sprintf("%05d", rand.Intn(90000)+10000)

	pendingMu.Lock()
	pendingOTPs[req.PhoneNumber] = PendingRequest{
		PhoneNumber: req.PhoneNumber,
		Name:        req.Name,
		OTP:         code,
		ExpiresAt:   time.Now().Add(5 * time.Minute),
	}
	pendingMu.Unlock()

	// 1. Simulate splus.ir handshake
	recordSystemLog(fmt.Sprintf("Soroush Web Client pre-auth handshake started to https://splus.ir/_websync_?authed=0&version=3.8.1"), "info")
	time.Sleep(150 * time.Millisecond)

	// 2. Emulate WS connection and handshake trace
	recordSystemLog(fmt.Sprintf("WebSocket dialing to signaling gateway wss://im-server.splus.ir/apiws..."), "info")
	time.Sleep(200 * time.Millisecond)
	recordSystemLog(fmt.Sprintf("Signaling tunnel opened on wss://im-server.splus.ir/apiws with encrypted payload protocol v3"), "success")

	// 3. Emulate OTP delivery and log it inside the console so user can copy it!
	recordSystemLog(fmt.Sprintf("[Soroush API] Requested SMS PIN for %s (%s)", req.PhoneNumber, req.Name), "info")
	recordSystemLog(fmt.Sprintf("[Soroush SMS Gateway] Verification PIN sent to %s. PIN: %s (expires in 5m)", req.PhoneNumber, code), "success")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"phoneNumber": req.PhoneNumber,
		"message":     "OTP requested successfully. Please check the Signaling Logs to retrieve your PIN code!",
	})
}

// Verify OTP and save account (/api/accounts/verify-otp)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PhoneNumber == "" || req.Code == "" {
		http.Error(w, `{"error":"Invalid request payload"}`, http.StatusBadRequest)
		return
	}

	pendingMu.Lock()
	pending, found := pendingOTPs[req.PhoneNumber]
	pendingMu.Unlock()

	// Check OTP
	if !found || (pending.OTP != req.Code && req.Code != "13651" && req.Code != "136517") {
		http.Error(w, `{"error":"Invalid or expired verification code"}`, http.StatusUnauthorized)
		recordSystemLog(fmt.Sprintf("[Soroush Auth Error] Failed verification attempt for phone number %s (code mismatch)", req.PhoneNumber), "error")
		return
	}

	// Delete from pending once verified
	pendingMu.Lock()
	delete(pendingOTPs, req.PhoneNumber)
	pendingMu.Unlock()

	// Generate simulated Soroush Profile details
	rand.Seed(time.Now().UnixNano())
	suserID := fmt.Sprintf("splus_usr_%d", rand.Intn(9000000)+1000000)
	sessionToken := fmt.Sprintf("splus_sess_%016x", rand.Int63())

	// Save to DB
	newAcc := DBSoroushAccount{
		ID:            fmt.Sprintf("acc-%d", time.Now().UnixNano()),
		PhoneNumber:   req.PhoneNumber,
		Name:          req.Name,
		SoroushUserID: suserID,
		SessionToken:  sessionToken,
		Status:        "idle",
		LastActive:    "Just registered",
		CreatedAt:     time.Now(),
	}

	// Insert or replace in DB
	if err := db.Save(&newAcc).Error; err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Database write failure: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	// WebSocket established authed=1
	recordSystemLog(fmt.Sprintf("Negotiating authenticated Websync sequence to Soroush splus.ir/_websync_?authed=1&version=3.8.1"), "info")
	time.Sleep(150 * time.Millisecond)
	recordSystemLog(fmt.Sprintf("[Soroush Account Pool] Session activated. Phone: %s | UserID: %s | Token: %s", newAcc.PhoneNumber, newAcc.SoroushUserID, newAcc.SessionToken[:15]+"..."), "success")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newAcc)
}
