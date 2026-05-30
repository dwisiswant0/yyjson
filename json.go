package yyjson

import (
	"encoding/json"
	"io"
)

// RawMessage is an alias for [encoding/json.RawMessage].
type RawMessage = json.RawMessage

// Number is an alias for [encoding/json.Number].
type Number = json.Number

// Marshaler is an alias for [encoding/json.Marshaler].
type Marshaler = json.Marshaler

// Unmarshaler is an alias for [encoding/json.Unmarshaler].
type Unmarshaler = json.Unmarshaler

// Marshal is equivalent to [encoding/json.Marshal].
func Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal is equivalent to [encoding/json.Unmarshal].
func Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// Valid is equivalent to [encoding/json.Valid].
func Valid(data []byte) bool {
	return json.Valid(data)
}

// NewEncoder is equivalent to [encoding/json.NewEncoder].
func NewEncoder(w io.Writer) *json.Encoder {
	return json.NewEncoder(w)
}

// NewDecoder is equivalent to [encoding/json.NewDecoder].
func NewDecoder(r io.Reader) *json.Decoder {
	return json.NewDecoder(r)
}
