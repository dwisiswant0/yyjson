# Contributing

Thanks for improving `go.dw1.io/yyjson`. This package wraps transpiled yyjson
code behind a Go API, so changes should keep generated internals separate
from the public surface.

Before making changes, read [DEVELOPMENT.md](/docs/DEVELOPMENT.md) for the local
setup, Makefile targets, generation workflow, and verification commands.

## Guidelines

- Keep public API changes small and compatible unless the change is explicitly
  intended to happen before a release.
- Preserve yyjson behavior unless a difference is documented and tested.
- Do not edit generated files in [`internal/yyjsonc`](/internal/yyjsonc) by
  hand. Update the generator or source inputs, then run `make generate`.
- Keep handwritten wrapper code idiomatic Go and avoid exposing generated C-like
  names, ccgo runtime details, or unsafe pointer conventions.
- Add or update tests for behavior changes. Fixture-backed tests require the
  `yyjson` submodule; `make test` initializes it automatically.
- Run `make check` before submitting changes. Run `make lint` when
  `golangci-lint` is available.

See [LIMITATIONS.md](/docs/LIMITATIONS.md) for current platform and tooling limits.
