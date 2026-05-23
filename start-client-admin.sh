#!/usr/bin/env bash

# Soroush WebRTC Relay Client Admin Panel Startup Script
# Automatically launches the Go backend engine serving the embedded React UI.

set -euo pipefail

# Soroush cyan highlights
CYAN='\033[0;36m'
GREEN='\033[0;32m'
BOLD='\033[1m'
NC='\033[0m'

echo -e "${CYAN}${BOLD}========================================================================${NC}"
echo -e "${GREEN}${BOLD}           STARTING SOROUSH WEBRTC TUNNEL - CLIENT ADMIN PANEL          ${NC}"
echo -e "${CYAN}${BOLD}========================================================================${NC}"

# Verify compiled binary exists
if [ ! -f "bin/soroush-client" ]; then
    echo -e "${CYAN}[Engine] Compiled client binary not found. Running compilation first...${NC}"
    make client
fi

echo -e "${GREEN}[Engine] Launching panel on http://127.0.0.1:8080...${NC}"
echo -e "${CYAN}[Engine] SOCKS5 traffic local listener is mapping to port 8080.${NC}"
echo -e "Press Ctrl+C to close and terminate the tunnel."
echo ""

exec ./bin/soroush-client
