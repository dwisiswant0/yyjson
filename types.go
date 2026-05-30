package yyjson

import (
	"fmt"

	"go.dw1.io/yyjson/internal/bind"
)

// Type identifies a JSON value's primary type.
type Type uint8

const (
	// TypeNone is returned for missing, nil, or closed values.
	TypeNone Type = iota
	// TypeRaw is a raw JSON value.
	TypeRaw
	// TypeNull is a JSON null value.
	TypeNull
	// TypeBool is a JSON boolean value.
	TypeBool
	// TypeNum is a JSON number value.
	TypeNum
	// TypeStr is a JSON string value.
	TypeStr
	// TypeArr is a JSON array value.
	TypeArr
	// TypeObj is a JSON object value.
	TypeObj
)

// Subtype identifies the detailed yyjson subtype for booleans, numbers, and strings.
type Subtype uint8

const (
	// SubtypeNone is returned for values without a subtype.
	SubtypeNone Subtype = 0
	// SubtypeFalse marks a false boolean.
	SubtypeFalse Subtype = 0
	// SubtypeTrue marks a true boolean.
	SubtypeTrue Subtype = 1 << 3
	// SubtypeUint marks an unsigned integer number.
	SubtypeUint Subtype = 0
	// SubtypeSint marks a signed integer number.
	SubtypeSint Subtype = 1 << 3
	// SubtypeReal marks a floating-point number.
	SubtypeReal Subtype = 2 << 3
	// SubtypeNoEsc marks a string that does not require escaping.
	SubtypeNoEsc Subtype = 1 << 3
)

// ReadFlag configures JSON parsing behavior.
type ReadFlag uint32

const (
	// ReadNoFlag uses yyjson's default strict parser behavior.
	ReadNoFlag ReadFlag = 0
	// ReadInsitu lets yyjson retain the copied input buffer for string storage.
	ReadInsitu ReadFlag = 1 << 0
	// ReadStopWhenDone stops after the first complete JSON document.
	ReadStopWhenDone ReadFlag = 1 << 1
	// ReadAllowTrailingCommas allows trailing commas in arrays and objects.
	ReadAllowTrailingCommas ReadFlag = 1 << 2
	// ReadAllowComments allows C-style comments.
	ReadAllowComments ReadFlag = 1 << 3
	// ReadAllowInfAndNaN allows Infinity and NaN number literals.
	ReadAllowInfAndNaN ReadFlag = 1 << 4
	// ReadNumberAsRaw stores numbers as raw values.
	ReadNumberAsRaw ReadFlag = 1 << 5
	// ReadAllowInvalidUnicode allows invalid Unicode input.
	ReadAllowInvalidUnicode ReadFlag = 1 << 6
	// ReadBignumAsRaw stores oversized numbers as raw values.
	ReadBignumAsRaw ReadFlag = 1 << 7
	// ReadAllowBOM allows a UTF-8 byte order mark.
	ReadAllowBOM ReadFlag = 1 << 8
	// ReadAllowExtNumber allows JSON5-style extended numbers.
	ReadAllowExtNumber ReadFlag = 1 << 9
	// ReadAllowExtEscape allows extended escape sequences.
	ReadAllowExtEscape ReadFlag = 1 << 10
	// ReadAllowExtWhitespace allows extended whitespace characters.
	ReadAllowExtWhitespace ReadFlag = 1 << 11
	// ReadAllowSingleQuotedStr allows single-quoted strings.
	ReadAllowSingleQuotedStr ReadFlag = 1 << 12
	// ReadAllowUnquotedKey allows unquoted object keys.
	ReadAllowUnquotedKey ReadFlag = 1 << 13
	// ReadJSON5 enables the JSON5-compatible read flags.
	ReadJSON5 ReadFlag = ReadAllowTrailingCommas | ReadAllowComments | ReadAllowInfAndNaN | ReadAllowExtNumber | ReadAllowExtEscape | ReadAllowExtWhitespace | ReadAllowSingleQuotedStr | ReadAllowUnquotedKey
)

// Read error codes returned by yyjson.
const (
	ReadErrorInvalidParameter    uint32 = 1
	ReadErrorMemoryAllocation    uint32 = 2
	ReadErrorEmptyContent        uint32 = 3
	ReadErrorUnexpectedContent   uint32 = 4
	ReadErrorUnexpectedEnd       uint32 = 5
	ReadErrorUnexpectedCharacter uint32 = 6
	ReadErrorJSONStructure       uint32 = 7
	ReadErrorInvalidComment      uint32 = 8
	ReadErrorInvalidNumber       uint32 = 9
	ReadErrorInvalidString       uint32 = 10
	ReadErrorLiteral             uint32 = 11
	ReadErrorFileOpen            uint32 = 12
	ReadErrorFileRead            uint32 = 13
	ReadErrorMore                uint32 = 14
)

// WriteFlag configures JSON serialization behavior.
type WriteFlag uint32

const (
	// WriteNoFlag uses yyjson's default writer behavior.
	WriteNoFlag WriteFlag = 0
	// WritePretty pretty-prints with four-space indentation.
	WritePretty WriteFlag = 1 << 0
	// WriteEscapeUnicode escapes non-ASCII Unicode characters.
	WriteEscapeUnicode WriteFlag = 1 << 1
	// WriteEscapeSlashes escapes forward slashes.
	WriteEscapeSlashes WriteFlag = 1 << 2
	// WriteAllowInfAndNaN writes Infinity and NaN literals.
	WriteAllowInfAndNaN WriteFlag = 1 << 3
	// WriteInfAndNaNAsNull writes Infinity and NaN as null.
	WriteInfAndNaNAsNull WriteFlag = 1 << 4
	// WriteAllowInvalidUnicode allows invalid Unicode output.
	WriteAllowInvalidUnicode WriteFlag = 1 << 5
	// WritePrettyTwoSpaces pretty-prints with two-space indentation.
	WritePrettyTwoSpaces WriteFlag = 1 << 6
	// WriteNewlineAtEnd appends a trailing newline.
	WriteNewlineAtEnd WriteFlag = 1 << 7
	// WriteFPToFloat serializes floating-point numbers as single-precision values.
	WriteFPToFloat WriteFlag = 1 << (32 - 5)
)

// WriteFPToFixed returns a flag that writes floating-point numbers in fixed-point notation.
// It returns [WriteNoFlag] when prec is outside yyjson's documented 1 to 15 range.
func WriteFPToFixed(prec int) WriteFlag {
	if prec < 1 || prec > 15 {
		return WriteNoFlag
	}
	return WriteFlag(uint32(prec) << (32 - 4))
}

// ReadError describes a JSON read failure.
type ReadError struct {
	Code      uint32
	Message   string
	Position  uint64
	Operation string
	Path      string
}

// Error returns the formatted read error.
func (e *ReadError) Error() string {
	if e == nil {
		return ""
	}
	msg := fmt.Sprintf("yyjson %s failed", e.Operation)
	if e.Path != "" {
		msg += " for " + e.Path
	}
	return fmt.Sprintf("%s: code=%d pos=%d: %s", msg, e.Code, e.Position, e.Message)
}

// WriteError describes a JSON write failure.
type WriteError struct {
	Code      uint32
	Message   string
	Operation string
	Path      string
}

// Error returns the formatted write error.
func (e *WriteError) Error() string {
	if e == nil {
		return ""
	}
	msg := fmt.Sprintf("yyjson %s failed", e.Operation)
	if e.Path != "" {
		msg += " for " + e.Path
	}
	return fmt.Sprintf("%s: code=%d: %s", msg, e.Code, e.Message)
}

// PointerError describes a JSON Pointer operation failure.
type PointerError struct {
	Code      uint32
	Message   string
	Position  uint64
	Operation string
	Path      string
}

// Error returns the formatted pointer error.
func (e *PointerError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("yyjson pointer %s failed at %q: code=%d pos=%d: %s", e.Operation, e.Path, e.Code, e.Position, e.Message)
}

// PatchError describes a JSON Patch operation failure.
type PatchError struct {
	Code    uint32
	Index   uint64
	Message string
	Pointer PointerError
}

// Error returns the formatted patch error.
func (e *PatchError) Error() string {
	if e == nil {
		return ""
	}
	if e.Pointer.Code != 0 {
		return fmt.Sprintf("yyjson patch failed: code=%d op=%d: %s: %s", e.Code, e.Index, e.Message, e.Pointer.Error())
	}
	return fmt.Sprintf("yyjson patch failed: code=%d op=%d: %s", e.Code, e.Index, e.Message)
}

// Version returns the linked yyjson version number.
func Version() uint32 {
	return bind.Version()
}

func readFlags(flags []ReadFlag) uint32 {
	var out ReadFlag
	for _, flag := range flags {
		out |= flag
	}
	return uint32(out)
}

func writeFlags(flags []WriteFlag) uint32 {
	var out WriteFlag
	for _, flag := range flags {
		out |= flag
	}
	return uint32(out)
}

func readError(err *bind.ReadError, op, path string) error {
	if err == nil {
		return nil
	}
	return &ReadError{Code: err.Code, Message: err.Message, Position: err.Position, Operation: op, Path: path}
}

func writeError(err *bind.WriteError, op, path string) error {
	if err == nil {
		return nil
	}
	return &WriteError{Code: err.Code, Message: err.Message, Operation: op, Path: path}
}

func pointerError(err *bind.PointerError, op, path string) error {
	if err == nil {
		return nil
	}
	return &PointerError{Code: err.Code, Message: err.Message, Position: err.Position, Operation: op, Path: path}
}

func patchError(err *bind.PatchError) error {
	if err == nil {
		return nil
	}
	return &PatchError{
		Code:    err.Code,
		Index:   err.Index,
		Message: err.Message,
		Pointer: PointerError{
			Code:      err.Pointer.Code,
			Message:   err.Pointer.Message,
			Position:  err.Pointer.Position,
			Operation: "patch",
		},
	}
}
