# Define the default goal to be 'help' so that running `make` with no arguments
# automatically displays the available commands.
.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message.
	@echo "Available commands:"
	@echo
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'


_lint-install:
	command -v golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@43d03392d7dc3746fa776dbddd66dfcccff70651 # v2.4.0

_vuln-install:
	command -v govulncheck >/dev/null 2>&1 || go install golang.org/x/vuln/cmd/govulncheck@d1f380186385b4f64e00313f31743df8e4b89a77 # v1.1.4


.PHONY: test
test: ## Run unit tests.
	go test -race -p 1 -parallel 2 -shuffle=on -count=1 -coverprofile=coverage.out `go list ./... | grep -v 'examples'`

.PHONY: testx
testx: ## Run unit tests multiple times to catch flakey tests.
	go test -v -count=1000 `go list ./... | grep -v 'examples'` -failfast -timeout=40s -p 1 -parallel 2

.PHONY: fuzz
fuzz: ## Run fuzz tests to try to find edge cases.
	go test -fuzz=Fuzz -fuzztime=5m -timeout=7m

.PHONY: lint
lint: _lint-install ## Show linting issues.
	golangci-lint run ./...

.PHONY: lint-fix
lint-fix: _lint-install ## Attempt to fix linting issues.
	golangci-lint run --fix ./...

.PHONY: coverage
coverage: ## Show coverage in html.
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html

.PHONY: vulncheck
vulncheck: _vuln-install ## Check for vulnerabilities in dependencies.
	govulncheck ./...
