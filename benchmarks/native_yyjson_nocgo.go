//go:build !cgo

package benchmarks

import "testing"

func benchmarkReadNativeYYJSON(b *testing.B, data []byte) {
	b.Skip("native yyjson benchmark requires cgo")
}

func benchmarkReadAllNativeYYJSON(b *testing.B, data []byte) {
	b.Skip("native yyjson benchmark requires cgo")
}

func benchmarkWriteNativeYYJSON(b *testing.B, data []byte) {
	b.Skip("native yyjson benchmark requires cgo")
}

func benchmarkRoundTripNativeYYJSON(b *testing.B, data []byte) {
	b.Skip("native yyjson benchmark requires cgo")
}

func benchmarkDOMReadNativeYYJSON(b *testing.B, data []byte) {
	b.Skip("native yyjson benchmark requires cgo")
}

func benchmarkDOMWriteNativeYYJSON(b *testing.B, data []byte) {
	b.Skip("native yyjson benchmark requires cgo")
}
