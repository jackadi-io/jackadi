.PHONY: help build proto ci clean build-examples dev stop sandbox-start sandbox-update sandbox-stop sandbox-enter sandbox-ps sandbox-logs sandbox-logs-follow

.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

###
# BUILD
##
build: ## Build manager, node, and CLI binaries
	goreleaser build --clean --snapshot --single-target --id manager -o ./manager
	goreleaser build --clean --snapshot --single-target --id node -o ./node
	goreleaser build --clean --snapshot --single-target --id cli -o ./jack

proto: ## Generate protobuf files
	buf generate

ci: ## Run CI checks (tests, linting, nilaway)
	go test -race ./...
	golangci-lint run ./...
	nilaway ./...

clean: ## Clean build artifacts
	rm -rf dist/
	rm -f manager node jack

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

dev: build-examples stop ## Start development environment in tmux with manager and 2 nodes
	$(MAKE) generate-tls
	tmux new-session -d -s manager 'go run -race ./cmd/manager --mtls-cert "./sandbox/tls/manager_cert.pem" --mtls-key "./sandbox/tls/manager_key.pem" --mtls-node-ca-cert "./sandbox/tls/node_ca_cert.pem"'
	tmux new-session -d -s node1 'go run -race ./cmd/node --id node1 --plugin-dir="./plugins" --mtls-cert "./sandbox/tls/node_cert.pem" --mtls-key "./sandbox/tls/node_key.pem" --mtls-manager-ca-cert "./sandbox/tls/manager_ca_cert.pem"'
	tmux new-session -d -s node2 'go run -race ./cmd/node --id node2 --plugin-dir="./plugins" --mtls-cert "./sandbox/tls/node_cert.pem" --mtls-key "./sandbox/tls/node_key.pem" --mtls-manager-ca-cert "./sandbox/tls/manager_ca_cert.pem"'

stop: ## Stop all development tmux sessions
	tmux kill-session -t manager || true
	tmux kill-session -t node1 || true
	tmux kill-session -t node2 || true

###
# SANDBOX
##
sandbox-start: build build-examples ## Start sandbox environment with Docker
	$(MAKE) generate-tls
	cp -f jack node manager sandbox/
	cp -rf plugins sandbox/
	docker compose --project-directory ./sandbox up -d --force-recreate

sandbox-update: build build-examples ## Update sandbox binaries without restarting
	cp -f jack node manager sandbox/
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