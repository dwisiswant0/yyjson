SHELL := /bin/bash

GO ?= go
BENCHSTAT ?= $(GO) tool benchstat
COUNT ?= 10
BENCHTIME ?= 2s
BENCHFLAGS = -run '^$$' -benchmem -count=$(COUNT) -benchtime=$(BENCHTIME)
PACKAGES = std json-iterator segmentio-json go-json sonic simdjson native-yyjson yyjson
COMPARE_PACKAGES = std json-iterator segmentio-json go-json sonic simdjson native-yyjson
BASELINE = yyjson
ARTIFACT ?= benchmark-results.tar

.PHONY: all submodule bench benchstats artifact extract-artifact clean FORCE
.DELETE_ON_ERROR:

all:
	@$(MAKE) bench
	@$(MAKE) benchstats

submodule:
	@git -C .. submodule update --init yyjson

bench: submodule
	@$(MAKE) $(PACKAGES)

benchstats: SEP = $(shell printf '=%.0s' {1..30})
benchstats:
	@for package in $(COMPARE_PACKAGES); do \
		printf '%s %s vs %s %s\n' "$(SEP)" "$$package" "$(BASELINE)" "$(SEP)"; \
		$(BENCHSTAT) $$package $(BASELINE); \
	done

artifact:
	@for package in $(PACKAGES); do \
		test -f "$$package"; \
		test ! -L "$$package"; \
	done
	@rm -f $(ARTIFACT)
	@tar -cf $(ARTIFACT) $(PACKAGES)

extract-artifact:
	@tar -xf $(ARTIFACT)

$(PACKAGES): FORCE
	@rm -f "$@" "$@.tmp"
	@tmp="$@.tmp"; \
	set -o pipefail; \
	if ! $(GO) test $(BENCHFLAGS) -bench 'Benchmark.*/.*/^$@$$' | tee "$$tmp"; then \
		rm -f "$$tmp" "$@"; \
		exit 1; \
	fi; \
	if ! sed 's#/$@-#-#' "$$tmp" > "$@"; then \
		rm -f "$$tmp" "$@"; \
		exit 1; \
	fi; \
	rm -f "$$tmp"

clean:
	@rm -f $(PACKAGES) $(addsuffix .tmp,$(PACKAGES)) $(ARTIFACT)

FORCE:
