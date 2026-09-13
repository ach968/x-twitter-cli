GO ?= go
BINARY ?= twt
CHROMIUM ?= /usr/bin/chromium

.DEFAULT_GOAL := help

.PHONY: help build run auth-browser test test-race test-browser test-browser-race test-live capture-search-evidence fmt fmt-check vet check check-browser clean

help: ## Show available targets.
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "%-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the twt executable.
	$(GO) build -o $(BINARY) ./cmd/twt

run: ## Run the current twt entrypoint without keeping a binary.
	$(GO) run ./cmd/twt

auth-browser: ## Temporarily open the persisted application profile for interactive login.
	$(CHROMIUM) --user-data-dir="$$HOME/.local/state/x-twitter-cli3/chromium-profile" https://x.com/home

test: ## Run deterministic unit tests.
	$(GO) test -v ./...

test-race: ## Run deterministic unit tests with the race detector.
	$(GO) test -race -v ./...

test-browser: ## Run local Chromium integration tests.
	TWT_CHROMIUM_EXECUTABLE="$(CHROMIUM)" $(GO) test -v -tags=browser -count=1 ./...

test-browser-race: ## Run local Chromium integration tests with the race detector.
	TWT_CHROMIUM_EXECUTABLE="$(CHROMIUM)" $(GO) test -race -v -tags=browser -count=1 ./...

test-live: ## Run all opt-in live X tests using the persisted application profile.
	TWT_LIVE_X=1 TWT_CHROMIUM_EXECUTABLE="$(CHROMIUM)" $(GO) test -tags=live -v -count=1 -timeout=2m ./test

capture-search-evidence: ## Save one SearchTimeline payload (requires query; product defaults to Top).
	@test -n "$$TWT_SEARCH_QUERY" || (echo "TWT_SEARCH_QUERY is required" >&2; exit 2)
	TWT_CAPTURE_SEARCH_EVIDENCE=1 $(GO) test -tags=live -v -count=1 -timeout=2m -run '^TestCaptureSearchTimelineEvidence$$' ./test

fmt: ## Format all Go packages.
	$(GO) fmt ./...

fmt-check: ## Fail if any Go source needs formatting.
	@test -z "$$(gofmt -l $$(find . -type f -name '*.go' -not -path './vendor/*'))"

vet: ## Run Go's static analyzer.
	$(GO) vet ./...

check: fmt-check vet test-race ## Run the deterministic pre-commit checks.

check-browser: check test-browser-race ## Run all deterministic checks, including Chromium integration.

clean: ## Remove locally built output.
	$(RM) $(BINARY)
