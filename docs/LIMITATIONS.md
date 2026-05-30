# Limitations

This package currently transpiles yyjson to cgo-free Go for `linux/amd64`
and `linux/arm64`. Unsupported platforms fail during build with an explicit
internal package error.

Darwin generation is currently blocked by ccgo reporting unsupported
`_Float128` declarations for the darwin target.

Windows generation currently produces non-buildable code because ccgo-generated
`Tsize_t` values are not consistently assignable to the
[`modernc.org/libc`](https://pkg.go.dev/modernc.org/libc) size types expected by
memory/string helpers.

To probe un/supported targets without touching checked-in generated files:

```bash
tmp=$(mktemp -d)
go run ./internal/cmd/genyyjson -targets "$(go env GOOS)/$(go env GOARCH)" -out-dir "$tmp"
```

The generator currently does not expose yyjson's compile-time feature toggles.
Generated Linux targets keep `YYJSON_DISABLE_FAST_FP_CONV=0` and undefine
compiler `__int128` support, so yyjson's fast floating-point conversion is built
using its portable 64-bit 128-bit arithmetic path.

Raw `go vet ./...` reports known diagnostics inside ccgo-generated code:
`self-assignment`, `unreachable code`, and
[`unsafe.Pointer`](https://pkg.go.dev/unsafe#Pointer) usage from C pointer
translation. The supported vet gate is `make vet`, which runs the repo
`go/analysis` multichecker instead of parsing `go vet` text output. It treats
generated `internal/yyjsonc` as generator-owned output, so analyzer reports
from that package are outside the vet gate. The yyjson-specific unsafe-pointer
policy permits only the handwritten `internal/bind.cPointer` conversion from
`uintptr` to `unsafe.Pointer`. Generated code verification is based on
reproducible `make generate-check` output, `make vet`, and the
supported-platform test matrix.

[`NewIncrReader`](https://pkg.go.dev/go.dw1.io/yyjson#NewIncrReader) currently
exposes a narrow incremental API: callers must provide a maximum input size up
front, chunks are copied into an internal buffer, and each reader is single-use
after completion or a non-recoverable parse error.

This package does not impose a global maximum document size. For untrusted
inputs, callers should enforce request or file size limits before calling
[`Read`](https://pkg.go.dev/go.dw1.io/yyjson#Read),
[`ReadAll`](https://pkg.go.dev/go.dw1.io/yyjson#ReadAll), or
[`ReadFile`](https://pkg.go.dev/go.dw1.io/yyjson#ReadFile).
Use [`NewIncrReader`](https://pkg.go.dev/go.dw1.io/yyjson#NewIncrReader) when an
explicit maximum input size is useful at the parser boundary.

Immutable [`Doc`](https://pkg.go.dev/go.dw1.io/yyjson#Doc) and
[`Val`](https://pkg.go.dev/go.dw1.io/yyjson#Val) read operations may be used
concurrently while the owning document remains open.
[`Close`](https://pkg.go.dev/go.dw1.io/yyjson#Doc.Close) must not race with any
operation. Mutable documents, mutable values, and incremental readers are
single-owner values and are not safe for concurrent use.
