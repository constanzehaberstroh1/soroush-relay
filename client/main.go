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

type Account struct {
	ID          string `json:"id"`
	PhoneNumber string `json:"phoneNumber"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	LastActive  string `json:"lastActive"`
}

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
	accounts     []Account
	socksPort    int
}

var state = &ServerState{
	socksPort: 4046,
	accounts: []Account{
		{ID: "acc-1", PhoneNumber: "+989123456789", Name: "Sorush Primary", Status: "connected", LastActive: "Just now"},
		{ID: "acc-2", PhoneNumber: "+989987654321", Name: "Backup Node", Status: "idle", LastActive: "2 hours ago"},
	},
}

func main() {
	port := flag.Int("port", 8080, "Port to launch the client admin panel")
	flag.Parse()

	// Get embedded assets filesystem
	distFS, err := fs.Sub(embedFS, "dist")
	if err != nil {
		log.Fatalf("Failed to sub embed FS: %v", err)
	}

	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/start", handleStart)
	mux.HandleFunc("/api/stop", handleStop)
	mux.HandleFunc("/api/accounts", handleAccounts)

	// File Server for React Frontend
	fileServer := http.FileServer(http.FS(distFS))
	
	// SPA routing wrapper
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check if file exists in embed filesystem
		f, err := distFS.Open(r.URL.Path[1:])
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// If file doesn't exist, fall back to index.html (SPA routing)
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
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

func handleStart(w http.ResponseWriter, r *http.Request) {
	state.mu.Lock()
	if state.tunnelActive {
		state.mu.Unlock()
		http.Error(w, "Tunnel already active", http.StatusBadRequest)
		return
	}
	state.connecting = true
	state.mu.Unlock()

	// Simulate connection asynchronous delay
	go func() {
		time.Sleep(3 * time.Second)
		state.mu.Lock()
		state.connecting = false
		state.tunnelActive = true
		state.startedAt = time.Now()
		state.mu.Unlock()
		fmt.Println("[Engine] Soroush WebRTC SCTP Data Channel Tunnel established. Forwarding SOCKS5 on port 4046.")
	}()

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"message": "Tunnel connection initiated"}`))
}

func handleStop(w http.ResponseWriter, r *http.Request) {
	state.mu.Lock()
	defer state.mu.Unlock()
	state.tunnelActive = false
	state.connecting = false
	fmt.Println("[Engine] Soroush WebRTC Tunnel disconnected safely.")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Tunnel closed"}`))
}

func handleAccounts(w http.ResponseWriter, r *http.Request) {
	state.mu.Lock()
	defer state.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		json.NewEncoder(w).Encode(state.accounts)
		return
	}

	if r.Method == http.MethodPost {
		var newAcc Account
		if err := json.NewDecoder(r.Body).Decode(&newAcc); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		newAcc.ID = fmt.Sprintf("acc-%d", time.Now().UnixNano())
		newAcc.Status = "idle"
		newAcc.LastActive = "Just registered"
		state.accounts = append(state.accounts, newAcc)
		fmt.Printf("[Engine] Added Soroush credentials for account %s\n", newAcc.PhoneNumber)
		json.NewEncoder(w).Encode(newAcc)
		return
	}

	if r.Method == http.MethodDelete {
		id := r.URL.Query().Get("id")
		for i, acc := range state.accounts {
			if acc.ID == id {
				state.accounts = append(state.accounts[:i], state.accounts[i+1:]...)
				fmt.Printf("[Engine] Removed credentials for account %s\n", acc.PhoneNumber)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"message": "Account removed"}`))
				return
			}
		}
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
