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

type TunnelStatus struct {
	Active     bool      `json:"active"`
	Connecting bool      `json:"connecting"`
	SocksPort  int       `json:"socksPort"`
	Uptime     string    `json:"uptime"`
	StartedAt  time.Time `json:"startedAt"`
}

type ServerState struct {
	mu           sync.RWMutex
	tunnelActive bool
	connecting   bool
	startedAt    time.Time
	socksPort    int
}

var state = &ServerState{
	socksPort: 4046,
}

// Global logs slice shared across files
var (
	logsMu     sync.Mutex
	globalLogs = []map[string]string{
		{"timestamp": time.Now().Format("15:04:05"), "type": "info", "message": "Soroush WebRTC engine initialized."},
		{"timestamp": time.Now().Format("15:04:05"), "type": "success", "message": "Ready to establish P2P voice call channel wrapper"},
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

	port := flag.Int("port", defaultPort, "Port to launch the client admin panel")
	flag.Parse()

	// 1. Initialize SQLite Database
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

	// Protected Admin endpoints
	mux.HandleFunc("/api/admin/me", JWTMiddleware(handleAdminMe))
	mux.HandleFunc("/api/status", JWTMiddleware(handleStatus))
	mux.HandleFunc("/api/start", JWTMiddleware(handleTunnelStart))
	mux.HandleFunc("/api/stop", JWTMiddleware(handleTunnelStop))
	mux.HandleFunc("/api/tunnel/status", JWTMiddleware(handleTunnelStatus))
	mux.HandleFunc("/api/tunnel/config", JWTMiddleware(handleTunnelConfig))
	mux.HandleFunc("/api/accounts", JWTMiddleware(handleAccounts))
	mux.HandleFunc("/api/accounts/request-otp", JWTMiddleware(handleRequestOTP))
	mux.HandleFunc("/api/accounts/verify-otp", JWTMiddleware(handleVerifyOTP))
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
	fmt.Printf(" Soroush WebRTC Relay CLIENT Panel launched on http://%s\n", addr)
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

// Re-defining handlers to support JWT protection and use DB

func handleStatus(w http.ResponseWriter, r *http.Request) {
	state.mu.RLock()
	defer state.mu.RUnlock()

	uptime := "0s"
	if state.tunnelActive {
		uptime = time.Since(state.startedAt).Round(time.Second).String()
	}

	res := TunnelStatus{
		Active:     state.tunnelActive,
		Connecting: state.connecting,
		SocksPort:  state.socksPort,
		Uptime:     uptime,
		StartedAt:  state.startedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// handleStart and handleStop are now handled by tunnel.go
// (handleTunnelStart and handleTunnelStop)

// DB-backed Soroush Accounts handler
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

		addLog(fmt.Sprintf("Account %s removed from Soroush credential library.", acc.PhoneNumber), "warn")
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
