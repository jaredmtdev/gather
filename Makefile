# Define the default goal to be 'help' so that running `make` with no arguments
# automatically displays the available commands.
.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message.
	@echo "Available commands:"
	@echo
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: test
test: ## Run unit tests.
	go test -race -coverprofile=coverage.out `go list ./... | grep -v 'examples'`

.PHONY: testx
testx: ## Run unit tests multiple times.
	go test -count=5 `go list ./... | grep -v 'examples'`

.PHONY: lint
lint: ## Show linting issues.
	golangci-lint run .

.PHONY: lint-fix
lint-fix: ## Attempt to fix linting issues.
	golangci-lint run --fix .

.PHONY: coverage
coverage: ## Show coverage in html.
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html

.PHONY: vulncheck
vulncheck: ## Check for vulnerabilities in dependencies.
	govulncheck ./...
