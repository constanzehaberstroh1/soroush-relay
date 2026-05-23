.PHONY: all install frontends client server clean help

# Colors for terminal printing
CYAN  := \033[0;36m
GREEN := \033[0;32m
RESET := \033[0m

all: install frontends client server ## Install dependencies, build panels and compile client/server Go binaries

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-15s$(RESET) %s\n", $$1, $$2}'

install: ## Install panel dependencies using Bun
	@echo "$(CYAN)Installing client-panel and server-panel dependencies...$(RESET)"
	@cd client-panel && bun install
	@cd server-panel && bun install
	@echo "$(GREEN)Dependencies successfully installed!$(RESET)"

frontends: ## Build client and server React dashboards in production mode
	@echo "$(CYAN)Building client-panel and server-panel React assets...$(RESET)"
	@cd client-panel && bun run build
	@cd server-panel && bun run build
	@echo "$(GREEN)React dashboards compiled and saved inside Go workspace!$(RESET)"

client: ## Compile Soroush client Go engine binary
	@echo "$(CYAN)Compiling soroush-client Go binary...$(RESET)"
	@mkdir -p bin
	@go build -o bin/soroush-client ./client
	@echo "$(GREEN)Binary built: bin/soroush-client$(RESET)"

server: ## Compile Soroush server Go engine binary
	@echo "$(CYAN)Compiling soroush-server Go binary...$(RESET)"
	@mkdir -p bin
	@go build -o bin/soroush-server ./server
	@echo "$(GREEN)Binary built: bin/soroush-server$(RESET)"

clean: ## Clean build artifacts, node_modules, and binaries
	@echo "$(CYAN)Cleaning build directories and binaries...$(RESET)"
	@rm -rf bin/
	@rm -rf client/dist/
	@rm -rf server/dist/
	@echo "$(GREEN)Clean completed successfully!$(RESET)"
