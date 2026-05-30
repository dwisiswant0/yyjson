// Package yyjson is yyjson for Go, transpiled to pure Go (CGo-free). Covers
// parse, inspect, mutate, write, and JSON Pointer/Patch.
package yyjson

import (
	"fmt"
	"io"
	"runtime"

	"go.dw1.io/yyjson/internal/bind"
)

// Doc owns an immutable yyjson document.
//
// Warning: read operations may be used concurrently, but [Doc.Close] must not
// race with any other operation.
type Doc struct {
	ptr    bind.Doc
	buf    uintptr
	closed bool
}

// IncrReader incrementally accepts JSON input before producing a [Doc].
//
// Warning: not safe for concurrent use.
type IncrReader struct {
	state  bind.IncrState
	buf    uintptr
	max    int
	n      int
	closed bool
}

// Read parses a JSON document from data.
func Read(data []byte, flags ...ReadFlag) (*Doc, error) {
	res, err := bind.Read(data, readFlags(flags))
	if err != nil {
		return nil, readError(err, "read", "")
	}
	return newDoc(res.Doc, res.Buf), nil
}

// ReadString parses a JSON document from s.
func ReadString(s string, flags ...ReadFlag) (*Doc, error) {
	return Read([]byte(s), flags...)
}

// ReadAll reads all data from r and parses it as a JSON document.
func ReadAll(r io.Reader, flags ...ReadFlag) (*Doc, error) {
	if r == nil {
		return nil, fmt.Errorf("yyjson read from reader failed: reader is nil")
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("yyjson read from reader failed: %w", err)
	}
	return Read(data, flags...)
}

// ReadFile reads and parses a JSON document from path.
func ReadFile(path string, flags ...ReadFlag) (*Doc, error) {
	doc, err := bind.ReadFile(path, readFlags(flags))
	if err != nil {
		return nil, readError(err, "read file", path)
	}
	return newDoc(doc, 0), nil
}

// NewIncrReader creates an incremental reader with space for maxLen bytes.
func NewIncrReader(maxLen int, flags ...ReadFlag) (*IncrReader, error) {
	state, buf, err := bind.NewIncrState(maxLen, readFlags(flags))
	if err != nil {
		return nil, readError(err, "incremental reader init", "")
	}
	r := &IncrReader{state: state, buf: buf, max: maxLen}
	runtime.SetFinalizer(r, (*IncrReader).Close)
	return r, nil
}

// Append appends p to the incremental input buffer.
func (r *IncrReader) Append(p []byte) error {
	if r == nil || r.closed || r.state == 0 {
		return fmt.Errorf("yyjson incremental append failed: reader is closed")
	}
	if len(p) > r.max-r.n {
		return fmt.Errorf("yyjson incremental append failed: %d bytes exceed remaining buffer capacity %d", len(p), r.max-r.n)
	}
	bind.IncrAppend(r.buf, r.n, p)
	runtime.KeepAlive(r)
	r.n += len(p)
	return nil
}

// Read parses the currently appended incremental input.
func (r *IncrReader) Read() (*Doc, error) {
	if r == nil || r.closed || r.state == 0 {
		return nil, fmt.Errorf("yyjson incremental read failed: reader is closed")
	}
	doc, err := bind.IncrRead(r.state, r.n)
	runtime.KeepAlive(r)
	if err != nil {
		if err.Code != ReadErrorMore {
			r.Close()
		}
		return nil, readError(err, "incremental read", "")
	}
	buf := r.buf
	runtime.SetFinalizer(r, nil)
	bind.FreeIncrState(r.state, 0)
	r.state = 0
	r.buf = 0
	r.closed = true
	return newDoc(doc, buf), nil
}

// Len returns the number of bytes appended to the incremental reader.
func (r *IncrReader) Len() int {
	if r == nil {
		return 0
	}
	return r.n
}

// Close releases the incremental reader and its input buffer.
func (r *IncrReader) Close() {
	if r == nil {
		return
	}
	runtime.SetFinalizer(r, nil)
	bind.FreeIncrState(r.state, r.buf)
	r.state = 0
	r.buf = 0
	r.closed = true
}

func newDoc(ptr bind.Doc, buf uintptr) *Doc {
	d := &Doc{ptr: ptr, buf: buf}
	runtime.SetFinalizer(d, (*Doc).Close)
	return d
}

// Close releases the document and any retained parse buffer.
func (d *Doc) Close() {
	if d == nil || d.closed {
		return
	}
	runtime.SetFinalizer(d, nil)
	bind.FreeDoc(d.ptr, d.buf)
	d.ptr = 0
	d.buf = 0
	d.closed = true
}

// Root returns the document root value.
func (d *Doc) Root() *Val {
	if d == nil || d.closed || d.ptr == 0 {
		return nil
	}
	ptr := bind.DocRoot(d.ptr)
	runtime.KeepAlive(d)
	if ptr == 0 {
		return nil
	}
	return &Val{ptr: ptr, doc: d}
}

// ReadSize returns the number of bytes read for the document.
func (d *Doc) ReadSize() uint64 {
	if d == nil || d.closed {
		return 0
	}
	n := bind.DocReadSize(d.ptr)
	runtime.KeepAlive(d)
	return n
}

// ValueCount returns the number of values in the document.
func (d *Doc) ValueCount() uint64 {
	if d == nil || d.closed {
		return 0
	}
	n := bind.DocValCount(d.ptr)
	runtime.KeepAlive(d)
	return n
}

// Write serializes the document.
func (d *Doc) Write(flags ...WriteFlag) ([]byte, error) {
	if d == nil || d.closed || d.ptr == 0 {
		return nil, fmt.Errorf("yyjson write failed: document is closed")
	}
	out, err := bind.WriteDoc(d.ptr, writeFlags(flags))
	runtime.KeepAlive(d)
	if err != nil {
		return nil, writeError(err, "write", "")
	}
	return out, nil
}

// WriteString serializes the document as a string.
func (d *Doc) WriteString(flags ...WriteFlag) (string, error) {
	out, err := d.Write(flags...)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// WriteTo serializes the document to w.
func (d *Doc) WriteTo(w io.Writer, flags ...WriteFlag) (int64, error) {
	if w == nil {
		return 0, fmt.Errorf("yyjson write to writer failed: writer is nil")
	}
	out, err := d.Write(flags...)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(out)
	if err != nil {
		return int64(n), fmt.Errorf("yyjson write to writer failed: %w", err)
	}
	if n != len(out) {
		return int64(n), io.ErrShortWrite
	}
	return int64(n), nil
}

// WriteFile serializes the document to path.
func (d *Doc) WriteFile(path string, flags ...WriteFlag) error {
	if d == nil || d.closed || d.ptr == 0 {
		return fmt.Errorf("yyjson write file failed for %s: document is closed", path)
	}
	if err := bind.WriteDocFile(path, d.ptr, writeFlags(flags)); err != nil {
		runtime.KeepAlive(d)
		return writeError(err, "write file", path)
	}
	runtime.KeepAlive(d)
	return nil
}

// Pointer resolves a JSON Pointer path from the document root.
func (d *Doc) Pointer(path string) (*Val, error) {
	if d == nil || d.closed || d.ptr == 0 {
		return nil, fmt.Errorf("yyjson pointer get failed at %q: document is closed", path)
	}
	ptr, err := bind.DocPointer(d.ptr, path)
	runtime.KeepAlive(d)
	if err != nil {
		return nil, pointerError(err, "get", path)
	}
	return &Val{ptr: ptr, doc: d}, nil
}
