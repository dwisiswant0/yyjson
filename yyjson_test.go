package yyjson

import (
	"bytes"
	stdjson "encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type customJSON struct{}

func (customJSON) MarshalJSON() ([]byte, error) {
	return []byte(`{"custom":true}`), nil
}

func TestReadInspectWrite(t *testing.T) {
	const input = `{"name":"yyjson","nums":[1,-2,3.5],"ok":true}`
	doc, err := ReadString(input)
	if err != nil {
		t.Fatalf("ReadString() error = %v", err)
	}
	defer doc.Close()
	if got := doc.ReadSize(); got != uint64(len(input)) {
		t.Fatalf("read size = %d, want %d", got, len(input))
	}
	if got := doc.ValueCount(); got == 0 {
		t.Fatal("value count = 0")
	}

	root := doc.Root()
	if root.Type() != TypeObj {
		t.Fatalf("root type = %v", root.Type())
	}
	if got := root.Get("name").String(); got != "yyjson" {
		t.Fatalf("name = %q", got)
	}
	nums := root.Get("nums")
	if got := nums.Len(); got != 3 {
		t.Fatalf("nums len = %d", got)
	}
	if got := nums.Index(0).Uint(); got != 1 {
		t.Fatalf("nums[0] = %d", got)
	}
	if got := nums.Index(1).Int(); got != -2 {
		t.Fatalf("nums[1] = %d", got)
	}
	if got := nums.Index(2).Float(); got != 3.5 {
		t.Fatalf("nums[2] = %v", got)
	}
	if !root.Get("ok").Bool() {
		t.Fatalf("ok = false")
	}
	out, err := doc.WriteString()
	if err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if out != `{"name":"yyjson","nums":[1,-2,3.5],"ok":true}` {
		t.Fatalf("write = %q", out)
	}
}

func TestImmutableValueAccessorsAndPointers(t *testing.T) {
	doc, err := ReadString(`{"none":null,"t":true,"f":false,"u":12,"i":-5,"r":2.5,"s":"text","arr":[null,true,3],"obj":{"a":1}}`)
	if err != nil {
		t.Fatalf("ReadString() error = %v", err)
	}
	defer doc.Close()

	root := doc.Root()
	if !root.IsObj() || root.Subtype() != SubtypeNone || root.TypeDescription() != "object" {
		t.Fatalf("root metadata: type=%v subtype=%v desc=%q", root.Type(), root.Subtype(), root.TypeDescription())
	}
	if !root.Get("none").IsNull() {
		t.Fatal("none is not null")
	}
	if tv := root.Get("t"); !tv.IsBool() || !tv.Bool() || tv.Subtype() != SubtypeTrue {
		t.Fatalf("true value: bool=%v subtype=%v", tv.Bool(), tv.Subtype())
	}
	if fv := root.Get("f"); !fv.IsBool() || fv.Bool() || fv.Subtype() != SubtypeFalse {
		t.Fatalf("false value: bool=%v subtype=%v", fv.Bool(), fv.Subtype())
	}
	if uv := root.Get("u"); !uv.IsNum() || uv.Uint() != 12 || uv.Subtype() != SubtypeUint {
		t.Fatalf("uint value: type=%v uint=%d subtype=%v", uv.Type(), uv.Uint(), uv.Subtype())
	}
	if iv := root.Get("i"); !iv.IsNum() || iv.Int() != -5 || iv.Subtype() != SubtypeSint {
		t.Fatalf("int value: type=%v int=%d subtype=%v", iv.Type(), iv.Int(), iv.Subtype())
	}
	if rv := root.Get("r"); !rv.IsNum() || rv.Float() != 2.5 || rv.Num() != 2.5 || rv.Subtype() != SubtypeReal {
		t.Fatalf("real value: type=%v float=%v num=%v subtype=%v", rv.Type(), rv.Float(), rv.Num(), rv.Subtype())
	}
	if sv := root.Get("s"); !sv.IsStr() || sv.String() != "text" || !sv.EqualsString("text") || sv.Len() != 4 {
		t.Fatalf("string value: type=%v string=%q len=%d", sv.Type(), sv.String(), sv.Len())
	}
	arr := root.Get("arr")
	if !arr.IsArr() || arr.Len() != 3 {
		t.Fatalf("array metadata: type=%v len=%d", arr.Type(), arr.Len())
	}
	values := arr.Array()
	if len(values) != 3 || !values[0].IsNull() || !values[1].Bool() || values[2].Int() != 3 {
		t.Fatalf("array values = %#v", values)
	}
	obj := root.Get("obj")
	if !obj.IsObj() || obj.Len() != 1 {
		t.Fatalf("object metadata: type=%v len=%d", obj.Type(), obj.Len())
	}
	pointed, err := root.Pointer("/obj/a")
	if err != nil {
		t.Fatalf("Val.Pointer() error = %v", err)
	}
	if pointed.Int() != 1 {
		t.Fatalf("pointed value = %d", pointed.Int())
	}
	if _, err := root.Pointer("/missing"); err == nil {
		t.Fatal("Val.Pointer(missing) error = nil")
	}
	if arr.Index(99) != nil {
		t.Fatal("array Index(99) returned a value")
	}
	if root.Get("missing") != nil {
		t.Fatal("object Get(missing) returned a value")
	}
	out, err := obj.WriteString()
	if err != nil {
		t.Fatalf("Val.WriteString() error = %v", err)
	}
	if out != `{"a":1}` {
		t.Fatalf("object write = %q", out)
	}
	rawOut, err := obj.Write()
	if err != nil {
		t.Fatalf("Val.Write() error = %v", err)
	}
	if string(rawOut) != `{"a":1}` {
		t.Fatalf("object write bytes = %q", rawOut)
	}

	rawDoc, err := ReadString(`{"n":123}`, ReadNumberAsRaw)
	if err != nil {
		t.Fatalf("ReadString(raw number) error = %v", err)
	}
	defer rawDoc.Close()
	raw := rawDoc.Root().Get("n")
	if !raw.IsRaw() || raw.Raw() != "123" || raw.Len() != 3 {
		t.Fatalf("raw number: type=%v raw=%q len=%d", raw.Type(), raw.Raw(), raw.Len())
	}

	var nilVal *Val
	if nilVal.Type() != TypeNone || nilVal.Subtype() != SubtypeNone || nilVal.TypeDescription() != "none" {
		t.Fatalf("nil value metadata: type=%v subtype=%v desc=%q", nilVal.Type(), nilVal.Subtype(), nilVal.TypeDescription())
	}
	if nilVal.Bool() || nilVal.Uint() != 0 || nilVal.Int() != 0 || nilVal.Float() != 0 || nilVal.Num() != 0 || nilVal.String() != "" || nilVal.Raw() != "" || nilVal.Len() != 0 {
		t.Fatal("nil value accessors returned non-zero data")
	}
	if nilVal.Array() != nil || nilVal.Object() != nil || nilVal.Index(0) != nil || nilVal.Get("x") != nil || nilVal.EqualsString("") || nilVal.Equal(root) {
		t.Fatal("nil value collection/equality methods returned data")
	}
	if _, err := nilVal.Write(); err == nil {
		t.Fatal("nil value Write() error = nil")
	}
	if _, err := nilVal.WriteString(); err == nil {
		t.Fatal("nil value WriteString() error = nil")
	}
	if _, err := nilVal.Pointer(""); err == nil {
		t.Fatal("nil value Pointer() error = nil")
	}

	infDoc, err := ReadString(`Infinity`, ReadAllowInfAndNaN)
	if err != nil {
		t.Fatalf("ReadString(infinity) error = %v", err)
	}
	defer infDoc.Close()
	if _, err := infDoc.Root().Write(); err == nil {
		t.Fatal("Val.Write() infinity without flag error = nil")
	}
}

func TestImmutableDocumentConcurrentReads(t *testing.T) {
	doc, err := ReadString(`{"items":[{"name":"first"},{"name":"second"}],"ok":true}`)
	if err != nil {
		t.Fatalf("ReadString() error = %v", err)
	}
	defer doc.Close()

	root := doc.Root()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if !root.Get("ok").Bool() {
					t.Error("ok = false")
				}
				if got := root.Get("items").Index(1).Get("name").String(); got != "second" {
					t.Errorf("name = %q", got)
				}
				if _, err := root.Pointer("/items/0/name"); err != nil {
					t.Errorf("Pointer() error = %v", err)
				}
			}
		}()
	}
	wg.Wait()
}

func TestObjectOrderDuplicateKeysAndEmbeddedNUL(t *testing.T) {
	doc, err := ReadString("{\"dup\":1,\"s\":\"a\\u0000b\",\"dup\":2}")
	if err != nil {
		t.Fatalf("ReadString() error = %v", err)
	}
	defer doc.Close()

	root := doc.Root()
	if got := root.Get("dup").Int(); got != 1 {
		t.Fatalf("duplicate key lookup = %d", got)
	}
	members := root.Object()
	if len(members) != 3 {
		t.Fatalf("object member count = %d", len(members))
	}
	if members[0].Key.String() != "dup" || members[1].Key.String() != "s" || members[2].Key.String() != "dup" {
		t.Fatalf("object order = %q, %q, %q", members[0].Key.String(), members[1].Key.String(), members[2].Key.String())
	}
	if got := root.Get("s").String(); got != "a\x00b" {
		t.Fatalf("embedded NUL string = %q", got)
	}
	if got := root.Get("s").Len(); got != 3 {
		t.Fatalf("embedded NUL len = %d", got)
	}

	mut, err := NewMutDoc()
	if err != nil {
		t.Fatalf("NewMutDoc() error = %v", err)
	}
	defer mut.Close()
	obj := mut.Object()
	if !obj.AddKey("s", mut.String("x\x00y")) {
		t.Fatal("AddKey() failed")
	}
	mut.SetRoot(obj)
	out, err := mut.WriteString()
	if err != nil {
		t.Fatalf("mutable WriteString() error = %v", err)
	}
	if !strings.Contains(out, `\u0000`) {
		t.Fatalf("mutable embedded NUL output = %q", out)
	}
}

func TestReadErrorsIncludePosition(t *testing.T) {
	_, err := ReadString(`{"x":}`)
	if err == nil {
		t.Fatal("ReadString() error = nil")
	}
	var readErr *ReadError
	if !errors.As(err, &readErr) {
		t.Fatalf("error type = %T", err)
	}
	if readErr.Code == 0 || readErr.Message == "" {
		t.Fatalf("read error missing detail: %#v", readErr)
	}
	if !strings.Contains(err.Error(), "pos=") {
		t.Fatalf("read error string missing position: %v", err)
	}
}

func TestReadFlags(t *testing.T) {
	flagged := []struct {
		name  string
		input string
		flags []ReadFlag
		check func(*testing.T, *Doc)
	}{
		{
			name:  "trailing comma",
			input: `[1,2,]`,
			flags: []ReadFlag{ReadAllowTrailingCommas},
			check: func(t *testing.T, doc *Doc) {
				t.Helper()
				if got := doc.Root().Len(); got != 2 {
					t.Fatalf("array len = %d", got)
				}
			},
		},
		{
			name:  "comments",
			input: `/* lead */ {"a":1} // tail`,
			flags: []ReadFlag{ReadAllowComments, ReadStopWhenDone},
			check: func(t *testing.T, doc *Doc) {
				t.Helper()
				if got := doc.Root().Get("a").Uint(); got != 1 {
					t.Fatalf("a = %d", got)
				}
			},
		},
		{
			name:  "bom",
			input: "\xef\xbb\xbf{\"a\":1}",
			flags: []ReadFlag{ReadAllowBOM},
			check: func(t *testing.T, doc *Doc) {
				t.Helper()
				if got := doc.Root().Get("a").Uint(); got != 1 {
					t.Fatalf("a = %d", got)
				}
			},
		},
		{
			name:  "json5",
			input: `{key:'value', nums:[0x10, Infinity]}`,
			flags: []ReadFlag{ReadJSON5},
			check: func(t *testing.T, doc *Doc) {
				t.Helper()
				root := doc.Root()
				if got := root.Get("key").String(); got != "value" {
					t.Fatalf("key = %q", got)
				}
				if got := root.Get("nums").Index(0).Uint(); got != 16 {
					t.Fatalf("hex = %d", got)
				}
				if !math.IsInf(root.Get("nums").Index(1).Float(), 1) {
					t.Fatalf("inf = %v", root.Get("nums").Index(1).Float())
				}
			},
		},
	}
	for _, tc := range flagged {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ReadString(tc.input); err == nil {
				t.Fatal("ReadString() without flag error = nil")
			}
			doc, err := ReadString(tc.input, tc.flags...)
			if err != nil {
				t.Fatalf("ReadString() with flags error = %v", err)
			}
			defer doc.Close()
			tc.check(t, doc)
		})
	}

	invalidUTF8 := []byte{'"', 0xff, '"'}
	if _, err := Read(invalidUTF8); err == nil {
		t.Fatal("Read() invalid UTF-8 without flag error = nil")
	}
	doc, err := Read(invalidUTF8, ReadAllowInvalidUnicode)
	if err != nil {
		t.Fatalf("Read() invalid UTF-8 with flag error = %v", err)
	}
	defer doc.Close()
	if got := doc.Root().String(); got != string([]byte{0xff}) {
		t.Fatalf("invalid UTF-8 string = %q", got)
	}
	if _, err := doc.WriteString(WriteAllowInvalidUnicode); err != nil {
		t.Fatalf("WriteString() invalid UTF-8 with flag error = %v", err)
	}

	insitu, err := Read([]byte(`{"s":"kept"}`), ReadInsitu)
	if err != nil {
		t.Fatalf("Read() insitu error = %v", err)
	}
	insituRoot := insitu.Root()
	if got := insituRoot.Get("s").String(); got != "kept" {
		t.Fatalf("insitu string = %q", got)
	}
	insitu.Close()
	if insituRoot.Type() != TypeNone {
		t.Fatalf("insitu value type after close = %v", insituRoot.Type())
	}
}

func TestNumberFixtures(t *testing.T) {
	validCases := []struct {
		path  string
		flags []ReadFlag
	}{
		{path: filepath.Join("yyjson", "test", "data", "num", "int.txt")},
		{path: filepath.Join("yyjson", "test", "data", "num", "real_1.txt"), flags: []ReadFlag{ReadAllowInfAndNaN}},
		{path: filepath.Join("yyjson", "test", "data", "num", "hex(ext).txt"), flags: []ReadFlag{ReadAllowExtNumber, ReadBignumAsRaw}},
		{path: filepath.Join("yyjson", "test", "data", "num", "literal(ext).txt"), flags: []ReadFlag{ReadAllowExtNumber, ReadAllowInfAndNaN}},
	}
	for _, tc := range validCases {
		for _, input := range numberFixtureLines(t, tc.path) {
			doc, err := ReadString(input, tc.flags...)
			if err != nil {
				t.Fatalf("%s: ReadString(%q) error = %v", tc.path, input, err)
			}
			doc.Close()
		}
	}

	for _, path := range []string{
		filepath.Join("yyjson", "test", "data", "num", "int(fail).txt"),
		filepath.Join("yyjson", "test", "data", "num", "real(fail).txt"),
		filepath.Join("yyjson", "test", "data", "num", "literal(fail).txt"),
	} {
		for _, input := range numberFixtureLines(t, path) {
			if _, err := ReadString(input, ReadAllowExtNumber, ReadAllowInfAndNaN); err == nil {
				t.Fatalf("%s: ReadString(%q) error = nil", path, input)
			}
		}
	}
}

func TestJSONCorpusFixtures(t *testing.T) {
	for _, path := range corpusFiles(t, filepath.Join("yyjson", "test", "data", "json", "test_parsing"), "y_") {
		assertCorpusRoundTrip(t, path)
	}
	for _, path := range corpusFiles(t, filepath.Join("yyjson", "test", "data", "json", "test_parsing"), "n_") {
		assertCorpusRejects(t, path)
	}
	for _, path := range corpusFiles(t, filepath.Join("yyjson", "test", "data", "json", "test_checker"), "pass") {
		assertCorpusRoundTrip(t, path)
	}
	for _, path := range corpusFiles(t, filepath.Join("yyjson", "test", "data", "json", "test_checker"), "fail") {
		if strings.Contains(filepath.Base(path), "_EXCLUDE") {
			continue
		}
		assertCorpusRejects(t, path)
	}
	for _, path := range corpusFiles(t, filepath.Join("yyjson", "test", "data", "json", "test_roundtrip"), "roundtrip") {
		assertCorpusRoundTrip(t, path)
	}
}

func TestFileAndReaderHelpers(t *testing.T) {
	doc, err := ReadAll(strings.NewReader(`{"a":1}`))
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	defer doc.Close()
	if got := doc.Root().Get("a").Uint(); got != 1 {
		t.Fatalf("a = %d", got)
	}

	path := filepath.Join(t.TempDir(), "doc.json")
	if err := doc.WriteFile(path); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	fromFile, err := ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	defer fromFile.Close()
	if got := fromFile.Root().Get("a").Uint(); got != 1 {
		t.Fatalf("file a = %d", got)
	}

	if _, err := ReadAll(nil); err == nil {
		t.Fatal("ReadAll(nil) error = nil")
	}
	readErrReader, readErrWriter := io.Pipe()
	if err := readErrWriter.CloseWithError(errors.New("read failed")); err != nil {
		t.Fatalf("PipeWriter.CloseWithError() error = %v", err)
	}
	if _, err := ReadAll(readErrReader); err == nil {
		t.Fatal("ReadAll(closed pipe) error = nil")
	}
	if err := readErrReader.Close(); err != nil {
		t.Fatalf("PipeReader.Close() error = %v", err)
	}
	if _, err := doc.WriteTo(nil); err == nil {
		t.Fatal("Doc.WriteTo(nil) error = nil")
	}
	var buf bytes.Buffer
	n, err := doc.WriteTo(&buf)
	if err != nil {
		t.Fatalf("Doc.WriteTo(buffer) error = %v", err)
	}
	if n != int64(buf.Len()) || buf.String() != `{"a":1}` {
		t.Fatalf("Doc.WriteTo(buffer) = n:%d out:%q", n, buf.String())
	}
	pipeReader, pipeWriter := io.Pipe()
	if err := pipeReader.Close(); err != nil {
		t.Fatalf("PipeReader.Close() error = %v", err)
	}
	if n, err := doc.WriteTo(pipeWriter); err == nil || n != 0 {
		t.Fatalf("Doc.WriteTo(closed pipe) = n:%d err:%v", n, err)
	}
	if err := pipeWriter.Close(); err != nil {
		t.Fatalf("PipeWriter.Close() error = %v", err)
	}
	mut, err := Mutate(doc)
	if err != nil {
		t.Fatalf("Mutate() error = %v", err)
	}
	defer mut.Close()
	if _, err := mut.WriteTo(nil); err == nil {
		t.Fatal("MutDoc.WriteTo(nil) error = nil")
	}
	buf.Reset()
	n, err = mut.WriteTo(&buf)
	if err != nil {
		t.Fatalf("MutDoc.WriteTo(buffer) error = %v", err)
	}
	if n != int64(buf.Len()) || buf.String() != `{"a":1}` {
		t.Fatalf("MutDoc.WriteTo(buffer) = n:%d out:%q", n, buf.String())
	}
	pipeReader, pipeWriter = io.Pipe()
	if err := pipeReader.Close(); err != nil {
		t.Fatalf("PipeReader.Close() error = %v", err)
	}
	if n, err := mut.WriteTo(pipeWriter); err == nil || n != 0 {
		t.Fatalf("MutDoc.WriteTo(closed pipe) = n:%d err:%v", n, err)
	}
	if err := pipeWriter.Close(); err != nil {
		t.Fatalf("PipeWriter.Close() error = %v", err)
	}
	mutPath := filepath.Join(t.TempDir(), "mut.json")
	if err := mut.WriteFile(mutPath); err != nil {
		t.Fatalf("MutDoc.WriteFile() error = %v", err)
	}
	fromMutFile, err := ReadFile(mutPath)
	if err != nil {
		t.Fatalf("ReadFile(mutPath) error = %v", err)
	}
	defer fromMutFile.Close()
	if got := fromMutFile.Root().Get("a").Uint(); got != 1 {
		t.Fatalf("mutable file a = %d", got)
	}
}

func TestFileHelpersRejectEmbeddedNULPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.json")
	original := []byte(`{"existing":true}`)
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	nulPath := path + "\x00ignored"

	if _, err := ReadFile(nulPath); err == nil || !strings.Contains(err.Error(), "NUL") {
		t.Fatalf("ReadFile(NUL path) error = %v, want NUL rejection", err)
	}

	doc, err := ReadString(`{"a":1}`)
	if err != nil {
		t.Fatalf("ReadString() error = %v", err)
	}
	defer doc.Close()
	if err := doc.WriteFile(nulPath); err == nil || !strings.Contains(err.Error(), "NUL") {
		t.Fatalf("Doc.WriteFile(NUL path) error = %v, want NUL rejection", err)
	}

	mut, err := Mutate(doc)
	if err != nil {
		t.Fatalf("Mutate() error = %v", err)
	}
	defer mut.Close()
	if err := mut.WriteFile(nulPath); err == nil || !strings.Contains(err.Error(), "NUL") {
		t.Fatalf("MutDoc.WriteFile(NUL path) error = %v, want NUL rejection", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(data) != string(original) {
		t.Fatalf("prefix path content = %q, want %q", data, original)
	}
}

func TestWriteFlags(t *testing.T) {
	doc, err := ReadString(`{"s":"雪/雪"}`)
	if err != nil {
		t.Fatalf("ReadString() error = %v", err)
	}
	defer doc.Close()
	out, err := doc.WriteString(WritePrettyTwoSpaces, WriteEscapeUnicode, WriteEscapeSlashes, WriteNewlineAtEnd)
	if err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if !strings.Contains(out, "\n  ") || !strings.Contains(out, `\u96EA`) || !strings.Contains(out, `\/`) || !strings.HasSuffix(out, "\n") {
		t.Fatalf("formatted output missing requested flags: %q", out)
	}
	nums, err := ReadString(`{"n":1.234}`)
	if err != nil {
		t.Fatalf("ReadString(number) error = %v", err)
	}
	defer nums.Close()
	out, err = nums.WriteString(WriteFPToFixed(2))
	if err != nil {
		t.Fatalf("WriteString(fixed) error = %v", err)
	}
	if out != `{"n":1.23}` {
		t.Fatalf("fixed float output = %q", out)
	}
	if got := WriteFPToFixed(0); got != WriteNoFlag {
		t.Fatalf("WriteFPToFixed(0) = %#x, want WriteNoFlag", got)
	}
	if got := WriteFPToFixed(-1); got != WriteNoFlag {
		t.Fatalf("WriteFPToFixed(-1) = %#x, want WriteNoFlag", got)
	}
	if got := WriteFPToFixed(16); got != WriteNoFlag {
		t.Fatalf("WriteFPToFixed(16) = %#x, want WriteNoFlag", got)
	}
}

func TestWriteInfAndNaNFlags(t *testing.T) {
	parsedInf, err := ReadString(`Infinity`, ReadAllowInfAndNaN)
	if err != nil {
		t.Fatalf("ReadString(infinity) error = %v", err)
	}
	defer parsedInf.Close()
	if _, err := parsedInf.WriteString(); err == nil {
		t.Fatal("Doc.WriteString() infinity without flag error = nil")
	}
	var buf bytes.Buffer
	if _, err := parsedInf.WriteTo(&buf); err == nil {
		t.Fatal("Doc.WriteTo() infinity without flag error = nil")
	}

	doc, err := NewMutDoc()
	if err != nil {
		t.Fatalf("NewMutDoc() error = %v", err)
	}
	defer doc.Close()
	root := doc.Object()
	root.AddKey("n", doc.Float(math.Inf(1)))
	doc.SetRoot(root)
	if _, err := doc.WriteString(); err == nil {
		t.Fatal("WriteString() default error = nil")
	}
	buf.Reset()
	if _, err := doc.WriteTo(&buf); err == nil {
		t.Fatal("MutDoc.WriteTo() infinity without flag error = nil")
	}
	out, err := doc.WriteString(WriteAllowInfAndNaN)
	if err != nil {
		t.Fatalf("WriteString(allow inf) error = %v", err)
	}
	if out != `{"n":Infinity}` {
		t.Fatalf("allow inf output = %q", out)
	}
	out, err = doc.WriteString(WriteInfAndNaNAsNull)
	if err != nil {
		t.Fatalf("WriteString(inf as null) error = %v", err)
	}
	if out != `{"n":null}` {
		t.Fatalf("inf as null output = %q", out)
	}
}

func TestIncrementalReader(t *testing.T) {
	data := []byte(`{"items":[1,2,3]}`)
	reader, err := NewIncrReader(len(data))
	if err != nil {
		t.Fatalf("NewIncrReader() error = %v", err)
	}
	defer reader.Close()
	if got := reader.Len(); got != 0 {
		t.Fatalf("initial Len() = %d", got)
	}
	split := len(data) - 1
	if err := reader.Append(data[:split]); err != nil {
		t.Fatalf("Append(first) error = %v", err)
	}
	if got := reader.Len(); got != split {
		t.Fatalf("Len() after first append = %d, want %d", got, split)
	}
	_, err = reader.Read()
	if err == nil {
		t.Fatal("Read() error = nil for partial input")
	}
	var readErr *ReadError
	if !errors.As(err, &readErr) {
		t.Fatalf("partial error type = %T", err)
	}
	if readErr.Code != ReadErrorMore {
		t.Fatalf("partial code = %d, want %d", readErr.Code, ReadErrorMore)
	}
	if err := reader.Append(data[split:]); err != nil {
		t.Fatalf("Append(second) error = %v", err)
	}
	if got := reader.Len(); got != len(data) {
		t.Fatalf("Len() after second append = %d, want %d", got, len(data))
	}
	doc, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() full error = %v", err)
	}
	defer doc.Close()
	if got := doc.Root().Get("items").Index(2).Uint(); got != 3 {
		t.Fatalf("items[2] = %d", got)
	}
	if err := reader.Append([]byte(` `)); err == nil {
		t.Fatal("Append() after complete error = nil")
	}
	if _, err := reader.Read(); err == nil {
		t.Fatal("Read() after complete error = nil")
	}

	var nilReader *IncrReader
	if got := nilReader.Len(); got != 0 {
		t.Fatalf("nil reader Len() = %d", got)
	}
	nilReader.Close()
}

func TestIncrementalReaderInvalidInput(t *testing.T) {
	reader, err := NewIncrReader(len(`{"x":}`) + 1)
	if err != nil {
		t.Fatalf("NewIncrReader() error = %v", err)
	}
	defer reader.Close()
	if err := reader.Append([]byte(`{"x":}`)); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	_, err = reader.Read()
	if err == nil {
		t.Fatal("Read() error = nil for invalid input")
	}
	var readErr *ReadError
	if !errors.As(err, &readErr) {
		t.Fatalf("invalid error type = %T", err)
	}
	if readErr.Code == ReadErrorMore {
		t.Fatalf("invalid code = ReadErrorMore, want concrete parse error")
	}
	if err := reader.Append([]byte(` `)); err == nil {
		t.Fatal("Append() after invalid input error = nil")
	}
	if _, err := reader.Read(); err == nil {
		t.Fatal("Read() after invalid input error = nil")
	}
}

func TestIncrementalReaderCapacity(t *testing.T) {
	if _, err := NewIncrReader(-1); err == nil {
		t.Fatal("NewIncrReader(-1) error = nil")
	}

	reader, err := NewIncrReader(2)
	if err != nil {
		t.Fatalf("NewIncrReader() error = %v", err)
	}
	defer reader.Close()
	if err := reader.Append([]byte(`{}`)); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := reader.Append([]byte(` `)); err == nil {
		t.Fatal("Append() over capacity error = nil")
	}
}

func TestLifecycleCloseBehavior(t *testing.T) {
	doc, err := ReadString(`{"a":1}`)
	if err != nil {
		t.Fatalf("ReadString() error = %v", err)
	}
	root := doc.Root()
	doc.Close()
	doc.Close()
	if doc.Root() != nil {
		t.Fatal("Root() after close != nil")
	}
	if root.Type() != TypeNone {
		t.Fatalf("value type after close = %v", root.Type())
	}
	if doc.ReadSize() != 0 || doc.ValueCount() != 0 {
		t.Fatalf("closed document metadata = read:%d values:%d", doc.ReadSize(), doc.ValueCount())
	}
	if _, err := doc.Write(); err == nil {
		t.Fatal("Write() after close error = nil")
	}
	if _, err := doc.WriteString(); err == nil {
		t.Fatal("WriteString() after close error = nil")
	}
	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err == nil {
		t.Fatal("WriteTo() after close error = nil")
	}
	if err := doc.WriteFile(filepath.Join(t.TempDir(), "closed.json")); err == nil {
		t.Fatal("WriteFile() after close error = nil")
	}
	if _, err := doc.Pointer("/a"); err == nil {
		t.Fatal("Pointer() after close error = nil")
	}

	var nilDoc *Doc
	nilDoc.Close()

	mut, err := NewMutDoc()
	if err != nil {
		t.Fatalf("NewMutDoc() error = %v", err)
	}
	val := mut.Object()
	mut.Close()
	mut.Close()
	if mut.Root() != nil {
		t.Fatal("mutable Root() after close != nil")
	}
	if val.Type() != TypeNone {
		t.Fatalf("mutable value type after close = %v", val.Type())
	}
	if _, err := mut.Write(); err == nil {
		t.Fatal("mutable Write() after close error = nil")
	}
	if _, err := mut.WriteString(); err == nil {
		t.Fatal("mutable WriteString() after close error = nil")
	}
	buf.Reset()
	if _, err := mut.WriteTo(&buf); err == nil {
		t.Fatal("mutable WriteTo() after close error = nil")
	}
	if err := mut.WriteFile(filepath.Join(t.TempDir(), "closed-mut.json")); err == nil {
		t.Fatal("mutable WriteFile() after close error = nil")
	}
	if _, err := mut.Immutable(); err == nil {
		t.Fatal("mutable Immutable() after close error = nil")
	}
	if _, err := mut.Pointer("/"); err == nil {
		t.Fatal("mutable Pointer() after close error = nil")
	}
	if err := mut.PointerSet("/x", val); err == nil {
		t.Fatal("mutable PointerSet() after close error = nil")
	}
	if err := mut.PointerAdd("/x", val); err == nil {
		t.Fatal("mutable PointerAdd() after close error = nil")
	}
	if _, err := mut.PointerReplace("/x", val); err == nil {
		t.Fatal("mutable PointerReplace() after close error = nil")
	}
	if _, err := mut.PointerRemove("/x"); err == nil {
		t.Fatal("mutable PointerRemove() after close error = nil")
	}

	closedDoc, err := ReadString(`{"a":1}`)
	if err != nil {
		t.Fatalf("ReadString(closedDoc) error = %v", err)
	}
	closedRoot := closedDoc.Root()
	closedDoc.Close()
	if _, err := Mutate(nil); err == nil {
		t.Fatal("Mutate(nil) error = nil")
	}
	if _, err := Mutate(closedDoc); err == nil {
		t.Fatal("Mutate(closed) error = nil")
	}
	patchDoc, err := ReadString(`[{"op":"remove","path":"/a"}]`)
	if err != nil {
		t.Fatalf("ReadString(patchDoc) error = %v", err)
	}
	defer patchDoc.Close()
	if _, err := mut.Patch(closedRoot, patchDoc.Root()); err == nil {
		t.Fatal("Patch() after mutable close error = nil")
	}
	if _, err := mut.MergePatch(closedRoot, patchDoc.Root()); err == nil {
		t.Fatal("MergePatch() after mutable close error = nil")
	}
	target, err := NewMutDoc()
	if err != nil {
		t.Fatalf("NewMutDoc(target) error = %v", err)
	}
	defer target.Close()
	if _, err := target.Patch(closedRoot, patchDoc.Root()); err == nil {
		t.Fatal("Patch() with closed value error = nil")
	}
	if _, err := target.MergePatch(closedRoot, patchDoc.Root()); err == nil {
		t.Fatal("MergePatch() with closed value error = nil")
	}
}

func TestMutableConstructorsRejectInvalidDocuments(t *testing.T) {
	var nilDoc *MutDoc
	nilDoc.SetRoot(nil)
	if nilDoc.Null() != nil || nilDoc.True() != nil || nilDoc.False() != nil || nilDoc.Uint(1) != nil || nilDoc.String("x") != nil || nilDoc.Array() != nil || nilDoc.Object() != nil {
		t.Fatal("nil mutable document constructor returned a value")
	}

	doc, err := NewMutDoc()
	if err != nil {
		t.Fatalf("NewMutDoc() error = %v", err)
	}
	if doc.Root() != nil {
		t.Fatal("empty mutable document root != nil")
	}
	doc.Close()
	if doc.True() != nil || doc.False() != nil || doc.Bool(true) != nil || doc.Uint(1) != nil || doc.Int(1) != nil || doc.Float(1) != nil || doc.Raw(`{"x":1}`) != nil {
		t.Fatal("closed mutable document constructor returned a value")
	}
}

func TestMutableDocument(t *testing.T) {
	doc, err := NewMutDoc()
	if err != nil {
		t.Fatalf("NewMutDoc() error = %v", err)
	}
	defer doc.Close()

	root := doc.Object()
	items := doc.Array()
	if !items.Append(doc.Int(1)) || !items.Append(doc.String("two")) {
		t.Fatal("append failed")
	}
	if !items.Index(1).EqualsString("two") || !items.Index(1).Equal(items.Index(1)) {
		t.Fatal("mutable equality failed")
	}
	if !items.Rotate(1) || items.Index(0).String() != "two" {
		t.Fatal("array rotate failed")
	}
	if !items.Rotate(1) || items.Index(0).Int() != 1 {
		t.Fatal("array rotate restore failed")
	}
	scratch := doc.Object()
	if !scratch.AddKey("drop", doc.Bool(true)) || !scratch.ClearObject() || scratch.Len() != 0 {
		t.Fatal("object clear failed")
	}
	if !root.AddKey("items", items) || !root.AddKey("ok", doc.Bool(true)) {
		t.Fatal("object add failed")
	}
	if !root.RenameKey("ok", "ready") {
		t.Fatal("rename failed")
	}
	doc.SetRoot(root)

	out, err := doc.WriteString()
	if err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if out != `{"items":[1,"two"],"ready":true}` {
		t.Fatalf("write = %q", out)
	}

	immutable, err := doc.Immutable()
	if err != nil {
		t.Fatalf("Immutable() error = %v", err)
	}
	defer immutable.Close()
	if got := immutable.Root().Get("items").Index(1).String(); got != "two" {
		t.Fatalf("immutable item = %q", got)
	}
}

func TestMutableValueAccessorsCollectionsAndSerialization(t *testing.T) {
	doc, err := NewMutDoc()
	if err != nil {
		t.Fatalf("NewMutDoc() error = %v", err)
	}
	defer doc.Close()

	arr := doc.Array()
	if !arr.Append(doc.String("first")) || !arr.Append(doc.Uint(2)) {
		t.Fatal("array append failed")
	}
	if !arr.Insert(1, doc.String("middle")) {
		t.Fatal("array insert failed")
	}
	if removed := arr.Remove(1); removed == nil || removed.String() != "middle" {
		t.Fatalf("array remove = %#v", removed)
	}
	if old := arr.Replace(1, doc.Int(3)); old == nil || old.Uint() != 2 {
		t.Fatalf("array replace old = %#v", old)
	}
	elements := arr.Array()
	if len(elements) != 2 || elements[0].String() != "first" || elements[1].Int() != 3 {
		t.Fatalf("array values = %#v", elements)
	}
	scratch := doc.Array()
	if !scratch.Append(doc.True()) || !scratch.ClearArray() || scratch.Len() != 0 {
		t.Fatal("array clear failed")
	}

	root := doc.Object()
	nested := doc.Object()
	if !nested.AddKey("x", doc.Uint(1)) {
		t.Fatal("nested object add failed")
	}
	if !root.Add(doc.String("null"), doc.Null()) ||
		!root.AddKey("true", doc.True()) ||
		!root.AddKey("false", doc.False()) ||
		!root.AddKey("uint", doc.Uint(12)) ||
		!root.AddKey("int", doc.Int(-5)) ||
		!root.AddKey("float", doc.Float(2.5)) ||
		!root.AddKey("str", doc.String("before")) ||
		!root.AddKey("raw", doc.Raw(`{"nested":true}`)) ||
		!root.AddKey("arr", arr) ||
		!root.AddKey("obj", nested) {
		t.Fatal("object add failed")
	}
	if !root.PutKey("str", doc.String("text")) {
		t.Fatal("object PutKey failed")
	}
	if !root.Put(doc.String("put"), doc.Bool(true)) || !root.PutKey("put", doc.False()) {
		t.Fatal("object Put failed")
	}
	if removed := root.RemoveKey("put"); removed == nil || removed.Bool() {
		t.Fatalf("object RemoveKey = %#v", removed)
	}
	members := root.Object()
	if len(members) != 10 {
		t.Fatalf("object member count = %d", len(members))
	}
	doc.SetRoot(root)

	if !root.IsObj() || root.Subtype() != SubtypeNone || root.TypeDescription() != "object" {
		t.Fatalf("mutable root metadata: type=%v subtype=%v desc=%q", root.Type(), root.Subtype(), root.TypeDescription())
	}
	if !root.Get("null").IsNull() {
		t.Fatal("mutable null is not null")
	}
	if tv := root.Get("true"); !tv.IsBool() || !tv.Bool() || tv.Subtype() != SubtypeTrue {
		t.Fatalf("mutable true: bool=%v subtype=%v", tv.Bool(), tv.Subtype())
	}
	if fv := root.Get("false"); !fv.IsBool() || fv.Bool() || fv.Subtype() != SubtypeFalse {
		t.Fatalf("mutable false: bool=%v subtype=%v", fv.Bool(), fv.Subtype())
	}
	if uv := root.Get("uint"); !uv.IsNum() || uv.Uint() != 12 || uv.Subtype() != SubtypeUint {
		t.Fatalf("mutable uint: type=%v uint=%d subtype=%v", uv.Type(), uv.Uint(), uv.Subtype())
	}
	if iv := root.Get("int"); !iv.IsNum() || iv.Int() != -5 || iv.Subtype() != SubtypeSint {
		t.Fatalf("mutable int: type=%v int=%d subtype=%v", iv.Type(), iv.Int(), iv.Subtype())
	}
	if fv := root.Get("float"); !fv.IsNum() || fv.Float() != 2.5 || fv.Num() != 2.5 || fv.Subtype() != SubtypeReal {
		t.Fatalf("mutable float: type=%v float=%v num=%v subtype=%v", fv.Type(), fv.Float(), fv.Num(), fv.Subtype())
	}
	str := root.Get("str")
	if !str.IsStr() || str.String() != "text" || !str.EqualsString("text") || !str.Equal(str) || str.Len() != 4 {
		t.Fatalf("mutable string: type=%v string=%q len=%d", str.Type(), str.String(), str.Len())
	}
	raw := root.Get("raw")
	if !raw.IsRaw() || raw.Raw() != `{"nested":true}` {
		t.Fatalf("mutable raw: type=%v raw=%q", raw.Type(), raw.Raw())
	}
	if arrVal := root.Get("arr"); !arrVal.IsArr() || arrVal.Len() != 2 || arrVal.Index(1).Int() != 3 {
		t.Fatalf("mutable array: type=%v len=%d", arrVal.Type(), arrVal.Len())
	}
	if root.Get("arr").Index(99) != nil {
		t.Fatal("mutable array Index(99) returned a value")
	}
	if root.Get("arr").Replace(99, doc.String("missing")) != nil {
		t.Fatal("mutable array Replace(99) returned a value")
	}
	if root.Get("arr").Remove(99) != nil {
		t.Fatal("mutable array Remove(99) returned a value")
	}
	if obj := root.Get("obj"); !obj.IsObj() || obj.Len() != 1 || obj.Get("x").Uint() != 1 {
		t.Fatalf("mutable object: type=%v len=%d", obj.Type(), obj.Len())
	}
	if root.Get("obj").Get("missing") != nil {
		t.Fatal("mutable object Get(missing) returned a value")
	}
	if root.Get("obj").RemoveKey("missing") != nil {
		t.Fatal("mutable object RemoveKey(missing) returned a value")
	}

	strOut, err := str.WriteString()
	if err != nil {
		t.Fatalf("MutVal.WriteString() error = %v", err)
	}
	if strOut != `"text"` {
		t.Fatalf("mutable string write = %q", strOut)
	}
	rawOut, err := raw.Write()
	if err != nil {
		t.Fatalf("MutVal.Write() error = %v", err)
	}
	if string(rawOut) != `{"nested":true}` {
		t.Fatalf("mutable raw write = %q", rawOut)
	}
	if _, err := doc.Float(math.Inf(1)).Write(); err == nil {
		t.Fatal("MutVal.Write() infinity without flag error = nil")
	}
	immutable, err := root.Get("arr").Immutable()
	if err != nil {
		t.Fatalf("MutVal.Immutable() error = %v", err)
	}
	defer immutable.Close()
	if got := immutable.Root().Index(1).Int(); got != 3 {
		t.Fatalf("mutable value immutable item = %d", got)
	}

	var nilVal *MutVal
	if nilVal.Type() != TypeNone || nilVal.Subtype() != SubtypeNone || nilVal.TypeDescription() != "none" {
		t.Fatalf("nil mutable value metadata: type=%v subtype=%v desc=%q", nilVal.Type(), nilVal.Subtype(), nilVal.TypeDescription())
	}
	if nilVal.Bool() || nilVal.Uint() != 0 || nilVal.Int() != 0 || nilVal.Float() != 0 || nilVal.Num() != 0 || nilVal.String() != "" || nilVal.Raw() != "" || nilVal.Len() != 0 {
		t.Fatal("nil mutable value accessors returned non-zero data")
	}
	if nilVal.Array() != nil || nilVal.Object() != nil || nilVal.Index(0) != nil || nilVal.Get("x") != nil || nilVal.EqualsString("") || nilVal.Equal(str) {
		t.Fatal("nil mutable collection/equality methods returned data")
	}
	if nilVal.Append(str) || nilVal.Insert(0, str) || nilVal.Replace(0, str) != nil || nilVal.Remove(0) != nil || nilVal.ClearArray() || nilVal.Rotate(0) {
		t.Fatal("nil mutable array method succeeded")
	}
	if nilVal.Add(str, str) || nilVal.Put(str, str) || nilVal.AddKey("x", str) || nilVal.PutKey("x", str) || nilVal.RemoveKey("x") != nil || nilVal.ClearObject() || nilVal.RenameKey("x", "y") {
		t.Fatal("nil mutable object method succeeded")
	}
	if _, err := nilVal.Write(); err == nil {
		t.Fatal("nil mutable value Write() error = nil")
	}
	if _, err := nilVal.WriteString(); err == nil {
		t.Fatal("nil mutable value WriteString() error = nil")
	}
	if _, err := nilVal.Immutable(); err == nil {
		t.Fatal("nil mutable value Immutable() error = nil")
	}
}

func TestMutableDocumentRejectsCrossDocumentValues(t *testing.T) {
	left, err := NewMutDoc()
	if err != nil {
		t.Fatalf("NewMutDoc(left) error = %v", err)
	}
	defer left.Close()
	right, err := NewMutDoc()
	if err != nil {
		t.Fatalf("NewMutDoc(right) error = %v", err)
	}
	defer right.Close()

	root := left.Object()
	items := left.Array()
	if !items.Append(left.String("kept")) {
		t.Fatal("same-document append failed")
	}
	if !root.AddKey("items", items) || !root.AddKey("name", left.String("old")) {
		t.Fatal("same-document object add failed")
	}
	left.SetRoot(root)

	foreign := right.String("foreign")
	if items.Append(foreign) {
		t.Fatal("cross-document append succeeded")
	}
	if items.Insert(0, foreign) {
		t.Fatal("cross-document insert succeeded")
	}
	if old := items.Replace(0, foreign); old != nil {
		t.Fatal("cross-document replace succeeded")
	}
	if got := items.Index(0).String(); got != "kept" {
		t.Fatalf("array changed after rejected replace: %q", got)
	}
	if root.AddKey("foreign", foreign) {
		t.Fatal("cross-document AddKey succeeded")
	}
	if root.PutKey("name", foreign) {
		t.Fatal("cross-document PutKey succeeded")
	}
	if root.Add(right.String("key"), left.String("value")) {
		t.Fatal("cross-document key add succeeded")
	}
	if err := left.PointerSet("/name", foreign); err == nil {
		t.Fatal("cross-document PointerSet error = nil")
	}
	if err := left.PointerAdd("/added", foreign); err == nil {
		t.Fatal("cross-document PointerAdd error = nil")
	}
	if _, err := left.PointerReplace("/name", foreign); err == nil {
		t.Fatal("cross-document PointerReplace error = nil")
	}
	left.SetRoot(foreign)
	if got := left.Root().Get("name").String(); got != "old" {
		t.Fatalf("root changed after rejected SetRoot: %q", got)
	}
}

func TestMutablePointerOperations(t *testing.T) {
	orig, err := ReadString(`{"items":["a"],"name":"old"}`)
	if err != nil {
		t.Fatalf("ReadString(orig) error = %v", err)
	}
	defer orig.Close()
	mut, err := Mutate(orig)
	if err != nil {
		t.Fatalf("Mutate() error = %v", err)
	}
	defer mut.Close()

	got, err := mut.Pointer("/items/0")
	if err != nil {
		t.Fatalf("MutDoc.Pointer() error = %v", err)
	}
	if got.String() != "a" {
		t.Fatalf("MutDoc.Pointer() = %q", got.String())
	}
	if err := mut.PointerSet("/items/99", mut.String("missing")); err == nil {
		t.Fatal("PointerSet(missing index) error = nil")
	}
	if err := mut.PointerAdd("/items/1", mut.String("b")); err != nil {
		t.Fatalf("PointerAdd() error = %v", err)
	}
	if err := mut.PointerAdd("/items/99", mut.String("missing")); err == nil {
		t.Fatal("PointerAdd(missing index) error = nil")
	}
	old, err := mut.PointerReplace("/name", mut.String("new"))
	if err != nil {
		t.Fatalf("PointerReplace() error = %v", err)
	}
	if old.String() != "old" || mut.Root().Get("name").String() != "new" {
		t.Fatalf("PointerReplace() old=%q new=%q", old.String(), mut.Root().Get("name").String())
	}
	if _, err := mut.PointerReplace("/items/99", mut.String("missing")); err == nil {
		t.Fatal("PointerReplace(missing index) error = nil")
	}
	removed, err := mut.PointerRemove("/items/0")
	if err != nil {
		t.Fatalf("PointerRemove() error = %v", err)
	}
	if removed.String() != "a" || mut.Root().Get("items").Index(0).String() != "b" {
		t.Fatalf("PointerRemove() removed=%q remaining=%q", removed.String(), mut.Root().Get("items").Index(0).String())
	}
	if _, err := mut.PointerRemove("/items/99"); err == nil {
		t.Fatal("PointerRemove(missing index) error = nil")
	}
	if _, err := mut.Pointer("/missing"); err == nil {
		t.Fatal("MutDoc.Pointer(missing) error = nil")
	}
}

func TestPointerPatchAndMergePatch(t *testing.T) {
	orig, err := ReadString(`{"foo":["bar","baz"],"name":"old"}`)
	if err != nil {
		t.Fatalf("ReadString(orig) error = %v", err)
	}
	defer orig.Close()
	got, err := orig.Pointer("/foo/1")
	if err != nil {
		t.Fatalf("Pointer() error = %v", err)
	}
	if got.String() != "baz" {
		t.Fatalf("pointer = %q", got.String())
	}

	mut, err := Mutate(orig)
	if err != nil {
		t.Fatalf("Mutate() error = %v", err)
	}
	defer mut.Close()
	if err := mut.PointerSet("/name", mut.String("new")); err != nil {
		t.Fatalf("PointerSet() error = %v", err)
	}
	if got := mut.Root().Get("name").String(); got != "new" {
		t.Fatalf("mut pointer = %q", got)
	}

	patchDoc, err := ReadString(`[{"op":"replace","path":"/name","value":"patched"}]`)
	if err != nil {
		t.Fatalf("ReadString(patch) error = %v", err)
	}
	defer patchDoc.Close()
	patched, err := mut.Patch(orig.Root(), patchDoc.Root())
	if err != nil {
		t.Fatalf("Patch() error = %v", err)
	}
	mut.SetRoot(patched)
	if got := mut.Root().Get("name").String(); got != "patched" {
		t.Fatalf("patch name = %q", got)
	}

	mergeDoc, err := ReadString(`{"extra":true}`)
	if err != nil {
		t.Fatalf("ReadString(merge) error = %v", err)
	}
	defer mergeDoc.Close()
	merged, err := mut.MergePatch(orig.Root(), mergeDoc.Root())
	if err != nil {
		t.Fatalf("MergePatch() error = %v", err)
	}
	mut.SetRoot(merged)
	if !mut.Root().Get("extra").Bool() {
		t.Fatal("merge extra = false")
	}
}

func TestPointerAndPatchErrorsAreTyped(t *testing.T) {
	doc, err := ReadString(`{"name":"old"}`)
	if err != nil {
		t.Fatalf("ReadString(doc) error = %v", err)
	}
	defer doc.Close()
	_, err = doc.Pointer("/missing")
	if err == nil {
		t.Fatal("Pointer() error = nil")
	}
	var ptrErr *PointerError
	if !errors.As(err, &ptrErr) {
		t.Fatalf("pointer error type = %T", err)
	}
	if ptrErr.Operation != "get" || ptrErr.Path != "/missing" || ptrErr.Message == "" {
		t.Fatalf("pointer error missing context: %#v", ptrErr)
	}

	patchDoc, err := ReadString(`[{"op":"remove","path":"/missing"}]`)
	if err != nil {
		t.Fatalf("ReadString(patch) error = %v", err)
	}
	defer patchDoc.Close()
	mut, err := NewMutDoc()
	if err != nil {
		t.Fatalf("NewMutDoc() error = %v", err)
	}
	defer mut.Close()
	_, err = mut.Patch(doc.Root(), patchDoc.Root())
	if err == nil {
		t.Fatal("Patch() error = nil")
	}
	var patchErr *PatchError
	if !errors.As(err, &patchErr) {
		t.Fatalf("patch error type = %T", err)
	}
	if patchErr.Message == "" {
		t.Fatalf("patch error missing message: %#v", patchErr)
	}
}

func TestEncodingJSONCompatibilitySurface(t *testing.T) {
	type sample struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	value := sample{Name: "yyjson", N: 7}
	got, err := Marshal(value)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want, err := stdjson.Marshal(value)
	if err != nil {
		t.Fatalf("encoding/json Marshal() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Marshal() = %s, want %s", got, want)
	}
	var decoded sample
	if err := Unmarshal(got, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded != value {
		t.Fatalf("decoded = %#v", decoded)
	}
	if !Valid(got) || Valid([]byte(`{"x":}`)) {
		t.Fatal("Valid() mismatch")
	}
	raw := RawMessage(`{"raw":true}`)
	got, err = Marshal(struct {
		Raw RawMessage `json:"raw"`
		C   customJSON `json:"c"`
	}{Raw: raw})
	if err != nil {
		t.Fatalf("Marshal(raw/custom) error = %v", err)
	}
	want, err = stdjson.Marshal(struct {
		Raw stdjson.RawMessage `json:"raw"`
		C   customJSON         `json:"c"`
	}{Raw: stdjson.RawMessage(raw)})
	if err != nil {
		t.Fatalf("encoding/json Marshal(raw/custom) error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("Marshal(raw/custom) = %s, want %s", got, want)
	}
	dec := NewDecoder(strings.NewReader(`{"n":12345678901234567890}`))
	dec.UseNumber()
	var withNumber map[string]any
	if err := dec.Decode(&withNumber); err != nil {
		t.Fatalf("Decoder.Decode() error = %v", err)
	}
	if _, ok := withNumber["n"].(Number); !ok {
		t.Fatalf("decoded number type = %T", withNumber["n"])
	}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(map[string]string{"s": "<&>"}); err != nil {
		t.Fatalf("Encoder.Encode() error = %v", err)
	}
	var wantBuf bytes.Buffer
	wantEnc := stdjson.NewEncoder(&wantBuf)
	wantEnc.SetEscapeHTML(false)
	if err := wantEnc.Encode(map[string]string{"s": "<&>"}); err != nil {
		t.Fatalf("encoding/json Encoder.Encode() error = %v", err)
	}
	if buf.String() != wantBuf.String() {
		t.Fatalf("Encoder.Encode() = %q, want %q", buf.String(), wantBuf.String())
	}
}

func TestEncodingJSONDecoderTokenParity(t *testing.T) {
	dec := NewDecoder(strings.NewReader(`{"a":[1,2],"b":false}`))

	assertJSONToken(t, dec, stdjson.Delim('{'))
	if !dec.More() {
		t.Fatal("More() at object start = false, want true")
	}
	assertJSONToken(t, dec, "a")
	assertJSONToken(t, dec, stdjson.Delim('['))
	if !dec.More() {
		t.Fatal("More() at array start = false, want true")
	}
	assertJSONToken(t, dec, float64(1))
	if !dec.More() {
		t.Fatal("More() after first array element = false, want true")
	}
	assertJSONToken(t, dec, float64(2))
	if dec.More() {
		t.Fatal("More() after final array element = true, want false")
	}
	assertJSONToken(t, dec, stdjson.Delim(']'))
	if !dec.More() {
		t.Fatal("More() before final object member = false, want true")
	}
	assertJSONToken(t, dec, "b")
	assertJSONToken(t, dec, false)
	if dec.More() {
		t.Fatal("More() after final object member = true, want false")
	}
	assertJSONToken(t, dec, stdjson.Delim('}'))
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		t.Fatalf("Token() after document error = %v, want EOF", err)
	}
}

func TestEncodingJSONStrictDecodeParity(t *testing.T) {
	type sample struct {
		Name string `json:"name"`
	}
	const input = `{"name":"yyjson","extra":1}`

	dec := NewDecoder(strings.NewReader(input))
	dec.DisallowUnknownFields()
	var got sample
	gotErr := dec.Decode(&got)

	wantDec := stdjson.NewDecoder(strings.NewReader(input))
	wantDec.DisallowUnknownFields()
	var want sample
	wantErr := wantDec.Decode(&want)

	if got != want {
		t.Fatalf("Decode() value = %#v, want %#v", got, want)
	}
	if (gotErr == nil) != (wantErr == nil) {
		t.Fatalf("Decode() error = %v, want %v", gotErr, wantErr)
	}
	if gotErr != nil && gotErr.Error() != wantErr.Error() {
		t.Fatalf("Decode() error = %q, want %q", gotErr, wantErr)
	}
}

func TestEncodingJSONEncoderFormattingParity(t *testing.T) {
	value := map[string]any{
		"html":   "<&>",
		"items":  []int{1, 2},
		"nested": map[string]string{"name": "yyjson"},
	}

	tests := []struct {
		name          string
		indentPrefix  string
		indentValue   string
		disableEscape bool
	}{
		{name: "default"},
		{name: "indent", indentPrefix: "", indentValue: "  "},
		{name: "no html escape", disableEscape: true},
		{name: "indent no html escape", indentPrefix: "", indentValue: "\t", disableEscape: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got bytes.Buffer
			enc := NewEncoder(&got)
			if tt.indentValue != "" {
				enc.SetIndent(tt.indentPrefix, tt.indentValue)
			}
			if tt.disableEscape {
				enc.SetEscapeHTML(false)
			}
			if err := enc.Encode(value); err != nil {
				t.Fatalf("Encoder.Encode() error = %v", err)
			}

			var want bytes.Buffer
			wantEnc := stdjson.NewEncoder(&want)
			if tt.indentValue != "" {
				wantEnc.SetIndent(tt.indentPrefix, tt.indentValue)
			}
			if tt.disableEscape {
				wantEnc.SetEscapeHTML(false)
			}
			if err := wantEnc.Encode(value); err != nil {
				t.Fatalf("encoding/json Encoder.Encode() error = %v", err)
			}

			if got.String() != want.String() {
				t.Fatalf("Encoder.Encode() = %q, want %q", got.String(), want.String())
			}
		})
	}
}

func assertJSONToken(t *testing.T, dec *stdjson.Decoder, want stdjson.Token) {
	t.Helper()
	got, err := dec.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if got != want {
		t.Fatalf("Token() = %#v (%T), want %#v (%T)", got, got, want, want)
	}
}

func TestFileWriteErrorMentionsPath(t *testing.T) {
	doc, err := ReadString(`{"a":1}`)
	if err != nil {
		t.Fatalf("ReadString() error = %v", err)
	}
	defer doc.Close()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "existing"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err = doc.WriteFile(filepath.Join(dir, "missing", "doc.json"))
	if err == nil {
		t.Fatal("WriteFile() error = nil")
	}
	if !strings.Contains(err.Error(), filepath.Join(dir, "missing", "doc.json")) {
		t.Fatalf("error does not mention path: %v", err)
	}
}

func TestErrorStringsAndVersion(t *testing.T) {
	if Version() == 0 {
		t.Fatal("Version() = 0")
	}

	var readErr *ReadError
	var writeErr *WriteError
	var pointerErr *PointerError
	var patchErr *PatchError
	if readErr.Error() != "" || writeErr.Error() != "" || pointerErr.Error() != "" || patchErr.Error() != "" {
		t.Fatal("nil error receiver returned non-empty string")
	}
	if readError(nil, "read", "") != nil || writeError(nil, "write", "") != nil || pointerError(nil, "get", "") != nil || patchError(nil) != nil {
		t.Fatal("nil bind error conversion returned an error")
	}

	readText := (&ReadError{
		Code:      ReadErrorUnexpectedCharacter,
		Message:   "bad character",
		Position:  7,
		Operation: "read file",
		Path:      "doc.json",
	}).Error()
	if !strings.Contains(readText, "read file") || !strings.Contains(readText, "doc.json") || !strings.Contains(readText, "pos=7") {
		t.Fatalf("read error string = %q", readText)
	}

	writeText := (&WriteError{
		Code:      1,
		Message:   "bad write",
		Operation: "write file",
		Path:      "doc.json",
	}).Error()
	if !strings.Contains(writeText, "write file") || !strings.Contains(writeText, "doc.json") || !strings.Contains(writeText, "bad write") {
		t.Fatalf("write error string = %q", writeText)
	}

	pointerText := (&PointerError{
		Code:      2,
		Message:   "missing",
		Position:  3,
		Operation: "get",
		Path:      "/missing",
	}).Error()
	if !strings.Contains(pointerText, "get") || !strings.Contains(pointerText, "/missing") || !strings.Contains(pointerText, "pos=3") {
		t.Fatalf("pointer error string = %q", pointerText)
	}

	patchText := (&PatchError{
		Code:    4,
		Index:   5,
		Message: "bad patch",
	}).Error()
	if !strings.Contains(patchText, "op=5") || !strings.Contains(patchText, "bad patch") {
		t.Fatalf("patch error string = %q", patchText)
	}

	patchText = (&PatchError{
		Code:    4,
		Index:   5,
		Message: "bad patch",
		Pointer: PointerError{
			Code:      6,
			Message:   "missing pointer",
			Position:  7,
			Operation: "patch",
			Path:      "/x",
		},
	}).Error()
	if !strings.Contains(patchText, "op=5") || !strings.Contains(patchText, "missing pointer") || !strings.Contains(patchText, "/x") {
		t.Fatalf("patch error string = %q", patchText)
	}
}

func numberFixtureLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		t.Fatalf("%s: no fixture lines", path)
	}
	return lines
}

func corpusFiles(t *testing.T, dir, prefix string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%s) error = %v", dir, err)
	}
	var out []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		out = append(out, filepath.Join(dir, entry.Name()))
	}
	return out
}

func assertCorpusRoundTrip(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: ReadFile error = %v", path, err)
	}
	doc, err := Read(data)
	if err != nil {
		t.Fatalf("%s: Read() error = %v", path, err)
	}
	defer doc.Close()
	out, err := doc.Write()
	if err != nil {
		t.Fatalf("%s: Write() error = %v", path, err)
	}
	round, err := Read(out)
	if err != nil {
		t.Fatalf("%s: Read(write output) error = %v; output=%q", path, err, out)
	}
	defer round.Close()
	if !doc.Root().Equal(round.Root()) {
		t.Fatalf("%s: round trip changed value; output=%q", path, out)
	}
}

func assertCorpusRejects(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: ReadFile error = %v", path, err)
	}
	if doc, err := Read(data); err == nil {
		doc.Close()
		t.Fatalf("%s: Read() accepted invalid JSON", path)
	}
}
