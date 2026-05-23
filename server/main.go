package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
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

func main() {
	port := flag.Int("port", 8080, "Port to launch the server admin panel")
	flag.Parse()

	// Get embedded assets filesystem
	distFS, err := fs.Sub(embedFS, "dist")
	if err != nil {
		log.Fatalf("Failed to sub embed FS: %v", err)
	}

	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/stats", handleStats)
	mux.HandleFunc("/api/clients", handleClients)
	mux.HandleFunc("/api/config", handleConfig)
	
	// Soroush WebRTC Signaling Channel path (/apiws WebSocket route)
	mux.HandleFunc("/apiws", handleSignalingWS)

	// File Server for React Frontend
	fileServer := http.FileServer(http.FS(distFS))

	// SPA routing wrapper
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		f, err := distFS.Open(r.URL.Path[1:])
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
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
		"cpu":          12.4,
		"memory":       34.2,
		"uptime":       "6d 18h 22m",
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
		fmt.Printf("[Engine] Server config updated. Gateway port: %d, forwarding exit SOCKS5 to %s:%d\n",
			state.config.ServerPort, state.config.SocksHost, state.config.SocksPort)
		json.NewEncoder(w).Encode(state.config)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// Soroush Signaling WebSocket Handler placeholder
func handleSignalingWS(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[SignalingWS] Soroush client handshake requested from IP %s. Upgrading connection...\n", r.RemoteAddr)
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Soroush signaling websocket protocol requires Go phase 2 implementation."))
}
