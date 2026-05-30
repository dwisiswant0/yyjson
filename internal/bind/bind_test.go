package bind

import (
	"math"
	"testing"

	"modernc.org/libc"
)

func TestReadWriteSmoke(t *testing.T) {
	res, err := Read([]byte(`{"ok":true,"items":[1]}`), 0)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	defer FreeDoc(res.Doc, res.Buf)

	root := DocRoot(res.Doc)
	if root == 0 {
		t.Fatal("DocRoot() = 0")
	}
	if got := ObjGet(root, "ok"); !ValBool(got) {
		t.Fatal(`ObjGet(root, "ok") != true`)
	}
	if got := ArrGet(ObjGet(root, "items"), 0); ValInt(got) != 1 {
		t.Fatalf("items[0] = %d", ValInt(got))
	}

	out, werr := WriteDoc(res.Doc, 0)
	if werr != nil {
		t.Fatalf("WriteDoc() error = %v", werr)
	}
	if string(out) != `{"ok":true,"items":[1]}` {
		t.Fatalf("WriteDoc() = %q", out)
	}
}

func TestAllocBytesAddsReadPadding(t *testing.T) {
	tls := libc.NewTLS()
	defer tls.Close()

	data := []byte(`{"s":"kept"}`)
	ptr, err := allocBytes(tls, data)
	if err != nil {
		t.Fatalf("allocBytes() error = %v", err)
	}
	defer libc.Xfree(tls, ptr)

	buf := cBytes(ptr, len(data)+readPaddingSize)
	if got := string(buf[:len(data)]); got != string(data) {
		t.Fatalf("copied data = %q", got)
	}
	for i, b := range buf[len(data):] {
		if b != 0 {
			t.Fatalf("padding byte %d = %d, want 0", i, b)
		}
	}
}

func TestNewIncrStateRejectsOversizedBuffer(t *testing.T) {
	state, buf, err := NewIncrState(math.MaxInt, 0)
	if err == nil {
		FreeIncrState(state, buf)
		t.Fatal("NewIncrState(math.MaxInt) error = nil")
	}
	if err.Code == 0 {
		t.Fatalf("NewIncrState(math.MaxInt) error missing code: %#v", err)
	}
}
