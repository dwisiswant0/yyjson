//go:build cgo

package benchmarks

/*
#cgo CFLAGS: -O3 -DNDEBUG -I../yyjson/src
#include <stdlib.h>
#include "native_yyjson.h"
*/
import "C"

import (
	"bytes"
	"io"
	"runtime"
	"testing"
	"unsafe"
)

type nativeYYJSONDoc struct {
	ptr *C.yyjson_doc
}

func benchmarkReadNativeYYJSON(b *testing.B, data []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := readNativeYYJSON(b, data)
		uintSink = uint64(C.bench_yyjson_doc_value_count(doc.ptr))
		doc.close()
	}
}

func benchmarkReadAllNativeYYJSON(b *testing.B, data []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := io.ReadAll(bytes.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
		doc := readNativeYYJSON(b, out)
		uintSink = uint64(C.bench_yyjson_doc_value_count(doc.ptr))
		doc.close()
	}
}

func benchmarkWriteNativeYYJSON(b *testing.B, data []byte) {
	doc := readNativeYYJSON(b, data)
	defer doc.close()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytesSink = nativeWrite(b, doc)
	}
}

func benchmarkRoundTripNativeYYJSON(b *testing.B, data []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := readNativeYYJSON(b, data)
		bytesSink = nativeWrite(b, doc)
		doc.close()
	}
}

func benchmarkDOMReadNativeYYJSON(b *testing.B, data []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := readNativeYYJSON(b, data)
		uintSink = inspectNativeYYJSONDOM(doc)
		doc.close()
	}
}

func benchmarkDOMWriteNativeYYJSON(b *testing.B, data []byte) {
	doc := readNativeYYJSON(b, data)
	defer doc.close()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytesSink = nativeWrite(b, doc)
	}
}

func readNativeYYJSON(tb testing.TB, data []byte) nativeYYJSONDoc {
	tb.Helper()
	if len(data) == 0 {
		tb.Fatal("native yyjson input is empty")
	}
	doc := C.bench_yyjson_read_copy((*C.char)(unsafe.Pointer(unsafe.SliceData(data))), C.size_t(len(data)))
	runtime.KeepAlive(data)
	if doc == nil {
		tb.Fatal("native yyjson read failed")
	}
	return nativeYYJSONDoc{ptr: doc}
}

func nativeWrite(tb testing.TB, doc nativeYYJSONDoc) []byte {
	tb.Helper()
	var outLen C.size_t
	out := C.bench_yyjson_write_doc(doc.ptr, &outLen)
	if out == nil {
		tb.Fatal("native yyjson write failed")
	}
	defer C.bench_yyjson_free(unsafe.Pointer(out))

	maxInt := int(^uint(0) >> 1)
	if uint64(outLen) > uint64(maxInt) {
		tb.Fatalf("native yyjson write output length %d exceeds max int %d", uint64(outLen), maxInt)
	}
	return append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(out)), int(outLen))...)
}

func inspectNativeYYJSONDOM(doc nativeYYJSONDoc) uint64 {
	root := C.bench_yyjson_doc_root(doc.ptr)
	if root == nil {
		return 0
	}
	items := nativeObjGet(root, "items")
	metrics := nativeObjGet(root, "metrics")
	if items == nil || metrics == nil {
		return 0
	}
	count := nativeObjGet(metrics, "count")
	active := nativeObjGet(root, "active")
	if count == nil || active == nil {
		return 0
	}

	total := uint64(C.bench_yyjson_arr_size(items)) + uint64(C.bench_yyjson_get_uint(count))
	if C.bench_yyjson_get_bool(active) != 0 {
		total++
	}
	return total
}

func nativeObjGet(obj *C.yyjson_val, key string) *C.yyjson_val {
	ckey := C.CString(key)
	defer C.free(unsafe.Pointer(ckey))
	return C.bench_yyjson_obj_getn(obj, ckey, C.size_t(len(key)))
}

func (d *nativeYYJSONDoc) close() {
	if d.ptr != nil {
		C.bench_yyjson_doc_free(d.ptr)
		d.ptr = nil
	}
}
