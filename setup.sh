#!/usr/bin/env bash

# Soroush WebRTC Relay Setup and Orchestration Script
# Designed with premium terminal UI components and error resilience.

set -euo pipefail

# ──────────────────────────────────────────────────────────────────────────────
# Terminal Color Codes & Styles
# ──────────────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# ──────────────────────────────────────────────────────────────────────────────
# Console Logging Helpers
# ──────────────────────────────────────────────────────────────────────────────
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    clear
    echo -e "${CYAN}${BOLD}========================================================================${NC}"
    echo -e "${PURPLE}${BOLD}             SOROUSH WEBRTC RELAY TUNNEL SYSTEM MANAGER                 ${NC}"
    echo -e "${CYAN}${BOLD}========================================================================${NC}"
    echo -e "  Phase 1 Architecture: Elegant MUI Frontends & Dual Control Planes"
    echo -e "  Uplink: Soroush apiws WebRTC Tunneling | Downlink: SOCKS5 Relay Port 8080"
    echo -e "${CYAN}========================================================================${NC}"
}

# ──────────────────────────────────────────────────────────────────────────────
# Dependency Validations
# ──────────────────────────────────────────────────────────────────────────────
check_dependency() {
    local name=$1
    local cmd=$2
    if ! command -v "$cmd" &> /dev/null; then
        log_error "Dependency '$name' ($cmd) is missing. Please install it to proceed."
        return 1
    fi
    local version
    version=$($cmd --version 2>&1 | head -n 1)
    log_success "Found $name ($version)"
    return 0
}

validate_environment() {
    echo -e "\n${BOLD}--- System Environment Validation ---${NC}"
    local failed=0
    check_dependency "Go Compiler" "go" || failed=1
    check_dependency "Bun Runtime" "bun" || failed=1
    check_dependency "Node.js" "node" || failed=1
    
    if [ $failed -ne 0 ]; then
        echo ""
        log_error "Environment verification failed. Please resolve dependencies."
        exit 1
    fi
    log_success "All dependencies validated successfully!"
}

# ──────────────────────────────────────────────────────────────────────────────
# Action Functions
# ──────────────────────────────────────────────────────────────────────────────
build_project() {
    echo -e "\n${BOLD}--- Initiating Full Project Compilation ---${NC}"
    log_info "Running local Make pipeline..."
    if make all; then
        echo ""
        log_success "All components built and compiled flawlessly!"
        log_info "Binaries located inside: bin/"
        log_info "  - Client Admin Panel + Engine: bin/soroush-client"
        log_info "  - Server Admin Panel + Engine: bin/soroush-server"
    else
        log_error "Compilation failed. Check console trace."
    fi
}

run_client() {
    echo -e "\n${BOLD}--- Launching Soroush Client Engine & Admin Panel ---${NC}"
    if [ ! -f "bin/soroush-client" ]; then
        log_warn "Binary not found. Initiating rapid compilation..."
        make client
    fi
    log_info "Starting client listener on http://127.0.0.1:8080..."
    log_info "SOCKS5 routing interface maps to Soroush WebRTC Obfuscator channel."
    exec ./bin/soroush-client
}

run_server() {
    echo -e "\n${BOLD}--- Launching Soroush Server Exit Node & Admin Panel ---${NC}"
    if [ ! -f "bin/soroush-server" ]; then
        log_warn "Binary not found. Initiating rapid compilation..."
        make server
    fi
    log_info "Starting server signaling coordinator on port 8080..."
    log_info "Routes traffic securely from client tunnels to the open web."
    # Since both default to port 8080, allow server panel port flag override if needed
    exec ./bin/soroush-server -port 8081
}

clean_project() {
    echo -e "\n${BOLD}--- Cleaning Workspace ---${NC}"
    make clean
    log_success "All temporary distributions and cached compilation binaries wiped."
}

# ──────────────────────────────────────────────────────────────────────────────
# Interactive Entry Menu
# ──────────────────────────────────────────────────────────────────────────────
show_menu() {
    print_header
    echo -e "Please select an orchestration command to run:"
    echo -e "  ${BOLD}1)${NC} Verify Environment & Setup"
    echo -e "  ${BOLD}2)${NC} Compile & Build All Components (UI + Go Binaries)"
    echo -e "  ${BOLD}3)${NC} Launch Client Admin Dashboard & Engine (Port 8080)"
    echo -e "  ${BOLD}4)${NC} Launch Server Exit Node Admin Dashboard (Port 8081)"
    echo -e "  ${BOLD}5)${NC} Clean Build Artifacts & Binaries"
    echo -e "  ${BOLD}6)${NC} Exit Manager"
    echo ""
    read -rp "Selection [1-6]: " choice

    case $choice in
        1)
            validate_environment
            ;;
        2)
            validate_environment
            build_project
            ;;
        3)
            run_client
            ;;
        4)
            run_server
            ;;
        5)
            clean_project
            ;;
        6)
            echo -e "\n${GREEN}Exiting system manager. Have a great session!${NC}"
            exit 0
            ;;
        *)
            log_warn "Invalid selection. Please input a number from 1 to 6."
            ;;
    esac
    echo -e "\nPress [ENTER] to return to the main menu..."
    read -r _
    show_menu
}

# Run the interactive menu if no CLI flags are supplied
if [ $# -eq 0 ]; then
    show_menu
else
    # Simple CLI argument routing
    case $1 in
        "build")
            validate_environment
            build_project
            ;;
        "client")
            run_client
            ;;
        "server")
            run_server
            ;;
        "clean")
            clean_project
            ;;
        *)
            echo "Usage: $0 [build|client|server|clean]"
            exit 1
            ;;
    esac
fi
