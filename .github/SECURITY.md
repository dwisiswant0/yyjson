# Security Policy

`go.dw1.io/yyjson` parses untrusted input and includes transpiled native-code
logic, so parser crashes, memory-safety issues, denial-of-service inputs, and
incorrect JSON Pointer/Patch behavior are security-relevant.

## Supported Versions

This project is currently pre-v1. Until a stable release is tagged, only the
current main development line receives fixes.

## Vulnerability Handling

Security reports are accepted and taken seriously. However, because this project
is pre-v1 and has not yet committed to a stable API or release cadence,
vulnerabilities are tracked and fixed as regular bugs. No CVE will be requested
for issues found during this phase.

Once a v1.0.0 release is tagged, this policy will be revisited and standard
vulnerability disclosure practices — including CVE assignment — will apply.

## Reporting

Open a GitHub issue describing the problem:

https://github.com/dwisiswant0/yyjson/issues/new

Please include:

- affected version or commit
- operating system (`go env GOOS`) and architecture (`go env GOARCH`)
- minimal reproducer or crashing input
- expected impact, such as panic, memory growth, data corruption, or invalid
  patch behavior
- whether the input came from fuzzing, production traffic, or manual analysis

Do not include secrets or production-only sensitive data in reports.
