//go:build !(linux && (amd64 || arm64))

package yyjsonc

var _ = yyjson_go_supports_only_linux_amd64_or_arm64
