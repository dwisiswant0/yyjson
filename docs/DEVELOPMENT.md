# Development

The root `Makefile` initializes the upstream yyjson submodule for targets that
need fixture or source files. To initialize it directly:

```bash
make submodule
```

Run the default root tests:

```bash
make test
```

Run the standard non-lint verification set:

```bash
make check
```

Transpiled yyjson code lives in `internal/yyjsonc`. Regenerate it with:

```bash
make generate
```

To check whether generated output is current:

```bash
make generate-check
```

`generate-check` defaults to the current Go host target. Override
`GENYYJSON_TARGETS` to check one or more explicit generated targets:

```bash
make generate-check GENYYJSON_TARGETS=linux/arm64
make generate-check GENYYJSON_TARGETS=linux/amd64,linux/arm64
```

To probe a specific target without touching checked-in generated files:

```bash
tmp=$(mktemp -d)
go run ./internal/cmd/genyyjson -targets darwin/arm64 -out-dir "$tmp"
```

Generated output is checked in CI across the OS and Go-version matrix defined in
[tests workflow](https://github.com/dwisiswant0/yyjson/actions/workflows/tests.yaml).
Each runner executes `make generate-check`, so it checks the runner's native
`GOOS/GOARCH` target with that matrix Go toolchain. Generated output should be
stable across the matrix; if a Go, ccgo, libc, or runner update changes output,
fix the generator normalization or checked-in generated files instead of
gating the check.

[`modernc.org/ccgo`](https://pkg.go.dev/modernc.org/ccgo) also emits host C
compiler definitions as unused Go declarations. The generator drops unused
top-level `const __...`, `const FP_FAST_...`, and `type T__predefined_...`
declarations so routine compiler and runner architecture differences do not
make `make generate-check` report stale output.

Additional development targets:

```bash
make vet              # run the repo go/analysis vet suite
make test-cgo0        # run root tests with CGO_ENABLED=0
make test-race        # run tests with the race detector and default checkptr settings
make fuzz-smoke       # run a short fuzzing smoke test
make fuzz             # run all fuzz targets; override FUZZTIME for campaign length
make lint             # requires golangci-lint
make bench-compare    # requires benchstat
```

`make check` runs `generate-check`, `vet`, normal tests, `CGO_ENABLED=0` tests,
race tests with default checkptr settings, and a short fuzz smoke test. CI runs
the same test steps for every OS and Go version in the workflow matrix.

Before stable release candidates, run a longer fuzz campaign such as:

```bash
make fuzz FUZZTIME=30m
```

`make vet` runs `internal/cmd/vetyyjson`, a `go/analysis` multichecker with
standard vet-style analyzers for handwritten packages plus yyjson's
unsafe-pointer policy. Analyzer reports from ccgo-generated `internal/yyjsonc`
code are intentionally outside the vet gate; generated code is covered by
`generate-check`, builds, and tests. The policy permits the single handwritten
`uintptr` to [`unsafe.Pointer`](https://pkg.go.dev/unsafe#Pointer) conversion in
`internal/bind.cPointer`.

The benchmark module includes JSON decode/encode, read-all, round-trip, and
explicit DOM read/write suites. It also includes a `native-yyjson` cgo baseline
compiled from the checked-out upstream C source. With `CGO_ENABLED=0`, the
native baseline benchmark cases are skipped.

The `native-yyjson` baseline intentionally pays the same benchmark-boundary
costs as the public Go package where practical: it copies input before parsing
and copies write output into a Go `[]byte`. Its `B/op` and `allocs/op` numbers
still do not include yyjson's internal C heap allocations.
