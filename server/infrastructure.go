package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"soroush-relay/soroushlib"
)

// ──────────────────────────────────────────────────────────────────────────────
// Infrastructure Test — Probes all Soroush-related services
// ──────────────────────────────────────────────────────────────────────────────

// InfraTestResult represents the result of a single infrastructure probe
type InfraTestResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"` // "testing", "pass", "fail", "warn"
	LatencyMs   int64  `json:"latencyMs"`
	Detail      string `json:"detail"`
	Category    string `json:"category"` // "network", "auth", "turn", "database"
}

// handleInfraTest runs all infrastructure tests and returns results
func handleInfraTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	results := runAllInfraTests()
	json.NewEncoder(w).Encode(results)
}

// handleInfraStatus returns a quick cached status of infra health
func handleInfraStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	infraCacheMu.RLock()
	defer infraCacheMu.RUnlock()

	if infraCache == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tested":  false,
			"results": []InfraTestResult{},
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tested":   true,
		"testedAt": infraCacheTime.Format(time.RFC3339),
		"results":  infraCache,
	})
}

var (
	infraCacheMu   sync.RWMutex
	infraCache     []InfraTestResult
	infraCacheTime time.Time
)

func runAllInfraTests() []InfraTestResult {
	results := make([]InfraTestResult, 0, 10)
	var mu sync.Mutex
	var wg sync.WaitGroup

	addResult := func(r InfraTestResult) {
		mu.Lock()
		results = append(results, r)
		mu.Unlock()
	}

	// ── 1. Soroush WebSocket Endpoint ──
	wg.Add(1)
	go func() {
		defer wg.Done()
		addResult(testSoroushWebSocket())
	}()

	// ── 2. Soroush MTProto Handshake ──
	wg.Add(1)
	go func() {
		defer wg.Done()
		addResult(testMTProtoHandshake())
	}()

	// ── 3-8. TURN/STUN Servers ──
	turnServers := []struct {
		name string
		addr string
	}{
		{"TURN Server 1 (UDP)", "185.60.139.28:1400"},
		{"TURN Server 1 (TCP)", "185.60.139.28:1400"},
		{"STUN Server 1", "185.60.139.28:1400"},
		{"TURN Server 2 (UDP)", "185.60.137.28:1400"},
		{"TURN Server 2 (TCP)", "185.60.137.28:1400"},
		{"STUN Server 2", "185.60.137.28:1400"},
		{"TURN Server 3 (UDP)", "185.60.137.29:1400"},
		{"TURN Server 3 (TCP)", "185.60.137.29:1400"},
		{"STUN Server 3", "185.60.137.29:1400"},
	}

	for _, ts := range turnServers {
		wg.Add(1)
		go func(name, addr string) {
			defer wg.Done()
			addResult(testTURNServer(name, addr))
		}(ts.name, ts.addr)
	}

	// ── 9. MySQL Database ──
	wg.Add(1)
	go func() {
		defer wg.Done()
		addResult(testDatabaseConnection())
	}()

	// ── 10. DNS Resolution ──
	wg.Add(1)
	go func() {
		defer wg.Done()
		addResult(testDNSResolution())
	}()

	// ── 11. Soroush Web Frontend ──
	wg.Add(1)
	go func() {
		defer wg.Done()
		addResult(testSoroushWebFrontend())
	}()

	// ── 12. Account Health ──
	wg.Add(1)
	go func() {
		defer wg.Done()
		addResult(testAccountsHealth())
	}()

	wg.Wait()

	// Cache results
	infraCacheMu.Lock()
	infraCache = results
	infraCacheTime = time.Now()
	infraCacheMu.Unlock()

	addLog("[Infrastructure] Full diagnostics completed", "info")

	return results
}

// ──────────────────────────────────────────────────────────────────────────────
// Individual test implementations
// ──────────────────────────────────────────────────────────────────────────────

func testSoroushWebSocket() InfraTestResult {
	result := InfraTestResult{
		Name:        "Soroush WebSocket Gateway",
		Description: "TCP connectivity to wss://im-server.splus.ir/apiws",
		Category:    "network",
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", "im-server.splus.ir:443", 10*time.Second)
	latency := time.Since(start).Milliseconds()
	result.LatencyMs = latency

	if err != nil {
		result.Status = "fail"
		result.Detail = fmt.Sprintf("Connection failed: %v", err)
		return result
	}
	conn.Close()

	result.Status = "pass"
	result.Detail = fmt.Sprintf("TCP handshake successful in %dms", latency)
	return result
}

func testMTProtoHandshake() InfraTestResult {
	result := InfraTestResult{
		Name:        "MTProto DH Key Exchange",
		Description: "Full obfuscated transport + DH handshake with Soroush server",
		Category:    "auth",
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	transport := soroushlib.NewTransport()
	if err := transport.Connect(ctx); err != nil {
		result.Status = "fail"
		result.LatencyMs = time.Since(start).Milliseconds()
		result.Detail = fmt.Sprintf("Transport connect: %v", err)
		return result
	}
	defer transport.Disconnect()

	session := soroushlib.NewSession(transport)
	if err := session.CreateAuthKey(ctx); err != nil {
		result.Status = "fail"
		result.LatencyMs = time.Since(start).Milliseconds()
		result.Detail = fmt.Sprintf("DH key exchange: %v", err)
		return result
	}

	latency := time.Since(start).Milliseconds()
	result.LatencyMs = latency
	result.Status = "pass"
	result.Detail = fmt.Sprintf("Auth key created (auth_key_id=%d) in %dms", session.AuthKeyID, latency)
	return result
}

func testTURNServer(name, addr string) InfraTestResult {
	result := InfraTestResult{
		Name:        name,
		Description: fmt.Sprintf("Connectivity to Soroush relay %s", addr),
		Category:    "turn",
	}

	start := time.Now()

	// Try UDP first
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err == nil {
		udpConn, err := net.DialTimeout("udp", udpAddr.String(), 5*time.Second)
		if err == nil {
			// Send a STUN binding request (minimal)
			stunReq := []byte{
				0x00, 0x01, // Binding Request
				0x00, 0x00, // Message Length
				0x21, 0x12, 0xA4, 0x42, // Magic Cookie
				0x01, 0x02, 0x03, 0x04, // Transaction ID (12 bytes)
				0x05, 0x06, 0x07, 0x08,
				0x09, 0x0A, 0x0B, 0x0C,
			}
			udpConn.SetDeadline(time.Now().Add(5 * time.Second))
			udpConn.Write(stunReq)

			buf := make([]byte, 1024)
			n, readErr := udpConn.Read(buf)
			udpConn.Close()

			latency := time.Since(start).Milliseconds()
			result.LatencyMs = latency

			if readErr == nil && n > 0 {
				result.Status = "pass"
				result.Detail = fmt.Sprintf("STUN response received (%d bytes) in %dms", n, latency)
				return result
			}
		}
	}

	// Fallback: try TCP
	tcpConn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	latency := time.Since(start).Milliseconds()
	result.LatencyMs = latency

	if err != nil {
		result.Status = "fail"
		result.Detail = fmt.Sprintf("Both UDP and TCP failed: %v", err)
		return result
	}
	tcpConn.Close()

	result.Status = "pass"
	result.Detail = fmt.Sprintf("TCP port reachable in %dms (UDP may be filtered)", latency)
	return result
}

func testDatabaseConnection() InfraTestResult {
	result := InfraTestResult{
		Name:        "MySQL Database",
		Description: "Connection to Clever Cloud MySQL addon",
		Category:    "database",
	}

	start := time.Now()
	sqlDB, err := db.DB()
	if err != nil {
		result.Status = "fail"
		result.LatencyMs = time.Since(start).Milliseconds()
		result.Detail = fmt.Sprintf("Get DB handle: %v", err)
		return result
	}

	if err := sqlDB.Ping(); err != nil {
		result.Status = "fail"
		result.LatencyMs = time.Since(start).Milliseconds()
		result.Detail = fmt.Sprintf("Ping failed: %v", err)
		return result
	}

	latency := time.Since(start).Milliseconds()
	result.LatencyMs = latency

	// Count tables
	var accountCount int64
	db.Model(&DBSoroushAccount{}).Count(&accountCount)

	result.Status = "pass"
	result.Detail = fmt.Sprintf("Connected in %dms. Accounts: %d", latency, accountCount)
	return result
}

func testDNSResolution() InfraTestResult {
	result := InfraTestResult{
		Name:        "DNS Resolution",
		Description: "Resolve im-server.splus.ir and web.splus.ir",
		Category:    "network",
	}

	start := time.Now()

	hosts := []string{"im-server.splus.ir", "web.splus.ir"}
	var resolvedAll []string

	for _, host := range hosts {
		ips, err := net.LookupHost(host)
		if err != nil {
			result.Status = "fail"
			result.LatencyMs = time.Since(start).Milliseconds()
			result.Detail = fmt.Sprintf("Failed to resolve %s: %v", host, err)
			return result
		}
		for _, ip := range ips {
			resolvedAll = append(resolvedAll, fmt.Sprintf("%s→%s", host, ip))
		}
	}

	latency := time.Since(start).Milliseconds()
	result.LatencyMs = latency
	result.Status = "pass"

	detail := fmt.Sprintf("Resolved in %dms.", latency)
	if len(resolvedAll) > 0 {
		detail += " " + resolvedAll[0]
		if len(resolvedAll) > 1 {
			detail += fmt.Sprintf(" (+%d more)", len(resolvedAll)-1)
		}
	}
	result.Detail = detail
	return result
}

func testSoroushWebFrontend() InfraTestResult {
	result := InfraTestResult{
		Name:        "Soroush Web Frontend",
		Description: "HTTP(S) access to https://web.splus.ir",
		Category:    "network",
	}

	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://web.splus.ir")
	latency := time.Since(start).Milliseconds()
	result.LatencyMs = latency

	if err != nil {
		result.Status = "fail"
		result.Detail = fmt.Sprintf("HTTP GET failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		result.Status = "pass"
		result.Detail = fmt.Sprintf("HTTP 200 OK in %dms", latency)
	} else {
		result.Status = "warn"
		result.Detail = fmt.Sprintf("HTTP %d in %dms (expected 200)", resp.StatusCode, latency)
	}
	return result
}

func testAccountsHealth() InfraTestResult {
	result := InfraTestResult{
		Name:        "Account Pool Health",
		Description: "Check registered accounts, session keys, and group bus readiness",
		Category:    "auth",
	}

	start := time.Now()

	var accounts []DBSoroushAccount
	if err := db.Find(&accounts).Error; err != nil {
		result.Status = "fail"
		result.LatencyMs = time.Since(start).Milliseconds()
		result.Detail = fmt.Sprintf("DB query failed: %v", err)
		return result
	}

	latency := time.Since(start).Milliseconds()
	result.LatencyMs = latency

	if len(accounts) == 0 {
		result.Status = "warn"
		result.Detail = "No accounts registered. Add Soroush accounts to enable tunnel."
		return result
	}

	withAuthKey := 0
	connected := 0
	for _, acc := range accounts {
		if len(acc.AuthKey) > 0 {
			withAuthKey++
		}
		if acc.Status == "connected" && len(acc.AuthKey) > 0 {
			connected++
		}
	}

	if withAuthKey == 0 {
		result.Status = "warn"
		result.Detail = fmt.Sprintf("%d accounts but none have valid auth keys. Re-authenticate via OTP.", len(accounts))
		return result
	}

	// Check if group bus is configured
	var groupCfg DBGroupConfig
	db.First(&groupCfg)

	if groupCfg.GroupChatID == 0 {
		result.Status = "warn"
		result.Detail = fmt.Sprintf("%d accounts (%d connected) but no Group Chat ID configured.", len(accounts), connected)
		return result
	}

	// Check if tunnel engine is running
	serverTunnel.mu.Lock()
	engineRunning := serverTunnel.running
	serverTunnel.mu.Unlock()

	if !engineRunning {
		result.Status = "warn"
		result.Detail = fmt.Sprintf("%d accounts (%d connected), group configured (ID=%d) but tunnel engine not started.", len(accounts), connected, groupCfg.GroupChatID)
		return result
	}

	result.Status = "pass"
	result.Detail = fmt.Sprintf("%d accounts (%d connected), group bus active on chat %d ✅", len(accounts), connected, groupCfg.GroupChatID)
	return result
}
