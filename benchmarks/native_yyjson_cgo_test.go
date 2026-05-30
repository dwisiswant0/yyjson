//go:build cgo

package benchmarks

import (
	"reflect"
	"testing"
)

func TestNativeYYJSONReadWrite(t *testing.T) {
	fixture := fixtures[0]
	doc := readNativeYYJSON(t, fixture.data)
	defer doc.close()

	out := nativeWrite(t, doc)
	got := decodeStd(t, out)
	want := decodeStd(t, fixture.data)
	if !reflect.DeepEqual(got, want) {
		t.Fatal("native yyjson round trip changed value")
	}
}
