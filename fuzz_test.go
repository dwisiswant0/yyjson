package yyjson

import (
	stdjson "encoding/json"
	"testing"
)

const fuzzMaxInput = 1 << 20

func FuzzReadWriteRoundTrip(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`null`),
		[]byte(`true`),
		[]byte(`123`),
		[]byte(`{"name":"yyjson","items":[1,true,null,"x"]}`),
		[]byte(`[{"dup":1,"dup":2},{"s":"a\u0000b"}]`),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > fuzzMaxInput {
			t.Skip("input too large")
		}
		doc, err := Read(data)
		if err != nil {
			return
		}
		defer doc.Close()

		out, err := doc.Write()
		if err != nil {
			t.Fatalf("Write() after successful Read() error = %v", err)
		}
		if !stdjson.Valid(out) {
			t.Fatalf("Write() produced invalid JSON: %q", out)
		}
		roundTrip, err := Read(out)
		if err != nil {
			t.Fatalf("Read(Write(input)) error = %v; output %q", err, out)
		}
		roundTrip.Close()
	})
}

func FuzzPointer(f *testing.F) {
	f.Add([]byte(`{"items":[{"name":"first"},{"name":"second"}]}`), "/items/1/name")
	f.Add([]byte(`{"a/b":{"~key":1}}`), "/a~1b/~0key")
	f.Add([]byte(`[0,1,2]`), "/2")

	f.Fuzz(func(t *testing.T, data []byte, path string) {
		if len(data) > fuzzMaxInput || len(path) > fuzzMaxInput {
			t.Skip("input too large")
		}
		doc, err := Read(data)
		if err != nil {
			return
		}
		defer doc.Close()

		_, _ = doc.Pointer(path)
		if root := doc.Root(); root != nil {
			_, _ = root.Pointer(path)
		}
	})
}

func FuzzPatchAndMergePatch(f *testing.F) {
	f.Add([]byte(`{"name":"old"}`), []byte(`[{"op":"replace","path":"/name","value":"new"}]`))
	f.Add([]byte(`{"a":1,"b":2}`), []byte(`{"b":null,"c":3}`))
	f.Add([]byte(`[1,2,3]`), []byte(`[{"op":"remove","path":"/1"}]`))

	f.Fuzz(func(t *testing.T, origData, patchData []byte) {
		if len(origData) > fuzzMaxInput || len(patchData) > fuzzMaxInput {
			t.Skip("input too large")
		}
		orig, err := Read(origData)
		if err != nil {
			return
		}
		defer orig.Close()
		patch, err := Read(patchData)
		if err != nil {
			return
		}
		defer patch.Close()

		doc, err := NewMutDoc()
		if err != nil {
			t.Fatalf("NewMutDoc() error = %v", err)
		}
		defer doc.Close()

		if patched, err := doc.Patch(orig.Root(), patch.Root()); err == nil {
			if patched == nil {
				t.Fatal("Patch() returned nil value and nil error")
			}
			doc.SetRoot(patched)
			if _, err := doc.Write(); err != nil {
				t.Fatalf("Write() after Patch() error = %v", err)
			}
		}
		if merged, err := doc.MergePatch(orig.Root(), patch.Root()); err == nil {
			if merged == nil {
				t.Fatal("MergePatch() returned nil value and nil error")
			}
			doc.SetRoot(merged)
			if _, err := doc.Write(); err != nil {
				t.Fatalf("Write() after MergePatch() error = %v", err)
			}
		}
	})
}

func FuzzIncrementalRead(f *testing.F) {
	f.Add([]byte(`{"ok":true}`), 5)
	f.Add([]byte(`[1,2,3]`), 1)
	f.Add([]byte(`null`), 4)

	f.Fuzz(func(t *testing.T, data []byte, split int) {
		if len(data) == 0 || len(data) > fuzzMaxInput {
			t.Skip("input size outside incremental fuzz bounds")
		}
		reader, err := NewIncrReader(len(data))
		if err != nil {
			t.Fatalf("NewIncrReader(%d) error = %v", len(data), err)
		}
		defer reader.Close()

		split %= len(data) + 1
		if split < 0 {
			split = -split
		}
		if err := reader.Append(data[:split]); err != nil {
			return
		}
		_, _ = reader.Read()
		if err := reader.Append(data[split:]); err != nil {
			return
		}
		doc, err := reader.Read()
		if err != nil {
			return
		}
		defer doc.Close()
		if _, err := doc.Write(); err != nil {
			t.Fatalf("Write() after successful incremental Read() error = %v", err)
		}
	})
}
