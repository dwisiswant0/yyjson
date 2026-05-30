package benchmarks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	sonicjson "github.com/bytedance/sonic"
	goccyjson "github.com/goccy/go-json"
	jsoniter "github.com/json-iterator/go"
	simdjson "github.com/minio/simdjson-go"
	segmentiojson "github.com/segmentio/encoding/json"
	"go.dw1.io/yyjson"
)

type benchFixture struct {
	name string
	size int
	data []byte
}

type benchPayload struct {
	ID      int            `json:"id"`
	Name    string         `json:"name"`
	Active  bool           `json:"active"`
	Tags    []string       `json:"tags"`
	Items   []benchItem    `json:"items"`
	Metrics map[string]any `json:"metrics"`
	Padding string         `json:"padding"`
}

type benchItem struct {
	Index  int      `json:"index"`
	Name   string   `json:"name"`
	Score  float64  `json:"score"`
	Flags  []bool   `json:"flags"`
	Labels []string `json:"labels"`
}

var fixtures = []benchFixture{
	makeFixture("500KiB", 500*kib),
	makeFixture("1MiB", 1*mib),
	makeFixture("4MiB", 4*mib),
	makeFixture("16MiB", 16*mib),
}

const (
	kib = 1024
	mib = 1024 * kib
)

var (
	anySink   any
	bytesSink []byte
	uintSink  uint64
)

type benchmarkCase struct {
	name string
	run  func(*testing.B, []byte)
}

type jsonDecoder interface {
	Decode(any) error
}

type unmarshalFunc func([]byte, any) error
type marshalFunc func(any) ([]byte, error)
type newDecoderFunc func(io.Reader) jsonDecoder

var readBenchmarks = []benchmarkCase{
	{name: "std", run: func(b *testing.B, data []byte) {
		benchmarkUnmarshal(b, data, json.Unmarshal)
	}},
	{name: "json-iterator", run: func(b *testing.B, data []byte) {
		benchmarkUnmarshal(b, data, jsoniter.Unmarshal)
	}},
	{name: "segmentio-json", run: func(b *testing.B, data []byte) {
		benchmarkUnmarshal(b, data, segmentiojson.Unmarshal)
	}},
	{name: "go-json", run: func(b *testing.B, data []byte) {
		benchmarkUnmarshal(b, data, goccyjson.Unmarshal)
	}},
	{name: "sonic", run: func(b *testing.B, data []byte) {
		benchmarkUnmarshal(b, data, sonicjson.Unmarshal)
	}},
	{name: "simdjson", run: benchmarkReadSIMDJSON},
	{name: "native-yyjson", run: benchmarkReadNativeYYJSON},
	{name: "yyjson", run: benchmarkReadYYJSON},
}

var readAllBenchmarks = []benchmarkCase{
	{name: "std", run: func(b *testing.B, data []byte) {
		benchmarkDecode(b, data, newStdDecoder)
	}},
	{name: "json-iterator", run: func(b *testing.B, data []byte) {
		benchmarkDecode(b, data, newJSONIteratorDecoder)
	}},
	{name: "segmentio-json", run: func(b *testing.B, data []byte) {
		benchmarkDecode(b, data, newSegmentioDecoder)
	}},
	{name: "go-json", run: func(b *testing.B, data []byte) {
		benchmarkDecode(b, data, newGoJSONDecoder)
	}},
	{name: "sonic", run: func(b *testing.B, data []byte) {
		benchmarkDecode(b, data, newSonicDecoder)
	}},
	{name: "simdjson", run: benchmarkReadAllSIMDJSON},
	{name: "native-yyjson", run: benchmarkReadAllNativeYYJSON},
	{name: "yyjson", run: benchmarkReadAllYYJSON},
}

var writeBenchmarks = []benchmarkCase{
	{name: "std", run: func(b *testing.B, data []byte) {
		benchmarkMarshal(b, data, json.Marshal)
	}},
	{name: "json-iterator", run: func(b *testing.B, data []byte) {
		benchmarkMarshal(b, data, jsoniter.Marshal)
	}},
	{name: "segmentio-json", run: func(b *testing.B, data []byte) {
		benchmarkMarshal(b, data, segmentiojson.Marshal)
	}},
	{name: "go-json", run: func(b *testing.B, data []byte) {
		benchmarkMarshal(b, data, goccyjson.Marshal)
	}},
	{name: "sonic", run: func(b *testing.B, data []byte) {
		benchmarkMarshal(b, data, sonicjson.Marshal)
	}},
	{name: "simdjson", run: benchmarkWriteSIMDJSON},
	{name: "native-yyjson", run: benchmarkWriteNativeYYJSON},
	{name: "yyjson", run: benchmarkWriteYYJSON},
}

var roundTripBenchmarks = []benchmarkCase{
	{name: "std", run: func(b *testing.B, data []byte) {
		benchmarkRoundTrip(b, data, json.Unmarshal, json.Marshal)
	}},
	{name: "json-iterator", run: func(b *testing.B, data []byte) {
		benchmarkRoundTrip(b, data, jsoniter.Unmarshal, jsoniter.Marshal)
	}},
	{name: "segmentio-json", run: func(b *testing.B, data []byte) {
		benchmarkRoundTrip(b, data, segmentiojson.Unmarshal, segmentiojson.Marshal)
	}},
	{name: "go-json", run: func(b *testing.B, data []byte) {
		benchmarkRoundTrip(b, data, goccyjson.Unmarshal, goccyjson.Marshal)
	}},
	{name: "sonic", run: func(b *testing.B, data []byte) {
		benchmarkRoundTrip(b, data, sonicjson.Unmarshal, sonicjson.Marshal)
	}},
	{name: "simdjson", run: benchmarkRoundTripSIMDJSON},
	{name: "native-yyjson", run: benchmarkRoundTripNativeYYJSON},
	{name: "yyjson", run: benchmarkRoundTripYYJSON},
}

var domReadBenchmarks = []benchmarkCase{
	{name: "std", run: benchmarkDOMReadStd},
	{name: "native-yyjson", run: benchmarkDOMReadNativeYYJSON},
	{name: "yyjson", run: benchmarkDOMReadYYJSON},
}

var domWriteBenchmarks = []benchmarkCase{
	{name: "std", run: benchmarkDOMWriteStd},
	{name: "native-yyjson", run: benchmarkDOMWriteNativeYYJSON},
	{name: "yyjson", run: benchmarkDOMWriteYYJSON},
}

func BenchmarkRead(b *testing.B) {
	benchmarkCases(b, readBenchmarks)
}

func BenchmarkReadAll(b *testing.B) {
	benchmarkCases(b, readAllBenchmarks)
}

func BenchmarkWrite(b *testing.B) {
	benchmarkCases(b, writeBenchmarks)
}

func BenchmarkRoundTrip(b *testing.B) {
	benchmarkCases(b, roundTripBenchmarks)
}

func BenchmarkDOMRead(b *testing.B) {
	benchmarkCases(b, domReadBenchmarks)
}

func BenchmarkDOMWrite(b *testing.B) {
	benchmarkCases(b, domWriteBenchmarks)
}

func newStdDecoder(r io.Reader) jsonDecoder {
	return json.NewDecoder(r)
}

func newJSONIteratorDecoder(r io.Reader) jsonDecoder {
	return jsoniter.NewDecoder(r)
}

func newSegmentioDecoder(r io.Reader) jsonDecoder {
	return segmentiojson.NewDecoder(r)
}

func newGoJSONDecoder(r io.Reader) jsonDecoder {
	return goccyjson.NewDecoder(r)
}

func newSonicDecoder(r io.Reader) jsonDecoder {
	return sonicjson.ConfigDefault.NewDecoder(r)
}

func benchmarkCases(b *testing.B, cases []benchmarkCase) {
	for _, fixture := range fixtures {
		fixture := fixture
		b.Run(fixture.name, func(b *testing.B) {
			for _, bc := range cases {
				bc := bc
				b.Run(bc.name, func(b *testing.B) {
					bc.run(b, fixture.data)
				})
			}
		})
	}
}

func benchmarkUnmarshal(b *testing.B, data []byte, unmarshal unmarshalFunc) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out any
		if err := unmarshal(data, &out); err != nil {
			b.Fatal(err)
		}
		anySink = out
	}
}

func benchmarkDecode(b *testing.B, data []byte, newDecoder newDecoderFunc) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out any
		if err := newDecoder(bytes.NewReader(data)).Decode(&out); err != nil {
			b.Fatal(err)
		}
		anySink = out
	}
}

func benchmarkMarshal(b *testing.B, data []byte, marshal marshalFunc) {
	b.Helper()
	value := decodeStd(b, data)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := marshal(value)
		if err != nil {
			b.Fatal(err)
		}
		bytesSink = out
	}
}

func benchmarkRoundTrip(b *testing.B, data []byte, unmarshal unmarshalFunc, marshal marshalFunc) {
	b.Helper()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var value any
		if err := unmarshal(data, &value); err != nil {
			b.Fatal(err)
		}
		out, err := marshal(value)
		if err != nil {
			b.Fatal(err)
		}
		bytesSink = out
	}
}

func benchmarkReadSIMDJSON(b *testing.B, data []byte) {
	skipUnsupportedSIMDJSON(b)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parsed, err := simdjson.Parse(data, nil)
		if err != nil {
			b.Fatal(err)
		}
		uintSink = uint64(len(parsed.Tape))
	}
}

func benchmarkReadAllSIMDJSON(b *testing.B, data []byte) {
	skipUnsupportedSIMDJSON(b)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := io.ReadAll(bytes.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
		parsed, err := simdjson.Parse(out, nil)
		if err != nil {
			b.Fatal(err)
		}
		uintSink = uint64(len(parsed.Tape))
	}
}

func benchmarkWriteSIMDJSON(b *testing.B, data []byte) {
	parsed := readSIMDJSON(b, data)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := marshalSIMDJSON(parsed)
		if err != nil {
			b.Fatal(err)
		}
		bytesSink = out
	}
}

func benchmarkRoundTripSIMDJSON(b *testing.B, data []byte) {
	skipUnsupportedSIMDJSON(b)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parsed, err := simdjson.Parse(data, nil)
		if err != nil {
			b.Fatal(err)
		}
		out, err := marshalSIMDJSON(parsed)
		if err != nil {
			b.Fatal(err)
		}
		bytesSink = out
	}
}

func benchmarkReadYYJSON(b *testing.B, data []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc, err := yyjson.Read(data)
		if err != nil {
			b.Fatal(err)
		}
		uintSink = doc.ValueCount()
		doc.Close()
	}
}

func benchmarkReadAllYYJSON(b *testing.B, data []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc, err := yyjson.ReadAll(bytes.NewReader(data))
		if err != nil {
			b.Fatal(err)
		}
		uintSink = doc.ValueCount()
		doc.Close()
	}
}

func benchmarkWriteYYJSON(b *testing.B, data []byte) {
	doc := readYYJSON(b, data)
	defer doc.Close()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := doc.Write()
		if err != nil {
			b.Fatal(err)
		}
		bytesSink = out
	}
	b.StopTimer()
}

func benchmarkRoundTripYYJSON(b *testing.B, data []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc, err := yyjson.Read(data)
		if err != nil {
			b.Fatal(err)
		}
		out, err := doc.Write()
		doc.Close()
		if err != nil {
			b.Fatal(err)
		}
		bytesSink = out
	}
}

func benchmarkDOMReadStd(b *testing.B, data []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var value any
		if err := json.Unmarshal(data, &value); err != nil {
			b.Fatal(err)
		}
		uintSink = inspectStdDOM(b, value)
	}
}

func benchmarkDOMReadYYJSON(b *testing.B, data []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc, err := yyjson.Read(data)
		if err != nil {
			b.Fatal(err)
		}
		uintSink = inspectYYJSONDOM(doc.Root())
		doc.Close()
	}
}

func benchmarkDOMWriteStd(b *testing.B, data []byte) {
	value := decodeStd(b, data)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := json.Marshal(value)
		if err != nil {
			b.Fatal(err)
		}
		bytesSink = out
	}
}

func benchmarkDOMWriteYYJSON(b *testing.B, data []byte) {
	doc := readYYJSON(b, data)
	defer doc.Close()
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := doc.Write()
		if err != nil {
			b.Fatal(err)
		}
		bytesSink = out
	}
	b.StopTimer()
}

func TestBenchmarkLibraryCompatibility(t *testing.T) {
	fixture := fixtures[0]
	want := decodeStd(t, fixture.data)

	for _, tt := range []struct {
		name      string
		unmarshal unmarshalFunc
		marshal   marshalFunc
	}{
		{name: "json-iterator", unmarshal: jsoniter.Unmarshal, marshal: jsoniter.Marshal},
		{name: "segmentio-json", unmarshal: segmentiojson.Unmarshal, marshal: segmentiojson.Marshal},
		{name: "go-json", unmarshal: goccyjson.Unmarshal, marshal: goccyjson.Marshal},
		{name: "sonic", unmarshal: sonicjson.Unmarshal, marshal: sonicjson.Marshal},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var value any
			if err := tt.unmarshal(fixture.data, &value); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			out, err := tt.marshal(value)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			got := decodeStd(t, out)
			if !reflect.DeepEqual(got, want) {
				t.Fatal("round trip changed value")
			}
		})
	}
}

func TestFixturesRoundTrip(t *testing.T) {
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			if got := len(fixture.data); got != fixture.size {
				t.Fatalf("fixture size = %d, want %d", got, fixture.size)
			}

			want := decodeStd(t, fixture.data)
			doc := readYYJSON(t, fixture.data)
			defer doc.Close()

			gotBytes, err := doc.Write()
			if err != nil {
				t.Fatalf("yyjson Write() error = %v", err)
			}
			got := decodeStd(t, gotBytes)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("yyjson round trip changed value")
			}

			if simdjson.SupportedCPU() {
				parsed := readSIMDJSON(t, fixture.data)
				gotBytes, err = marshalSIMDJSON(parsed)
				if err != nil {
					t.Fatalf("simdjson MarshalJSON() error = %v", err)
				}
				got = decodeStd(t, gotBytes)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("simdjson round trip changed value")
				}
			}
		})
	}
}

func makeFixture(name string, size int) benchFixture {
	data := makeSizedJSON(makePayload(maxPayloadItems(size)), size)
	return benchFixture{name: name, size: size, data: data}
}

func makePayload(items int) benchPayload {
	payload := benchPayload{
		ID:     items,
		Name:   fmt.Sprintf("fixture-%d", items),
		Active: true,
		Tags:   []string{"go", "json", "benchmark", "yyjson"},
		Items:  make([]benchItem, 0, items),
		Metrics: map[string]any{
			"count":   items,
			"ratio":   float64(items) / 3.0,
			"enabled": true,
			"source":  "generated",
		},
	}
	for i := 0; i < items; i++ {
		payload.Items = append(payload.Items, benchItem{
			Index:  i,
			Name:   fmt.Sprintf("item-%04d", i),
			Score:  float64(i)*1.25 + 0.5,
			Flags:  []bool{i%2 == 0, i%3 == 0, i%5 == 0},
			Labels: []string{fmt.Sprintf("group-%d", i%7), fmt.Sprintf("shard-%d", i%11)},
		})
	}
	return payload
}

func maxPayloadItems(size int) int {
	lo, hi := 0, 1
	for len(mustJSON(makePayload(hi))) <= size {
		lo = hi
		hi *= 2
		if hi <= 0 {
			panic("benchmark fixture size is too large")
		}
	}
	for lo+1 < hi {
		mid := lo + (hi-lo)/2
		if len(mustJSON(makePayload(mid))) <= size {
			lo = mid
		} else {
			hi = mid
		}
	}
	return lo
}

func makeSizedJSON(payload benchPayload, size int) []byte {
	payload.Padding = ""
	data := mustJSON(payload)
	if len(data) > size {
		panic(fmt.Sprintf("benchmark fixture minimum size %d exceeds target %d", len(data), size))
	}

	payload.Padding = strings.Repeat("x", size-len(data))
	data = mustJSON(payload)
	if len(data) != size {
		panic(fmt.Sprintf("benchmark fixture size %d, want %d", len(data), size))
	}
	return data
}

func mustJSON(payload benchPayload) []byte {
	data, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return data
}

func decodeStd(tb testing.TB, data []byte) any {
	tb.Helper()
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		tb.Fatalf("json.Unmarshal() error = %v", err)
	}
	return out
}

func inspectStdDOM(tb testing.TB, value any) uint64 {
	tb.Helper()
	root, ok := value.(map[string]any)
	if !ok {
		tb.Fatalf("root type = %T, want map[string]any", value)
	}
	items, ok := root["items"].([]any)
	if !ok {
		tb.Fatalf("items type = %T, want []any", root["items"])
	}
	metrics, ok := root["metrics"].(map[string]any)
	if !ok {
		tb.Fatalf("metrics type = %T, want map[string]any", root["metrics"])
	}
	count, ok := metrics["count"].(float64)
	if !ok {
		tb.Fatalf("metrics.count type = %T, want float64", metrics["count"])
	}
	total := uint64(len(items)) + uint64(count)
	if active, ok := root["active"].(bool); ok && active {
		total++
	}
	return total
}

func inspectYYJSONDOM(root *yyjson.Val) uint64 {
	total := root.Get("items").Len() + root.Get("metrics").Get("count").Uint()
	if root.Get("active").Bool() {
		total++
	}
	return total
}

func readYYJSON(tb testing.TB, data []byte) *yyjson.Doc {
	tb.Helper()
	doc, err := yyjson.Read(data)
	if err != nil {
		tb.Fatalf("yyjson.Read() error = %v", err)
	}
	return doc
}

func readSIMDJSON(tb testing.TB, data []byte) *simdjson.ParsedJson {
	tb.Helper()
	skipUnsupportedSIMDJSON(tb)
	parsed, err := simdjson.Parse(data, nil)
	if err != nil {
		tb.Fatalf("simdjson.Parse() error = %v", err)
	}
	return parsed
}

func marshalSIMDJSON(parsed *simdjson.ParsedJson) ([]byte, error) {
	iter := parsed.Iter()
	return iter.MarshalJSON()
}

func skipUnsupportedSIMDJSON(tb testing.TB) {
	tb.Helper()
	if !simdjson.SupportedCPU() {
		tb.Skip("simdjson-go requires AVX2 and CLMUL")
	}
}
