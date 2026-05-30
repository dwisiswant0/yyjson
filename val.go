package yyjson

import (
	"fmt"
	"runtime"

	"go.dw1.io/yyjson/internal/bind"
)

// Val is an immutable JSON value owned by a [Doc].
type Val struct {
	ptr bind.Val
	doc *Doc
}

// Member is an immutable object member.
type Member struct {
	Key   *Val
	Value *Val
}

func (v *Val) valid() bool {
	return v != nil && v.ptr != 0 && v.doc != nil && !v.doc.closed
}

// Type returns the value's primary JSON type.
func (v *Val) Type() Type {
	if !v.valid() {
		return TypeNone
	}
	typ := Type(bind.ValType(v.ptr))
	runtime.KeepAlive(v.doc)
	return typ
}

// Subtype returns the value's yyjson subtype.
func (v *Val) Subtype() Subtype {
	if !v.valid() {
		return SubtypeNone
	}
	subtype := Subtype(bind.ValSubtype(v.ptr))
	runtime.KeepAlive(v.doc)
	return subtype
}

// TypeDescription returns yyjson's textual type description.
func (v *Val) TypeDescription() string {
	if !v.valid() {
		return "none"
	}
	desc := bind.ValTypeDesc(v.ptr)
	runtime.KeepAlive(v.doc)
	return desc
}

// IsNull reports whether v is a JSON null value.
func (v *Val) IsNull() bool { return v.Type() == TypeNull }

// IsBool reports whether v is a JSON boolean value.
func (v *Val) IsBool() bool { return v.Type() == TypeBool }

// IsNum reports whether v is a JSON number value.
func (v *Val) IsNum() bool { return v.Type() == TypeNum }

// IsStr reports whether v is a JSON string value.
func (v *Val) IsStr() bool { return v.Type() == TypeStr }

// IsArr reports whether v is a JSON array value.
func (v *Val) IsArr() bool { return v.Type() == TypeArr }

// IsObj reports whether v is a JSON object value.
func (v *Val) IsObj() bool { return v.Type() == TypeObj }

// IsRaw reports whether v is a raw JSON value.
func (v *Val) IsRaw() bool { return v.Type() == TypeRaw }

// Bool returns v as a bool.
func (v *Val) Bool() bool {
	if !v.valid() {
		return false
	}
	out := bind.ValBool(v.ptr)
	runtime.KeepAlive(v.doc)
	return out
}

// Uint returns v as an unsigned integer.
func (v *Val) Uint() uint64 {
	if !v.valid() {
		return 0
	}
	out := bind.ValUint(v.ptr)
	runtime.KeepAlive(v.doc)
	return out
}

// Int returns v as a signed integer.
func (v *Val) Int() int64 {
	if !v.valid() {
		return 0
	}
	out := bind.ValInt(v.ptr)
	runtime.KeepAlive(v.doc)
	return out
}

// Float returns v as a floating-point number.
func (v *Val) Float() float64 {
	if !v.valid() {
		return 0
	}
	out := bind.ValFloat(v.ptr)
	runtime.KeepAlive(v.doc)
	return out
}

// Num returns v as a floating-point number.
func (v *Val) Num() float64 {
	if !v.valid() {
		return 0
	}
	out := bind.ValNum(v.ptr)
	runtime.KeepAlive(v.doc)
	return out
}

// String returns v as a string.
func (v *Val) String() string {
	if !v.valid() {
		return ""
	}
	out := bind.ValString(v.ptr)
	runtime.KeepAlive(v.doc)
	return out
}

// Raw returns v as raw JSON text.
func (v *Val) Raw() string {
	if !v.valid() {
		return ""
	}
	out := bind.ValRaw(v.ptr)
	runtime.KeepAlive(v.doc)
	return out
}

// Len returns the length of v when it is an array, object, string, or raw value.
func (v *Val) Len() uint64 {
	if !v.valid() {
		return 0
	}
	n := bind.ValLen(v.ptr)
	runtime.KeepAlive(v.doc)
	return n
}

// Index returns the array element at idx.
func (v *Val) Index(idx uint64) *Val {
	if !v.valid() {
		return nil
	}
	ptr := bind.ArrGet(v.ptr, idx)
	runtime.KeepAlive(v.doc)
	if ptr == 0 {
		return nil
	}
	return &Val{ptr: ptr, doc: v.doc}
}

// Get returns the object value for key.
func (v *Val) Get(key string) *Val {
	if !v.valid() {
		return nil
	}
	ptr := bind.ObjGet(v.ptr, key)
	runtime.KeepAlive(v.doc)
	if ptr == 0 {
		return nil
	}
	return &Val{ptr: ptr, doc: v.doc}
}

// Array returns the array elements.
func (v *Val) Array() []*Val {
	if !v.valid() {
		return nil
	}
	raw := bind.ArrValues(v.ptr)
	runtime.KeepAlive(v.doc)
	out := make([]*Val, 0, len(raw))
	for _, ptr := range raw {
		out = append(out, &Val{ptr: ptr, doc: v.doc})
	}
	return out
}

// Object returns the object members in document order.
func (v *Val) Object() []Member {
	if !v.valid() {
		return nil
	}
	raw := bind.ObjMembers(v.ptr)
	runtime.KeepAlive(v.doc)
	out := make([]Member, 0, len(raw))
	for _, member := range raw {
		out = append(out, Member{
			Key:   &Val{ptr: member.Key, doc: v.doc},
			Value: &Val{ptr: member.Val, doc: v.doc},
		})
	}
	return out
}

// EqualsString reports whether v equals s.
func (v *Val) EqualsString(s string) bool {
	if !v.valid() {
		return false
	}
	out := bind.ValEqualsString(v.ptr, s)
	runtime.KeepAlive(v.doc)
	return out
}

// Equal reports whether v and other are equal JSON values.
func (v *Val) Equal(other *Val) bool {
	if !v.valid() || other == nil || !other.valid() {
		return false
	}
	out := bind.ValEqual(v.ptr, other.ptr)
	runtime.KeepAlive(v.doc)
	runtime.KeepAlive(other.doc)
	return out
}

// Write serializes the value.
func (v *Val) Write(flags ...WriteFlag) ([]byte, error) {
	if !v.valid() {
		return nil, fmt.Errorf("yyjson value write failed: value is closed")
	}
	out, err := bind.WriteVal(v.ptr, writeFlags(flags))
	runtime.KeepAlive(v.doc)
	if err != nil {
		return nil, writeError(err, "write value", "")
	}
	return out, nil
}

// WriteString serializes the value as a string.
func (v *Val) WriteString(flags ...WriteFlag) (string, error) {
	out, err := v.Write(flags...)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Pointer resolves a JSON Pointer path from v.
func (v *Val) Pointer(path string) (*Val, error) {
	if !v.valid() {
		return nil, fmt.Errorf("yyjson pointer get failed at %q: value is closed", path)
	}
	ptr, err := bind.Pointer(v.ptr, path)
	runtime.KeepAlive(v.doc)
	if err != nil {
		return nil, pointerError(err, "get", path)
	}
	return &Val{ptr: ptr, doc: v.doc}, nil
}
