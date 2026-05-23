package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

//go:embed dist
var embedFS embed.FS

type ClientConnection struct {
	ID        string    `json:"id"`
	IP        string    `json:"ip"`
	Phone     string    `json:"phone"`
	Uptime    string    `json:"uptime"`
	Bandwidth string    `json:"bandwidth"`
	Status    string    `json:"status"`
	Connected time.Time `json:"connected"`
}

type Config struct {
	ServerPort     int    `json:"serverPort"`
	SocksHost      string `json:"socksHost"`
	SocksPort      int    `json:"socksPort"`
	BandwidthLimit int    `json:"bandwidthLimit"`
}

type ServerState struct {
	mu      sync.RWMutex
	config  Config
	clients []ClientConnection
}

var state = &ServerState{
	config: Config{
		ServerPort:     8080,
		SocksHost:      "127.0.0.1",
		SocksPort:      1080,
		BandwidthLimit: 100,
	},
	clients: []ClientConnection{
		{ID: "cli-1", IP: "192.168.10.12", Phone: "+989123456789", Uptime: "1h 14m", Bandwidth: "12.4 MB/s", Status: "active", Connected: time.Now().Add(-1 * time.Hour)},
		{ID: "cli-2", IP: "192.168.10.25", Phone: "+989987654321", Uptime: "42m", Bandwidth: "2.1 MB/s", Status: "active", Connected: time.Now().Add(-42 * time.Minute)},
	},
}

// Global signaling logs slice shared across files
var (
	logsMu     sync.Mutex
	globalLogs = []map[string]string{
		{"timestamp": time.Now().Format("15:04:05"), "type": "info", "message": "Soroush exit node engine initialized."},
		{"timestamp": time.Now().Format("15:04:05"), "type": "success", "message": "WebRTC Signaling handler active on path /apiws"},
		{"timestamp": time.Now().Format("15:04:05"), "type": "info", "message": "Ready to route decapsulated tunnel traffic to open Web"},
	}
)

func addLog(message string, typeStr string) {
	logsMu.Lock()
	defer logsMu.Unlock()
	globalLogs = append([]map[string]string{
		{"timestamp": time.Now().Format("15:04:05"), "type": typeStr, "message": message},
	}, globalLogs...)
	if len(globalLogs) > 100 {
		globalLogs = globalLogs[:100]
	}
}

func main() {
	defaultPort := 8080
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			defaultPort = p
		}
	}

	port := flag.Int("port", defaultPort, "Port to launch the server admin panel")
	flag.Parse()

	// 1. Initialize SQLite database for Server exit node
	initDB()

	// Get embedded assets filesystem
	var distFS fs.FS
	if _, err := embedFS.ReadDir("dist"); err == nil {
		subFS, err := fs.Sub(embedFS, "dist")
		if err != nil {
			log.Fatalf("Failed to sub embed FS: %v", err)
		}
		distFS = subFS
	} else {
		// Fallback to local disk if embedded assets are not compiled yet during local dev
		distFS = os.DirFS("dist")
	}

	mux := http.NewServeMux()

	// Authentication API endpoints (Public)
	mux.HandleFunc("/api/admin/login", handleAdminLogin)

	// Public endpoint — no auth required (for client connectivity test)
	mux.HandleFunc("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(fmt.Sprintf(`{"status":"ok","server":"soroush-relay","version":"3.0","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))))
	})

	// Protected Admin endpoints
	mux.HandleFunc("/api/admin/me", JWTMiddleware(handleAdminMe))
	mux.HandleFunc("/api/stats", JWTMiddleware(handleStats))
	mux.HandleFunc("/api/clients", JWTMiddleware(handleClients))
	mux.HandleFunc("/api/config", JWTMiddleware(handleConfig))
	mux.HandleFunc("/api/accounts", JWTMiddleware(handleAccounts))
	mux.HandleFunc("/api/accounts/request-otp", JWTMiddleware(handleRequestOTP))
	mux.HandleFunc("/api/accounts/verify-otp", JWTMiddleware(handleVerifyOTP))
	mux.HandleFunc("/api/accounts/set-role", JWTMiddleware(handleSetAccountRole))
	mux.HandleFunc("/api/tunnel/start", JWTMiddleware(handleServerTunnelStart))
	mux.HandleFunc("/api/tunnel/stop", JWTMiddleware(handleServerTunnelStop))
	mux.HandleFunc("/api/tunnel/status", JWTMiddleware(handleServerTunnelStatus))
	mux.HandleFunc("/api/infra/test", JWTMiddleware(handleInfraTest))
	mux.HandleFunc("/api/infra/status", JWTMiddleware(handleInfraStatus))
	mux.HandleFunc("/api/logs", JWTMiddleware(handleGetLogs))

	// CORS Preflight handler
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, `{"error":"Not found"}`, http.StatusNotFound)
	})

	// Soroush WebRTC Signaling Channel path (/apiws WebSocket route - handled separately or via auth)
	mux.HandleFunc("/apiws", handleSignalingWS)

	// File Server for React Frontend
	fileServer := http.FileServer(http.FS(distFS))

	// SPA routing wrapper
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If requesting an API, fallback to 404
		if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
			return
		}

		f, err := distFS.Open(r.URL.Path[1:])
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	fmt.Println("==========================================================")
	fmt.Printf(" Soroush WebRTC Relay SERVER Panel launched on http://%s\n", addr)
	fmt.Println("==========================================================")

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server listen failed: %v", err)
	}
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	state.mu.RLock()
	defer state.mu.RUnlock()

	stats := map[string]interface{}{
		"cpu":           12.4,
		"memory":        34.2,
		"uptime":        "6d 18h 22m",
		"activeTunnels": len(state.clients),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func handleClients(w http.ResponseWriter, r *http.Request) {
	state.mu.RLock()
	defer state.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state.clients)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	state.mu.Lock()
	defer state.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		json.NewEncoder(w).Encode(state.config)
		return
	}

	if r.Method == http.MethodPost {
		var newConfig Config
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		state.config = newConfig
		addLog(fmt.Sprintf("Server config updated. Gateway port: %d, forwarding exit SOCKS5 to %s:%d",
			state.config.ServerPort, state.config.SocksHost, state.config.SocksPort), "success")
		json.NewEncoder(w).Encode(state.config)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Soroush Signaling WebSocket Handler
func handleSignalingWS(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[SignalingWS] Soroush client handshake requested from IP %s.\n", r.RemoteAddr)
	// The real signaling is handled through the MTProto session in tunnel.go
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Signaling handled via MTProto tunnel engine."))
}

// DB-backed Soroush Accounts handler for server side exit routing
func handleAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		var accounts []DBSoroushAccount
		if err := db.Order("created_at desc").Find(&accounts).Error; err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Database read failure: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(accounts)
		return
	}

	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, `{"error":"Missing account id query parameter"}`, http.StatusBadRequest)
			return
		}

		var acc DBSoroushAccount
		if err := db.Where("id = ?", id).First(&acc).Error; err != nil {
			http.Error(w, `{"error":"Account not found"}`, http.StatusNotFound)
			return
		}

		if err := db.Delete(&acc).Error; err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"Database delete failure: %s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		addLog(fmt.Sprintf("Account %s removed from Soroush exit configuration.", acc.PhoneNumber), "warn")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Account removed successfully"})
		return
	}

	http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
}

func handleGetLogs(w http.ResponseWriter, r *http.Request) {
	logsMu.Lock()
	defer logsMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(globalLogs)
}
