.PHONY: help check fmt vet test proto-lint proto-gen py-lint docker-up docker-down

GO_MODULES := ./api-geteway/... ./user-service/... ./catalog-service/... ./order-service/... ./payment-service/... ./billing-service/... ./pkg/...

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Format Go source code
	@echo "==> Formatting Go files..."
	@gofmt -w .

vet: ## Run Go static analysis (go vet)
	@echo "==> Running go vet..."
	@go vet $(GO_MODULES)

test: ## Run all unit tests with race detection
	@echo "==> Running Go unit tests..."
	@go test -race $(GO_MODULES)

proto-lint: ## Lint Protobuf schemas
	@echo "==> Linting Protobuf schemas..."
	@cd proto && buf lint

proto-gen: ## Generate Go code from Protobuf
	@echo "==> Generating Protobuf code..."
	@cd proto && buf generate

py-lint: ## Run Ruff linter on audio-worker
	@echo "==> Running Ruff on audio-worker..."
	@cd audio-worker && uvx ruff check .

check: fmt proto-lint vet test py-lint ## Run all pre-push checks (fmt, lint, vet, test)
	@echo "==> [OK] All pre-push checks passed successfully!"

docker-up: ## Start local platform containers via Docker Compose
	@docker compose up -d

docker-down: ## Stop local platform containers
	@docker compose down
