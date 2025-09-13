.PHONY: help build proto ci clean build-examples dev stop sandbox-start sandbox-update sandbox-stop sandbox-enter sandbox-ps sandbox-logs sandbox-logs-follow

.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

###
# BUILD
##
build: ## Build manager, agent, and CLI binaries
	goreleaser build --clean --snapshot --single-target --id manager -o ./manager
	goreleaser build --clean --snapshot --single-target --id agent -o ./agent
	goreleaser build --clean --snapshot --single-target --id cli -o ./jack

proto: ## Generate protobuf files
	buf generate

ci: ## Run CI checks (tests, linting, nilaway)
	go test -race ./...
	golangci-lint run ./...
	nilaway ./...

clean: ## Clean build artifacts
	rm -rf dist/
	rm -f manager agent jack

###
# DEV
##
sandbox/tls/manager_cert.pem: sandbox/tls/create.sh
	cd sandbox/tls && bash create.sh

generate-tls: sandbox/tls/manager_cert.pem ## Generate TLS certificates for development

build-examples: ## Build plugin examples (use OPTI=1 for UPX optimization)
	rm -f plugins/*
	CGO_ENABLED=0 go build -o ./plugins/demo ./examples/plugin/...
ifeq ($(OPTI),1)
	@if [ -d "./plugins" ]; then \
		for i in ./plugins/*; do \
			if [ -f "$$i" ]; then \
				echo "Optimizing $$(basename $$i)..."; \
				upx --best --lzma "$$i"; \
			fi; \
		done; \
	fi
endif

dev: build-examples stop ## Start development environment in tmux with manager and 2 agents
	$(MAKE) generate-tls
	tmux new-session -d -s manager 'go run -race ./cmd/manager --mtls-cert "./sandbox/tls/manager_cert.pem" --mtls-key "./sandbox/tls/manager_key.pem" --mtls-agent-ca-cert "./sandbox/tls/agent_ca_cert.pem"'
	tmux new-session -d -s agent1 'go run -race ./cmd/agent --id agent1 --plugin-dir="./plugins" --mtls-cert "./sandbox/tls/agent_cert.pem" --mtls-key "./sandbox/tls/agent_key.pem" --mtls-manager-ca-cert "./sandbox/tls/manager_ca_cert.pem"'
	tmux new-session -d -s agent2 'go run -race ./cmd/agent --id agent2 --plugin-dir="./plugins" --mtls-cert "./sandbox/tls/agent_cert.pem" --mtls-key "./sandbox/tls/agent_key.pem" --mtls-manager-ca-cert "./sandbox/tls/manager_ca_cert.pem"'

stop: ## Stop all development tmux sessions
	tmux kill-session -t manager || true
	tmux kill-session -t agent1 || true
	tmux kill-session -t agent2 || true

###
# SANDBOX
##
sandbox-start: build build-examples ## Start sandbox environment with Docker
	$(MAKE) generate-tls
	cp -f jack agent manager sandbox/
	cp -rf plugins sandbox/
	docker compose --project-directory ./sandbox up -d --force-recreate

sandbox-update: build build-examples ## Update sandbox binaries without restarting
	cp -f jack agent manager sandbox/
	cp -rf plugins sandbox/

sandbox-stop: ## Stop sandbox environment
	docker compose --project-directory ./sandbox down

sandbox-enter: ## Enter sandbox manager container
	docker compose --project-directory ./sandbox exec manager bash

sandbox-ps: ## Show sandbox container status
	docker compose --project-directory ./sandbox ps -a

sandbox-logs: ## Show sandbox logs
	docker compose --project-directory ./sandbox logs

sandbox-logs-follow: ## Follow sandbox logs in real-time
	docker compose --project-directory ./sandbox logs -f
