GO ?= go
GOLANGCI_LINT ?= golangci-lint
FUZZTIME ?= 5m
FUZZ_TARGETS = FuzzReadWriteRoundTrip FuzzPointer FuzzPatchAndMergePatch FuzzIncrementalRead
GENYYJSON_TARGETS ?= $(shell $(GO) env GOOS)/$(shell $(GO) env GOARCH)
GENYYJSON_TARGETS_FLAG := $(if $(strip $(GENYYJSON_TARGETS)),-targets "$(GENYYJSON_TARGETS)")

.DEFAULT_GOAL := test

.PHONY: submodule generate generate-check test test-cgo0 test-race tests fuzz-smoke fuzz vet check lint bench-compare

submodule:
	@git submodule update --init yyjson

generate: submodule
	@$(GO) generate ./...

generate-check: submodule
	@$(GO) run ./internal/cmd/genyyjson $(GENYYJSON_TARGETS_FLAG) -check

test: submodule
	@$(GO) test ./...

test-cgo0: submodule
	@CGO_ENABLED=0 $(GO) test ./...

test-race: submodule
	@$(GO) test -race ./...

tests: test test-cgo0 test-race

fuzz-smoke: submodule
	@$(GO) test -run '^$$' -fuzz=FuzzReadWriteRoundTrip -fuzztime=10s .

fuzz: submodule
	@for target in $(FUZZ_TARGETS); do \
		$(GO) test -run '^$$' -fuzz="^$$target$$" -fuzztime=$(FUZZTIME) . || exit 1; \
	done

vet:
	@$(GO) run ./internal/cmd/vetyyjson ./...

check: generate-check vet tests fuzz-smoke

lint:
	$(GOLANGCI_LINT) run ./...

bench-compare:
	@$(MAKE) -C benchmarks
