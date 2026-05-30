package yyjson

import (
	"fmt"
	"io"
	"runtime"

	"go.dw1.io/yyjson/internal/bind"
)

// MutDoc owns a mutable yyjson document.
//
// Warning: not safe for concurrent use.
type MutDoc struct {
	ptr    bind.MutDoc
	closed bool
}

// MutVal is a mutable JSON value owned by a [MutDoc].
//
// Warning: not safe for concurrent use.
type MutVal struct {
	ptr bind.MutVal
	doc *MutDoc
}

// MutMember is a mutable object member.
type MutMember struct {
	Key   *MutVal
	Value *MutVal
}

// NewMutDoc creates an empty mutable document.
func NewMutDoc() (*MutDoc, error) {
	ptr := bind.NewMutDoc()
	if ptr == 0 {
		return nil, fmt.Errorf("yyjson mutable document allocation failed")
	}
	return newMutDoc(ptr), nil
}

// Mutate creates a mutable copy of doc.
func Mutate(doc *Doc) (*MutDoc, error) {
	if doc == nil || doc.closed || doc.ptr == 0 {
		return nil, fmt.Errorf("yyjson mutable copy failed: document is closed")
	}
	ptr := bind.MutDocFromDoc(doc.ptr)
	runtime.KeepAlive(doc)
	if ptr == 0 {
		return nil, fmt.Errorf("yyjson mutable copy failed: allocation failed")
	}
	return newMutDoc(ptr), nil
}

func (d *MutDoc) valid() bool {
	return d != nil && !d.closed && d.ptr != 0
}

// Close releases the mutable document.
func (d *MutDoc) Close() {
	if d == nil || d.closed {
		return
	}
	runtime.SetFinalizer(d, nil)
	bind.FreeMutDoc(d.ptr)
	d.ptr = 0
	d.closed = true
}

// Root returns the mutable document root value.
func (d *MutDoc) Root() *MutVal {
	if !d.valid() {
		return nil
	}
	ptr := bind.MutDocRoot(d.ptr)
	runtime.KeepAlive(d)
	if ptr == 0 {
		return nil
	}
	return &MutVal{ptr: ptr, doc: d}
}

// SetRoot sets the mutable document root.
func (d *MutDoc) SetRoot(root *MutVal) {
	if !d.valid() {
		return
	}
	var ptr bind.MutVal
	if root != nil {
		if !d.owns(root) {
			return
		}
		ptr = root.ptr
	}
	bind.MutDocSetRoot(d.ptr, ptr)
	runtime.KeepAlive(d)
	if root != nil {
		runtime.KeepAlive(root.doc)
	}
}

// Null creates a mutable null value.
func (d *MutDoc) Null() *MutVal {
	if !d.valid() {
		return nil
	}
	ptr := bind.MutNull(d.ptr)
	runtime.KeepAlive(d)
	return d.wrap(ptr)
}

// True creates a mutable true value.
func (d *MutDoc) True() *MutVal {
	if !d.valid() {
		return nil
	}
	ptr := bind.MutTrue(d.ptr)
	runtime.KeepAlive(d)
	return d.wrap(ptr)
}

// False creates a mutable false value.
func (d *MutDoc) False() *MutVal {
	if !d.valid() {
		return nil
	}
	ptr := bind.MutFalse(d.ptr)
	runtime.KeepAlive(d)
	return d.wrap(ptr)
}

// Bool creates a mutable boolean value.
func (d *MutDoc) Bool(v bool) *MutVal {
	if !d.valid() {
		return nil
	}
	ptr := bind.MutBool(d.ptr, v)
	runtime.KeepAlive(d)
	return d.wrap(ptr)
}

// Uint creates a mutable unsigned integer value.
func (d *MutDoc) Uint(v uint64) *MutVal {
	if !d.valid() {
		return nil
	}
	ptr := bind.MutUint(d.ptr, v)
	runtime.KeepAlive(d)
	return d.wrap(ptr)
}

// Int creates a mutable signed integer value.
func (d *MutDoc) Int(v int64) *MutVal {
	if !d.valid() {
		return nil
	}
	ptr := bind.MutInt(d.ptr, v)
	runtime.KeepAlive(d)
	return d.wrap(ptr)
}

// Float creates a mutable floating-point value.
func (d *MutDoc) Float(v float64) *MutVal {
	if !d.valid() {
		return nil
	}
	ptr := bind.MutReal(d.ptr, v)
	runtime.KeepAlive(d)
	return d.wrap(ptr)
}

// String creates a mutable string value.
func (d *MutDoc) String(v string) *MutVal {
	if !d.valid() {
		return nil
	}
	ptr := bind.MutString(d.ptr, v)
	runtime.KeepAlive(d)
	return d.wrap(ptr)
}

// Raw creates a mutable raw JSON value.
func (d *MutDoc) Raw(v string) *MutVal {
	if !d.valid() {
		return nil
	}
	ptr := bind.MutRaw(d.ptr, v)
	runtime.KeepAlive(d)
	return d.wrap(ptr)
}

// Array creates a mutable array value.
func (d *MutDoc) Array() *MutVal {
	if !d.valid() {
		return nil
	}
	ptr := bind.MutArray(d.ptr)
	runtime.KeepAlive(d)
	return d.wrap(ptr)
}

// Object creates a mutable object value.
func (d *MutDoc) Object() *MutVal {
	if !d.valid() {
		return nil
	}
	ptr := bind.MutObject(d.ptr)
	runtime.KeepAlive(d)
	return d.wrap(ptr)
}

func (d *MutDoc) wrap(ptr bind.MutVal) *MutVal {
	if !d.valid() || ptr == 0 {
		return nil
	}
	return &MutVal{ptr: ptr, doc: d}
}

func newMutDoc(ptr bind.MutDoc) *MutDoc {
	d := &MutDoc{ptr: ptr}
	runtime.SetFinalizer(d, (*MutDoc).Close)
	return d
}

func (d *MutDoc) owns(val *MutVal) bool {
	return d.valid() && val != nil && val.valid() && val.doc == d
}

// Write serializes the mutable document.
func (d *MutDoc) Write(flags ...WriteFlag) ([]byte, error) {
	if !d.valid() {
		return nil, fmt.Errorf("yyjson mutable write failed: document is closed")
	}
	out, err := bind.WriteMutDoc(d.ptr, writeFlags(flags))
	runtime.KeepAlive(d)
	if err != nil {
		return nil, writeError(err, "write mutable document", "")
	}
	return out, nil
}

// WriteString serializes the mutable document as a string.
func (d *MutDoc) WriteString(flags ...WriteFlag) (string, error) {
	out, err := d.Write(flags...)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// WriteTo serializes the mutable document to w.
func (d *MutDoc) WriteTo(w io.Writer, flags ...WriteFlag) (int64, error) {
	if w == nil {
		return 0, fmt.Errorf("yyjson write mutable document to writer failed: writer is nil")
	}
	out, err := d.Write(flags...)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(out)
	if err != nil {
		return int64(n), fmt.Errorf("yyjson write mutable document to writer failed: %w", err)
	}
	if n != len(out) {
		return int64(n), io.ErrShortWrite
	}
	return int64(n), nil
}

// WriteFile serializes the mutable document to path.
func (d *MutDoc) WriteFile(path string, flags ...WriteFlag) error {
	if !d.valid() {
		return fmt.Errorf("yyjson write mutable file failed for %s: document is closed", path)
	}
	if err := bind.WriteMutDocFile(path, d.ptr, writeFlags(flags)); err != nil {
		runtime.KeepAlive(d)
		return writeError(err, "write mutable file", path)
	}
	runtime.KeepAlive(d)
	return nil
}

// Immutable creates an immutable copy of the mutable document.
func (d *MutDoc) Immutable() (*Doc, error) {
	if !d.valid() {
		return nil, fmt.Errorf("yyjson immutable copy failed: mutable document is closed")
	}
	ptr := bind.MutDocToDoc(d.ptr)
	runtime.KeepAlive(d)
	if ptr == 0 {
		return nil, fmt.Errorf("yyjson immutable copy failed: allocation failed")
	}
	return newDoc(ptr, 0), nil
}

// Pointer resolves a JSON Pointer path from the mutable document root.
func (d *MutDoc) Pointer(path string) (*MutVal, error) {
	if !d.valid() {
		return nil, fmt.Errorf("yyjson pointer get failed at %q: mutable document is closed", path)
	}
	ptr, err := bind.MutDocPointer(d.ptr, path)
	runtime.KeepAlive(d)
	if err != nil {
		return nil, pointerError(err, "get", path)
	}
	return &MutVal{ptr: ptr, doc: d}, nil
}

// PointerSet sets the value at a JSON Pointer path.
func (d *MutDoc) PointerSet(path string, val *MutVal) error {
	if !d.valid() {
		return fmt.Errorf("yyjson pointer set failed at %q: mutable document is closed", path)
	}
	if !d.owns(val) {
		return fmt.Errorf("yyjson pointer set failed at %q: value belongs to another mutable document or is closed", path)
	}
	if err := bind.MutDocPointerSet(d.ptr, path, val.ptr, true); err != nil {
		runtime.KeepAlive(d)
		runtime.KeepAlive(val.doc)
		return pointerError(err, "set", path)
	}
	runtime.KeepAlive(d)
	runtime.KeepAlive(val.doc)
	return nil
}

// PointerAdd adds val at a JSON Pointer path.
func (d *MutDoc) PointerAdd(path string, val *MutVal) error {
	if !d.valid() {
		return fmt.Errorf("yyjson pointer add failed at %q: mutable document is closed", path)
	}
	if !d.owns(val) {
		return fmt.Errorf("yyjson pointer add failed at %q: value belongs to another mutable document or is closed", path)
	}
	if err := bind.MutDocPointerAdd(d.ptr, path, val.ptr, true); err != nil {
		runtime.KeepAlive(d)
		runtime.KeepAlive(val.doc)
		return pointerError(err, "add", path)
	}
	runtime.KeepAlive(d)
	runtime.KeepAlive(val.doc)
	return nil
}

// PointerReplace replaces the value at a JSON Pointer path and returns the old value.
func (d *MutDoc) PointerReplace(path string, val *MutVal) (*MutVal, error) {
	if !d.valid() {
		return nil, fmt.Errorf("yyjson pointer replace failed at %q: mutable document is closed", path)
	}
	if !d.owns(val) {
		return nil, fmt.Errorf("yyjson pointer replace failed at %q: value belongs to another mutable document or is closed", path)
	}
	old, err := bind.MutDocPointerReplace(d.ptr, path, val.ptr)
	runtime.KeepAlive(d)
	runtime.KeepAlive(val.doc)
	if err != nil {
		return nil, pointerError(err, "replace", path)
	}
	return &MutVal{ptr: old, doc: d}, nil
}

// PointerRemove removes and returns the value at a JSON Pointer path.
func (d *MutDoc) PointerRemove(path string) (*MutVal, error) {
	if !d.valid() {
		return nil, fmt.Errorf("yyjson pointer remove failed at %q: mutable document is closed", path)
	}
	old, err := bind.MutDocPointerRemove(d.ptr, path)
	runtime.KeepAlive(d)
	if err != nil {
		return nil, pointerError(err, "remove", path)
	}
	return &MutVal{ptr: old, doc: d}, nil
}

// Patch applies a JSON Patch document to orig and stores the result in d.
func (d *MutDoc) Patch(orig, patch *Val) (*MutVal, error) {
	if !d.valid() {
		return nil, fmt.Errorf("yyjson patch failed: mutable document is closed")
	}
	if orig == nil || !orig.valid() || patch == nil || !patch.valid() {
		return nil, fmt.Errorf("yyjson patch failed: orig and patch must be open values")
	}
	out, err := bind.Patch(d.ptr, orig.ptr, patch.ptr)
	runtime.KeepAlive(d)
	runtime.KeepAlive(orig.doc)
	runtime.KeepAlive(patch.doc)
	if err != nil {
		return nil, patchError(err)
	}
	return &MutVal{ptr: out, doc: d}, nil
}

// MergePatch applies a JSON Merge Patch document to orig and stores the result in d.
func (d *MutDoc) MergePatch(orig, patch *Val) (*MutVal, error) {
	if !d.valid() {
		return nil, fmt.Errorf("yyjson merge patch failed: mutable document is closed")
	}
	if orig == nil || !orig.valid() || patch == nil || !patch.valid() {
		return nil, fmt.Errorf("yyjson merge patch failed: orig and patch must be open values")
	}
	out := bind.MergePatch(d.ptr, orig.ptr, patch.ptr)
	runtime.KeepAlive(d)
	runtime.KeepAlive(orig.doc)
	runtime.KeepAlive(patch.doc)
	if out == 0 {
		return nil, fmt.Errorf("yyjson merge patch failed: allocation failed or invalid input")
	}
	return &MutVal{ptr: out, doc: d}, nil
}

func (v *MutVal) valid() bool {
	return v != nil && v.ptr != 0 && v.doc != nil && v.doc.valid()
}

// Type returns the mutable value's primary JSON type.
func (v *MutVal) Type() Type {
	if !v.valid() {
		return TypeNone
	}
	typ := Type(bind.MutValType(v.ptr))
	runtime.KeepAlive(v.doc)
	return typ
}

// Subtype returns the mutable value's yyjson subtype.
func (v *MutVal) Subtype() Subtype {
	if !v.valid() {
		return SubtypeNone
	}
	subtype := Subtype(bind.MutValSubtype(v.ptr))
	runtime.KeepAlive(v.doc)
	return subtype
}

// TypeDescription returns yyjson's textual type description.
func (v *MutVal) TypeDescription() string {
	if !v.valid() {
		return "none"
	}
	desc := bind.MutValTypeDesc(v.ptr)
	runtime.KeepAlive(v.doc)
	return desc
}

// IsNull reports whether v is a JSON null value.
func (v *MutVal) IsNull() bool { return v.Type() == TypeNull }

// IsBool reports whether v is a JSON boolean value.
func (v *MutVal) IsBool() bool { return v.Type() == TypeBool }

// IsNum reports whether v is a JSON number value.
func (v *MutVal) IsNum() bool { return v.Type() == TypeNum }

// IsStr reports whether v is a JSON string value.
func (v *MutVal) IsStr() bool { return v.Type() == TypeStr }

// IsArr reports whether v is a JSON array value.
func (v *MutVal) IsArr() bool { return v.Type() == TypeArr }

// IsObj reports whether v is a JSON object value.
func (v *MutVal) IsObj() bool { return v.Type() == TypeObj }

// IsRaw reports whether v is a raw JSON value.
func (v *MutVal) IsRaw() bool { return v.Type() == TypeRaw }

// Bool returns v as a bool.
func (v *MutVal) Bool() bool {
	if !v.valid() {
		return false
	}
	out := bind.MutValBool(v.ptr)
	runtime.KeepAlive(v.doc)
	return out
}

// Uint returns v as an unsigned integer.
func (v *MutVal) Uint() uint64 {
	if !v.valid() {
		return 0
	}
	out := bind.MutValUint(v.ptr)
	runtime.KeepAlive(v.doc)
	return out
}

// Int returns v as a signed integer.
func (v *MutVal) Int() int64 {
	if !v.valid() {
		return 0
	}
	out := bind.MutValInt(v.ptr)
	runtime.KeepAlive(v.doc)
	return out
}

// Float returns v as a floating-point number.
func (v *MutVal) Float() float64 {
	if !v.valid() {
		return 0
	}
	out := bind.MutValFloat(v.ptr)
	runtime.KeepAlive(v.doc)
	return out
}

// Num returns v as a floating-point number.
func (v *MutVal) Num() float64 {
	if !v.valid() {
		return 0
	}
	out := bind.MutValNum(v.ptr)
	runtime.KeepAlive(v.doc)
	return out
}

// String returns v as a string.
func (v *MutVal) String() string {
	if !v.valid() {
		return ""
	}
	out := bind.MutValString(v.ptr)
	runtime.KeepAlive(v.doc)
	return out
}

// Raw returns v as raw JSON text.
func (v *MutVal) Raw() string {
	if !v.valid() {
		return ""
	}
	out := bind.MutValRaw(v.ptr)
	runtime.KeepAlive(v.doc)
	return out
}

// EqualsString reports whether v equals s.
func (v *MutVal) EqualsString(s string) bool {
	if !v.valid() {
		return false
	}
	out := bind.MutValEqualsString(v.ptr, s)
	runtime.KeepAlive(v.doc)
	return out
}

// Equal reports whether v and other are equal JSON values.
func (v *MutVal) Equal(other *MutVal) bool {
	if !v.valid() || other == nil || !other.valid() {
		return false
	}
	out := bind.MutValEqual(v.ptr, other.ptr)
	runtime.KeepAlive(v.doc)
	runtime.KeepAlive(other.doc)
	return out
}

// Len returns the length of v when it is an array, object, string, or raw value.
func (v *MutVal) Len() uint64 {
	if !v.valid() {
		return 0
	}
	n := bind.MutValLen(v.ptr)
	runtime.KeepAlive(v.doc)
	return n
}

// Index returns the array element at idx.
func (v *MutVal) Index(idx uint64) *MutVal {
	if !v.valid() {
		return nil
	}
	ptr := bind.MutArrGet(v.ptr, idx)
	runtime.KeepAlive(v.doc)
	if ptr == 0 {
		return nil
	}
	return &MutVal{ptr: ptr, doc: v.doc}
}

// Append appends val to an array.
func (v *MutVal) Append(val *MutVal) bool {
	if !v.valid() || !v.doc.owns(val) {
		return false
	}
	ok := bind.MutArrAppend(v.ptr, val.ptr)
	runtime.KeepAlive(v.doc)
	runtime.KeepAlive(val.doc)
	return ok
}

// Insert inserts val into an array at idx.
func (v *MutVal) Insert(idx uint64, val *MutVal) bool {
	if !v.valid() || !v.doc.owns(val) {
		return false
	}
	ok := bind.MutArrInsert(v.ptr, idx, val.ptr)
	runtime.KeepAlive(v.doc)
	runtime.KeepAlive(val.doc)
	return ok
}

// Replace replaces an array element and returns the old value.
func (v *MutVal) Replace(idx uint64, val *MutVal) *MutVal {
	if !v.valid() || !v.doc.owns(val) {
		return nil
	}
	old := bind.MutArrReplace(v.ptr, idx, val.ptr)
	runtime.KeepAlive(v.doc)
	runtime.KeepAlive(val.doc)
	if old == 0 {
		return nil
	}
	return &MutVal{ptr: old, doc: v.doc}
}

// Remove removes and returns an array element.
func (v *MutVal) Remove(idx uint64) *MutVal {
	if !v.valid() {
		return nil
	}
	old := bind.MutArrRemove(v.ptr, idx)
	runtime.KeepAlive(v.doc)
	if old == 0 {
		return nil
	}
	return &MutVal{ptr: old, doc: v.doc}
}

// ClearArray removes all array elements.
func (v *MutVal) ClearArray() bool {
	if !v.valid() {
		return false
	}
	ok := bind.MutArrClear(v.ptr)
	runtime.KeepAlive(v.doc)
	return ok
}

// Rotate rotates an array around idx.
func (v *MutVal) Rotate(idx uint64) bool {
	if !v.valid() {
		return false
	}
	ok := bind.MutArrRotate(v.ptr, idx)
	runtime.KeepAlive(v.doc)
	return ok
}

// Array returns the array elements.
func (v *MutVal) Array() []*MutVal {
	if !v.valid() {
		return nil
	}
	raw := bind.MutArrValues(v.ptr)
	runtime.KeepAlive(v.doc)
	out := make([]*MutVal, 0, len(raw))
	for _, ptr := range raw {
		out = append(out, &MutVal{ptr: ptr, doc: v.doc})
	}
	return out
}

// Get returns the object value for key.
func (v *MutVal) Get(key string) *MutVal {
	if !v.valid() {
		return nil
	}
	ptr := bind.MutObjGet(v.ptr, key)
	runtime.KeepAlive(v.doc)
	if ptr == 0 {
		return nil
	}
	return &MutVal{ptr: ptr, doc: v.doc}
}

// Add adds key and val to an object.
func (v *MutVal) Add(key *MutVal, val *MutVal) bool {
	if !v.valid() || !v.doc.owns(key) || !v.doc.owns(val) {
		return false
	}
	ok := bind.MutObjAdd(v.ptr, key.ptr, val.ptr)
	runtime.KeepAlive(v.doc)
	runtime.KeepAlive(key.doc)
	runtime.KeepAlive(val.doc)
	return ok
}

// Put adds or replaces key and val in an object.
func (v *MutVal) Put(key *MutVal, val *MutVal) bool {
	if !v.valid() || !v.doc.owns(key) || !v.doc.owns(val) {
		return false
	}
	ok := bind.MutObjPut(v.ptr, key.ptr, val.ptr)
	runtime.KeepAlive(v.doc)
	runtime.KeepAlive(key.doc)
	runtime.KeepAlive(val.doc)
	return ok
}

// AddKey adds val to an object using key.
func (v *MutVal) AddKey(key string, val *MutVal) bool {
	if !v.valid() || val == nil {
		return false
	}
	return v.Add(v.doc.String(key), val)
}

// PutKey adds or replaces val in an object using key.
func (v *MutVal) PutKey(key string, val *MutVal) bool {
	if !v.valid() || val == nil {
		return false
	}
	return v.Put(v.doc.String(key), val)
}

// RemoveKey removes and returns the object value for key.
func (v *MutVal) RemoveKey(key string) *MutVal {
	if !v.valid() {
		return nil
	}
	old := bind.MutObjRemoveKey(v.ptr, key)
	runtime.KeepAlive(v.doc)
	if old == 0 {
		return nil
	}
	return &MutVal{ptr: old, doc: v.doc}
}

// ClearObject removes all object members.
func (v *MutVal) ClearObject() bool {
	if !v.valid() {
		return false
	}
	ok := bind.MutObjClear(v.ptr)
	runtime.KeepAlive(v.doc)
	return ok
}

// RenameKey renames an object key.
func (v *MutVal) RenameKey(oldKey, newKey string) bool {
	if !v.valid() {
		return false
	}
	ok := bind.MutObjRenameKey(v.doc.ptr, v.ptr, oldKey, newKey)
	runtime.KeepAlive(v.doc)
	return ok
}

// Object returns the object members in document order.
func (v *MutVal) Object() []MutMember {
	if !v.valid() {
		return nil
	}
	raw := bind.MutObjMembers(v.ptr)
	runtime.KeepAlive(v.doc)
	out := make([]MutMember, 0, len(raw))
	for _, member := range raw {
		out = append(out, MutMember{
			Key:   &MutVal{ptr: member.Key, doc: v.doc},
			Value: &MutVal{ptr: member.Val, doc: v.doc},
		})
	}
	return out
}

// Write serializes the mutable value.
func (v *MutVal) Write(flags ...WriteFlag) ([]byte, error) {
	if !v.valid() {
		return nil, fmt.Errorf("yyjson mutable value write failed: value is closed")
	}
	out, err := bind.WriteMutVal(v.ptr, writeFlags(flags))
	runtime.KeepAlive(v.doc)
	if err != nil {
		return nil, writeError(err, "write mutable value", "")
	}
	return out, nil
}

// WriteString serializes the mutable value as a string.
func (v *MutVal) WriteString(flags ...WriteFlag) (string, error) {
	out, err := v.Write(flags...)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Immutable creates an immutable document from the mutable value.
func (v *MutVal) Immutable() (*Doc, error) {
	if !v.valid() {
		return nil, fmt.Errorf("yyjson immutable copy failed: mutable value is closed")
	}
	ptr := bind.MutValToDoc(v.ptr)
	runtime.KeepAlive(v.doc)
	if ptr == 0 {
		return nil, fmt.Errorf("yyjson immutable copy failed: allocation failed")
	}
	return newDoc(ptr, 0), nil
}
