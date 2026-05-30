# yyjson

yyjson for Go, transpiled to pure Go (CGo-free).
Covers parse, inspect, mutate, write, and JSON Pointer/Patch.

The module path is `go.dw1.io/yyjson`, get it with:

```bash
go get go.dw1.io/yyjson
```

## Status

> [!CAUTION]
> **`yyjson`** is pre-v1 and does NOT provide a stable API; **use at your own risk**.

The transpiled yyjson package currently targets:

- `linux/amd64`
- `linux/arm64`

Unsupported platforms fail at build time. See [LIMITATIONS.md](/docs/LIMITATIONS.md)
for the current platform and tooling constraints.

## Reading JSON

```go
package main

import (
	"fmt"

	"go.dw1.io/yyjson"
)

func main() {
	doc, err := yyjson.ReadString(`{"name":"yyjson","items":[1,2]}`)
	if err != nil {
		panic(err)
	}
	defer doc.Close()

	root := doc.Root()
	fmt.Println(root.Get("name").String())
	fmt.Println(root.Get("items").Index(1).Uint())
}
```

Documents own their values. Close a [`Doc`](https://pkg.go.dev/go.dw1.io/yyjson#Doc)
when you are done with it; values derived from a closed document become invalid
and return zero values or `nil`.

Read-only operations on immutable documents and values may run concurrently
while the owning document remains open.

> [!WARNING]
> [`Close`](https://pkg.go.dev/go.dw1.io/yyjson#Doc.Close) must NOT race with
> any other operation.

## Writing JSON

```go
doc, err := yyjson.NewMutDoc()
if err != nil {
	panic(err)
}
defer doc.Close()

root := doc.Object()
root.AddKey("name", doc.String("yyjson"))
root.AddKey("ok", doc.Bool(true))
doc.SetRoot(root)

out, err := doc.WriteString()
if err != nil {
	panic(err)
}
fmt.Println(out)
```

Mutable values belong to the [`MutDoc`](https://pkg.go.dev/go.dw1.io/yyjson#MutDoc)
that created them. Operations reject values from a different mutable document.

> [!WARNING]
> Mutable documents, mutable values, and incremental readers are NOT safe for
> concurrent use.

## JSON Pointer & Patch

```go
doc, err := yyjson.ReadString(`{"items":[{"name":"first"},{"name":"second"}]}`)
if err != nil {
	panic(err)
}
defer doc.Close()

name, err := doc.Pointer("/items/1/name")
if err != nil {
	panic(err)
}
fmt.Println(name.String())
```

[`MutDoc`](https://pkg.go.dev/go.dw1.io/yyjson#MutDoc) also supports JSON Pointer
mutation plus JSON Patch and JSON Merge Patch through
[`PointerSet`](https://pkg.go.dev/go.dw1.io/yyjson#MutDoc.PointerSet),
[`PointerAdd`](https://pkg.go.dev/go.dw1.io/yyjson#MutDoc.PointerAdd),
[`PointerReplace`](https://pkg.go.dev/go.dw1.io/yyjson#MutDoc.PointerReplace),
[`PointerRemove`](https://pkg.go.dev/go.dw1.io/yyjson#MutDoc.PointerRemove),
[`Patch`](https://pkg.go.dev/go.dw1.io/yyjson#MutDoc.Patch),
and [`MergePatch`](https://pkg.go.dev/go.dw1.io/yyjson#MutDoc.MergePatch).

## Read & Write Flags

Read flags are passed as variadic options:

```go
doc, err := yyjson.ReadString(
	`{key:'value', nums:[0x10, Infinity]}`,
	yyjson.ReadJSON5,
)
```

Write flags work the same way:

```go
out, err := doc.WriteString(
	yyjson.WritePrettyTwoSpaces,
	yyjson.WriteEscapeUnicode,
	yyjson.WriteNewlineAtEnd,
)
```

See [Type](https://pkg.go.dev/go.dw1.io/yyjson#Type) and [Subtype](https://pkg.go.dev/go.dw1.io/yyjson#Subtype) for the complete flag list.

## Incremental Reading

[`NewIncrReader`](https://pkg.go.dev/go.dw1.io/yyjson#NewIncrReader) wraps
yyjson's stateful incremental reader. It requires a maximum input size up front,
accepts appended chunks, and closes itself after a completed document or a
non-recoverable parse error.

```go
reader, err := yyjson.NewIncrReader(len(`{"ok":true}`))
if err != nil {
	panic(err)
}
defer reader.Close()

_ = reader.Append([]byte(`{"ok":true`))
_, _ = reader.Read() // returns ReadErrorMore
_ = reader.Append([]byte(`}`))

doc, err := reader.Read()
if err != nil {
	panic(err)
}
defer doc.Close()
```

## Compatibility Helpers

The package aliases common [`encoding/json`](https://pkg.go.dev/encoding/json) types:

- [`RawMessage`](https://pkg.go.dev/encoding/json#RawMessage)
- [`Number`](https://pkg.go.dev/encoding/json#Number)
- [`Marshaler`](https://pkg.go.dev/encoding/json#Marshaler)
- [`Unmarshaler`](https://pkg.go.dev/encoding/json#Unmarshaler)

It also exposes [`Marshal`](https://pkg.go.dev/encoding/json#Marshal),
[`Unmarshal`](https://pkg.go.dev/encoding/json#Unmarshal),
[`Valid`](https://pkg.go.dev/encoding/json#Valid),
[`NewEncoder`](https://pkg.go.dev/encoding/json#NewEncoder), and
[`NewDecoder`](https://pkg.go.dev/encoding/json#NewDecoder). These functions
currently delegate to [`encoding/json`](https://pkg.go.dev/encoding/json).

## Benchmarks

These benchmarks cover the document API: [`Read`](https://pkg.go.dev/go.dw1.io/yyjson#Read),
[`ReadAll`](https://pkg.go.dev/go.dw1.io/yyjson#ReadAll), and
[`Doc.Write`](https://pkg.go.dev/go.dw1.io/yyjson#Doc.Write). The [compatibility helpers](#compatibility-helpers)
are not included.

<details open>
  <summary>vs. std (<code>encoding/json</code>)</summary>

```
goos: linux
goarch: amd64
pkg: benchmarks
cpu: AMD EPYC 9V74 80-Core Processor
                   │     std      │               yyjson                │
                   │    sec/op    │   sec/op     vs base                │
Read/500KiB-4         7.737m ± 1%   3.192m ± 0%  -58.74% (p=0.000 n=10)
Read/1MiB-4          16.347m ± 1%   6.539m ± 0%  -60.00% (p=0.000 n=10)
Read/4MiB-4           69.87m ± 1%   19.45m ± 0%  -72.16% (p=0.000 n=10)
Read/16MiB-4         294.44m ± 1%   67.87m ± 0%  -76.95% (p=0.000 n=10)
ReadAll/500KiB-4      8.180m ± 0%   3.340m ± 0%  -59.16% (p=0.000 n=10)
ReadAll/1MiB-4       17.587m ± 1%   6.868m ± 0%  -60.95% (p=0.000 n=10)
ReadAll/4MiB-4        76.47m ± 1%   21.13m ± 1%  -72.36% (p=0.000 n=10)
ReadAll/16MiB-4      326.34m ± 4%   72.40m ± 3%  -77.81% (p=0.000 n=10)
Write/500KiB-4        7.403m ± 0%   1.835m ± 0%  -75.21% (p=0.000 n=10)
Write/1MiB-4         15.138m ± 1%   3.701m ± 0%  -75.55% (p=0.000 n=10)
Write/4MiB-4          60.33m ± 1%   12.70m ± 0%  -78.95% (p=0.000 n=10)
Write/16MiB-4        238.63m ± 1%   48.78m ± 0%  -79.56% (p=0.000 n=10)
RoundTrip/500KiB-4   16.258m ± 1%   5.029m ± 0%  -69.07% (p=0.000 n=10)
RoundTrip/1MiB-4      33.36m ± 1%   10.20m ± 0%  -69.43% (p=0.000 n=10)
RoundTrip/4MiB-4     133.73m ± 1%   32.23m ± 1%  -75.90% (p=0.000 n=10)
RoundTrip/16MiB-4     534.8m ± 3%   118.2m ± 1%  -77.89% (p=0.000 n=10)
DOMRead/500KiB-4      8.669m ± 1%   3.288m ± 0%  -62.07% (p=0.000 n=10)
DOMRead/1MiB-4       17.986m ± 1%   6.642m ± 0%  -63.07% (p=0.000 n=10)
DOMRead/4MiB-4        72.42m ± 2%   19.53m ± 0%  -73.04% (p=0.000 n=10)
DOMRead/16MiB-4      289.08m ± 2%   67.99m ± 0%  -76.48% (p=0.000 n=10)
DOMWrite/500KiB-4     7.339m ± 0%   1.807m ± 0%  -75.37% (p=0.000 n=10)
DOMWrite/1MiB-4      15.052m ± 1%   3.661m ± 0%  -75.68% (p=0.000 n=10)
DOMWrite/4MiB-4       59.96m ± 1%   12.65m ± 0%  -78.91% (p=0.000 n=10)
DOMWrite/16MiB-4     236.68m ± 1%   48.89m ± 0%  -79.34% (p=0.000 n=10)
geomean               44.14m        12.09m       -72.62%

                   │     std      │                 yyjson                 │
                   │     B/s      │      B/s       vs base                 │
Read/500KiB-4        63.11Mi ± 1%   152.96Mi ± 0%  +142.36% (p=0.000 n=10)
Read/1MiB-4          61.17Mi ± 1%   152.93Mi ± 0%  +150.00% (p=0.000 n=10)
Read/4MiB-4          57.25Mi ± 1%   205.63Mi ± 0%  +259.19% (p=0.000 n=10)
Read/16MiB-4         54.34Mi ± 1%   235.76Mi ± 0%  +333.85% (p=0.000 n=10)
ReadAll/500KiB-4     59.70Mi ± 0%   146.18Mi ± 0%  +144.88% (p=0.000 n=10)
ReadAll/1MiB-4       56.86Mi ± 1%   145.59Mi ± 0%  +156.04% (p=0.000 n=10)
ReadAll/4MiB-4       52.31Mi ± 1%   189.28Mi ± 1%  +261.84% (p=0.000 n=10)
ReadAll/16MiB-4      49.03Mi ± 4%   220.99Mi ± 3%  +350.73% (p=0.000 n=10)
Write/500KiB-4       65.96Mi ± 0%   266.04Mi ± 0%  +303.35% (p=0.000 n=10)
Write/1MiB-4         66.06Mi ± 1%   270.20Mi ± 0%  +309.04% (p=0.000 n=10)
Write/4MiB-4         66.30Mi ± 1%   315.03Mi ± 0%  +375.17% (p=0.000 n=10)
Write/16MiB-4        67.05Mi ± 1%   328.03Mi ± 0%  +389.24% (p=0.000 n=10)
RoundTrip/500KiB-4   30.03Mi ± 1%    97.09Mi ± 0%  +223.29% (p=0.000 n=10)
RoundTrip/1MiB-4     29.98Mi ± 1%    98.06Mi ± 0%  +227.09% (p=0.000 n=10)
RoundTrip/4MiB-4     29.91Mi ± 1%   124.10Mi ± 1%  +314.89% (p=0.000 n=10)
RoundTrip/16MiB-4    29.91Mi ± 3%   135.32Mi ± 1%  +352.40% (p=0.000 n=10)
DOMRead/500KiB-4     56.32Mi ± 1%   148.50Mi ± 0%  +163.65% (p=0.000 n=10)
DOMRead/1MiB-4       55.59Mi ± 1%   150.55Mi ± 0%  +170.80% (p=0.000 n=10)
DOMRead/4MiB-4       55.24Mi ± 2%   204.85Mi ± 0%  +270.82% (p=0.000 n=10)
DOMRead/16MiB-4      55.35Mi ± 2%   235.34Mi ± 0%  +325.18% (p=0.000 n=10)
DOMWrite/500KiB-4    66.54Mi ± 0%   270.16Mi ± 0%  +306.03% (p=0.000 n=10)
DOMWrite/1MiB-4      66.44Mi ± 1%   273.13Mi ± 0%  +311.10% (p=0.000 n=10)
DOMWrite/4MiB-4      66.71Mi ± 1%   316.31Mi ± 0%  +374.13% (p=0.000 n=10)
DOMWrite/16MiB-4     67.60Mi ± 1%   327.24Mi ± 0%  +384.07% (p=0.000 n=10)
geomean              53.57Mi         195.6Mi       +265.21%

                   │        std        │                yyjson                 │
                   │       B/op        │     B/op      vs base                 │
Read/500KiB-4           3534856.0 ± 0%     808.0 ± 0%   -99.98% (p=0.000 n=10)
Read/1MiB-4             7397432.0 ± 0%     808.0 ± 0%   -99.99% (p=0.000 n=10)
Read/4MiB-4            29792596.5 ± 0%     808.0 ± 0%  -100.00% (p=0.000 n=10)
Read/16MiB-4          118214239.5 ± 0%     808.0 ± 0%  -100.00% (p=0.000 n=10)
ReadAll/500KiB-4          4.369Mi ± 0%   1.008Mi ± 0%   -76.92% (p=0.000 n=10)
ReadAll/1MiB-4           11.053Mi ± 0%   2.126Mi ± 0%   -80.77% (p=0.000 n=10)
ReadAll/4MiB-4           44.411Mi ± 0%   9.532Mi ± 0%   -78.54% (p=0.000 n=10)
ReadAll/16MiB-4          176.74Mi ± 0%   34.55Mi ± 0%   -80.45% (p=0.000 n=10)
Write/500KiB-4           2230.7Ki ± 0%   504.5Ki ± 0%   -77.39% (p=0.000 n=10)
Write/1MiB-4              4.443Mi ± 0%   1.000Mi ± 0%   -77.48% (p=0.000 n=10)
Write/4MiB-4             17.550Mi ± 0%   4.000Mi ± 0%   -77.21% (p=0.000 n=10)
Write/16MiB-4             70.32Mi ± 0%   16.00Mi ± 0%   -77.25% (p=0.000 n=10)
RoundTrip/500KiB-4       5745.1Ki ± 0%   505.0Ki ± 0%   -91.21% (p=0.000 n=10)
RoundTrip/1MiB-4         11.710Mi ± 0%   1.001Mi ± 0%   -91.45% (p=0.000 n=10)
RoundTrip/4MiB-4         48.942Mi ± 1%   4.001Mi ± 0%   -91.82% (p=0.000 n=10)
RoundTrip/16MiB-4        200.98Mi ± 3%   16.00Mi ± 0%   -92.04% (p=0.000 n=10)
DOMRead/500KiB-4       3452.007Ki ± 0%   2.398Ki ± 0%   -99.93% (p=0.000 n=10)
DOMRead/1MiB-4         7224.054Ki ± 0%   2.398Ki ± 0%   -99.97% (p=0.000 n=10)
DOMRead/4MiB-4        29094.326Ki ± 0%   2.398Ki ± 0%   -99.99% (p=0.000 n=10)
DOMRead/16MiB-4      115443.588Ki ± 0%   2.398Ki ± 0%  -100.00% (p=0.000 n=10)
DOMWrite/500KiB-4        2230.7Ki ± 0%   504.5Ki ± 0%   -77.39% (p=0.000 n=10)
DOMWrite/1MiB-4           4.443Mi ± 0%   1.000Mi ± 0%   -77.48% (p=0.000 n=10)
DOMWrite/4MiB-4          17.550Mi ± 0%   4.000Mi ± 0%   -77.21% (p=0.000 n=10)
DOMWrite/16MiB-4          70.32Mi ± 2%   16.00Mi ± 0%   -77.25% (p=0.000 n=10)
geomean                   16.64Mi        228.6Ki        -98.66%

                   │       std        │               yyjson                │
                   │    allocs/op     │ allocs/op   vs base                 │
Read/500KiB-4          115223.00 ± 0%   10.00 ± 0%   -99.99% (p=0.000 n=10)
Read/1MiB-4            235106.00 ± 0%   10.00 ± 0%  -100.00% (p=0.000 n=10)
Read/4MiB-4            921176.00 ± 0%   10.00 ± 0%  -100.00% (p=0.000 n=10)
Read/16MiB-4          3627302.00 ± 0%   10.00 ± 0%  -100.00% (p=0.000 n=10)
ReadAll/500KiB-4       115237.00 ± 0%   33.00 ± 0%   -99.97% (p=0.000 n=10)
ReadAll/1MiB-4         235122.00 ± 0%   35.00 ± 0%   -99.99% (p=0.000 n=10)
ReadAll/4MiB-4         921194.00 ± 0%   39.00 ± 0%  -100.00% (p=0.000 n=10)
ReadAll/16MiB-4       3627322.00 ± 0%   42.00 ± 0%  -100.00% (p=0.000 n=10)
Write/500KiB-4         52814.000 ± 0%   7.000 ± 0%   -99.99% (p=0.000 n=10)
Write/1MiB-4          107759.000 ± 0%   7.000 ± 0%   -99.99% (p=0.000 n=10)
Write/4MiB-4          422205.000 ± 0%   7.000 ± 0%  -100.00% (p=0.000 n=10)
Write/16MiB-4        1662512.000 ± 0%   7.000 ± 0%  -100.00% (p=0.000 n=10)
RoundTrip/500KiB-4     168038.00 ± 0%   15.00 ± 0%   -99.99% (p=0.000 n=10)
RoundTrip/1MiB-4       342868.00 ± 0%   15.00 ± 0%  -100.00% (p=0.000 n=10)
RoundTrip/4MiB-4      1343393.50 ± 0%   15.00 ± 0%  -100.00% (p=0.000 n=10)
RoundTrip/16MiB-4     5289837.00 ± 0%   15.00 ± 0%  -100.00% (p=0.000 n=10)
DOMRead/500KiB-4       115223.00 ± 0%   29.00 ± 0%   -99.97% (p=0.000 n=10)
DOMRead/1MiB-4         235106.00 ± 0%   29.00 ± 0%   -99.99% (p=0.000 n=10)
DOMRead/4MiB-4         921176.00 ± 0%   29.00 ± 0%  -100.00% (p=0.000 n=10)
DOMRead/16MiB-4       3627302.00 ± 0%   29.00 ± 0%  -100.00% (p=0.000 n=10)
DOMWrite/500KiB-4      52814.000 ± 0%   7.000 ± 0%   -99.99% (p=0.000 n=10)
DOMWrite/1MiB-4       107759.000 ± 0%   7.000 ± 0%   -99.99% (p=0.000 n=10)
DOMWrite/4MiB-4       422205.000 ± 0%   7.000 ± 0%  -100.00% (p=0.000 n=10)
DOMWrite/16MiB-4     1662512.000 ± 0%   7.000 ± 0%  -100.00% (p=0.000 n=10)
geomean                   450.4k        14.11       -100.00%
```
</details>

<details>
  <summary>vs. json-iterator</summary>

```
goos: linux
goarch: amd64
pkg: benchmarks
cpu: AMD EPYC 9V74 80-Core Processor
                   │ json-iterator │               yyjson                │
                   │    sec/op     │   sec/op     vs base                │
Read/500KiB-4          7.015m ± 0%   3.192m ± 0%  -54.49% (p=0.000 n=10)
Read/1MiB-4           14.941m ± 0%   6.539m ± 0%  -56.24% (p=0.000 n=10)
Read/4MiB-4            66.71m ± 1%   19.45m ± 0%  -70.84% (p=0.000 n=10)
Read/16MiB-4          283.64m ± 2%   67.87m ± 0%  -76.07% (p=0.000 n=10)
ReadAll/500KiB-4       7.308m ± 0%   3.340m ± 0%  -54.29% (p=0.000 n=10)
ReadAll/1MiB-4        15.532m ± 1%   6.868m ± 0%  -55.78% (p=0.000 n=10)
ReadAll/4MiB-4         69.10m ± 2%   21.13m ± 1%  -69.42% (p=0.000 n=10)
ReadAll/16MiB-4       294.14m ± 2%   72.40m ± 3%  -75.38% (p=0.000 n=10)
Write/500KiB-4         4.178m ± 2%   1.835m ± 0%  -56.07% (p=0.000 n=10)
Write/1MiB-4           8.577m ± 2%   3.701m ± 0%  -56.85% (p=0.000 n=10)
Write/4MiB-4           35.63m ± 2%   12.70m ± 0%  -64.36% (p=0.000 n=10)
Write/16MiB-4         141.71m ± 2%   48.78m ± 0%  -65.58% (p=0.000 n=10)
RoundTrip/500KiB-4    12.986m ± 1%   5.029m ± 0%  -61.27% (p=0.000 n=10)
RoundTrip/1MiB-4       27.12m ± 2%   10.20m ± 0%  -62.40% (p=0.000 n=10)
RoundTrip/4MiB-4      110.13m ± 1%   32.23m ± 1%  -70.73% (p=0.000 n=10)
RoundTrip/16MiB-4      463.3m ± 2%   118.2m ± 1%  -74.48% (p=0.000 n=10)
DOMRead/500KiB-4                     3.288m ± 0%
DOMRead/1MiB-4                       6.642m ± 0%
DOMRead/4MiB-4                       19.53m ± 0%
DOMRead/16MiB-4                      67.99m ± 0%
DOMWrite/500KiB-4                    1.807m ± 0%
DOMWrite/1MiB-4                      3.661m ± 0%
DOMWrite/4MiB-4                      12.65m ± 0%
DOMWrite/16MiB-4                     48.89m ± 0%
geomean                37.44m        12.09m       -64.88%

                   │ json-iterator │                 yyjson                 │
                   │      B/s      │      B/s       vs base                 │
Read/500KiB-4         69.61Mi ± 0%   152.96Mi ± 0%  +119.74% (p=0.000 n=10)
Read/1MiB-4           66.93Mi ± 0%   152.93Mi ± 0%  +128.50% (p=0.000 n=10)
Read/4MiB-4           59.96Mi ± 1%   205.63Mi ± 0%  +242.93% (p=0.000 n=10)
Read/16MiB-4          56.41Mi ± 2%   235.76Mi ± 0%  +317.94% (p=0.000 n=10)
ReadAll/500KiB-4      66.81Mi ± 0%   146.18Mi ± 0%  +118.80% (p=0.000 n=10)
ReadAll/1MiB-4        64.38Mi ± 1%   145.59Mi ± 0%  +126.14% (p=0.000 n=10)
ReadAll/4MiB-4        57.89Mi ± 2%   189.28Mi ± 1%  +226.94% (p=0.000 n=10)
ReadAll/16MiB-4       54.40Mi ± 2%   220.99Mi ± 3%  +306.21% (p=0.000 n=10)
Write/500KiB-4        116.9Mi ± 1%    266.0Mi ± 0%  +127.64% (p=0.000 n=10)
Write/1MiB-4          116.6Mi ± 2%    270.2Mi ± 0%  +131.75% (p=0.000 n=10)
Write/4MiB-4          112.3Mi ± 2%    315.0Mi ± 0%  +180.60% (p=0.000 n=10)
Write/16MiB-4         112.9Mi ± 2%    328.0Mi ± 0%  +190.52% (p=0.000 n=10)
RoundTrip/500KiB-4    37.60Mi ± 1%    97.09Mi ± 0%  +158.19% (p=0.000 n=10)
RoundTrip/1MiB-4      36.87Mi ± 2%    98.06Mi ± 0%  +165.96% (p=0.000 n=10)
RoundTrip/4MiB-4      36.32Mi ± 1%   124.10Mi ± 1%  +241.68% (p=0.000 n=10)
RoundTrip/16MiB-4     34.54Mi ± 2%   135.32Mi ± 1%  +291.81% (p=0.000 n=10)
DOMRead/500KiB-4                      148.5Mi ± 0%
DOMRead/1MiB-4                        150.6Mi ± 0%
DOMRead/4MiB-4                        204.8Mi ± 0%
DOMRead/16MiB-4                       235.3Mi ± 0%
DOMWrite/500KiB-4                     270.2Mi ± 0%
DOMWrite/1MiB-4                       273.1Mi ± 0%
DOMWrite/4MiB-4                       316.3Mi ± 0%
DOMWrite/16MiB-4                      327.2Mi ± 0%
geomean               63.16Mi         195.6Mi       +184.76%

                   │  json-iterator   │                yyjson                 │
                   │       B/op       │     B/op      vs base                 │
Read/500KiB-4          4311046.0 ± 0%     808.0 ± 0%   -99.98% (p=0.000 n=10)
Read/1MiB-4            8972893.0 ± 0%     808.0 ± 0%   -99.99% (p=0.000 n=10)
Read/4MiB-4           35942018.0 ± 0%     808.0 ± 0%  -100.00% (p=0.000 n=10)
Read/16MiB-4         142404500.0 ± 0%     808.0 ± 0%  -100.00% (p=0.000 n=10)
ReadAll/500KiB-4         4.113Mi ± 0%   1.008Mi ± 0%   -75.48% (p=0.000 n=10)
ReadAll/1MiB-4           8.561Mi ± 0%   2.126Mi ± 0%   -75.17% (p=0.000 n=10)
ReadAll/4MiB-4          34.291Mi ± 0%   9.532Mi ± 0%   -72.20% (p=0.000 n=10)
ReadAll/16MiB-4         135.86Mi ± 0%   34.55Mi ± 0%   -74.57% (p=0.000 n=10)
Write/500KiB-4          1555.9Ki ± 0%   504.5Ki ± 0%   -67.58% (p=0.000 n=10)
Write/1MiB-4             3.100Mi ± 0%   1.000Mi ± 0%   -67.72% (p=0.000 n=10)
Write/4MiB-4            12.319Mi ± 1%   4.000Mi ± 0%   -67.53% (p=0.000 n=10)
Write/16MiB-4            50.68Mi ± 5%   16.00Mi ± 0%   -68.43% (p=0.000 n=10)
RoundTrip/500KiB-4      5964.5Ki ± 0%   505.0Ki ± 0%   -91.53% (p=0.000 n=10)
RoundTrip/1MiB-4        12.438Mi ± 1%   1.001Mi ± 0%   -91.95% (p=0.000 n=10)
RoundTrip/4MiB-4        56.607Mi ± 3%   4.001Mi ± 0%   -92.93% (p=0.000 n=10)
RoundTrip/16MiB-4       254.68Mi ± 5%   16.00Mi ± 0%   -93.72% (p=0.000 n=10)
DOMRead/500KiB-4                        2.398Ki ± 0%
DOMRead/1MiB-4                          2.398Ki ± 0%
DOMRead/4MiB-4                          2.398Ki ± 0%
DOMRead/16MiB-4                         2.398Ki ± 0%
DOMWrite/500KiB-4                       504.5Ki ± 0%
DOMWrite/1MiB-4                         1.000Mi ± 0%
DOMWrite/4MiB-4                         4.000Mi ± 0%
DOMWrite/16MiB-4                        16.00Mi ± 0%
geomean                  17.57Mi        228.6Ki        -97.80%

                   │  json-iterator  │               yyjson                │
                   │    allocs/op    │ allocs/op   vs base                 │
Read/500KiB-4         158436.00 ± 0%   10.00 ± 0%   -99.99% (p=0.000 n=10)
Read/1MiB-4           323274.00 ± 0%   10.00 ± 0%  -100.00% (p=0.000 n=10)
Read/4MiB-4          1266619.00 ± 0%   10.00 ± 0%  -100.00% (p=0.000 n=10)
Read/16MiB-4         4987540.00 ± 0%   10.00 ± 0%  -100.00% (p=0.000 n=10)
ReadAll/500KiB-4      158526.00 ± 0%   33.00 ± 0%   -99.98% (p=0.000 n=10)
ReadAll/1MiB-4        323465.00 ± 0%   35.00 ± 0%   -99.99% (p=0.000 n=10)
ReadAll/4MiB-4       1267487.00 ± 0%   39.00 ± 0%  -100.00% (p=0.000 n=10)
ReadAll/16MiB-4      4991275.00 ± 0%   42.00 ± 0%  -100.00% (p=0.000 n=10)
Write/500KiB-4        19205.000 ± 0%   7.000 ± 0%   -99.96% (p=0.000 n=10)
Write/1MiB-4          39185.000 ± 0%   7.000 ± 0%   -99.98% (p=0.000 n=10)
Write/4MiB-4         153529.000 ± 0%   7.000 ± 0%  -100.00% (p=0.000 n=10)
Write/16MiB-4        604550.000 ± 0%   7.000 ± 0%  -100.00% (p=0.000 n=10)
RoundTrip/500KiB-4    177643.00 ± 0%   15.00 ± 0%   -99.99% (p=0.000 n=10)
RoundTrip/1MiB-4      362464.00 ± 0%   15.00 ± 0%  -100.00% (p=0.000 n=10)
RoundTrip/4MiB-4     1420166.50 ± 0%   15.00 ± 0%  -100.00% (p=0.000 n=10)
RoundTrip/16MiB-4    5592123.50 ± 0%   15.00 ± 0%  -100.00% (p=0.000 n=10)
DOMRead/500KiB-4                       29.00 ± 0%
DOMRead/1MiB-4                         29.00 ± 0%
DOMRead/4MiB-4                         29.00 ± 0%
DOMRead/16MiB-4                        29.00 ± 0%
DOMWrite/500KiB-4                      7.000 ± 0%
DOMWrite/1MiB-4                        7.000 ± 0%
DOMWrite/4MiB-4                        7.000 ± 0%
DOMWrite/16MiB-4                       7.000 ± 0%
geomean                  458.0k        14.11       -100.00%
```
</details>

<details>
  <summary>vs. segmentio/encoding/json</summary>

```
goos: linux
goarch: amd64
pkg: benchmarks
cpu: AMD EPYC 9V74 80-Core Processor
                   │ segmentio-json │               yyjson                │
                   │     sec/op     │   sec/op     vs base                │
Read/500KiB-4           7.042m ± 0%   3.192m ± 0%  -54.67% (p=0.000 n=10)
Read/1MiB-4            14.802m ± 0%   6.539m ± 0%  -55.82% (p=0.000 n=10)
Read/4MiB-4             63.03m ± 1%   19.45m ± 0%  -69.14% (p=0.000 n=10)
Read/16MiB-4           265.82m ± 1%   67.87m ± 0%  -74.47% (p=0.000 n=10)
ReadAll/500KiB-4        8.533m ± 0%   3.340m ± 0%  -60.85% (p=0.000 n=10)
ReadAll/1MiB-4         18.032m ± 1%   6.868m ± 0%  -61.91% (p=0.000 n=10)
ReadAll/4MiB-4          76.72m ± 1%   21.13m ± 1%  -72.45% (p=0.000 n=10)
ReadAll/16MiB-4        319.62m ± 1%   72.40m ± 3%  -77.35% (p=0.000 n=10)
Write/500KiB-4          3.002m ± 1%   1.835m ± 0%  -38.85% (p=0.000 n=10)
Write/1MiB-4            6.093m ± 1%   3.701m ± 0%  -39.26% (p=0.000 n=10)
Write/4MiB-4            24.78m ± 1%   12.70m ± 0%  -48.76% (p=0.000 n=10)
Write/16MiB-4          100.20m ± 1%   48.78m ± 0%  -51.32% (p=0.000 n=10)
RoundTrip/500KiB-4     11.255m ± 1%   5.029m ± 0%  -55.31% (p=0.000 n=10)
RoundTrip/1MiB-4        23.23m ± 1%   10.20m ± 0%  -56.10% (p=0.000 n=10)
RoundTrip/4MiB-4        93.70m ± 2%   32.23m ± 1%  -65.60% (p=0.000 n=10)
RoundTrip/16MiB-4       389.0m ± 1%   118.2m ± 1%  -69.60% (p=0.000 n=10)
DOMRead/500KiB-4                      3.288m ± 0%
DOMRead/1MiB-4                        6.642m ± 0%
DOMRead/4MiB-4                        19.53m ± 0%
DOMRead/16MiB-4                       67.99m ± 0%
DOMWrite/500KiB-4                     1.807m ± 0%
DOMWrite/1MiB-4                       3.661m ± 0%
DOMWrite/4MiB-4                       12.65m ± 0%
DOMWrite/16MiB-4                      48.89m ± 0%
geomean                 33.76m        12.09m       -61.06%

                   │ segmentio-json │                 yyjson                 │
                   │      B/s       │      B/s       vs base                 │
Read/500KiB-4          69.33Mi ± 0%   152.96Mi ± 0%  +120.61% (p=0.000 n=10)
Read/1MiB-4            67.56Mi ± 0%   152.93Mi ± 0%  +126.37% (p=0.000 n=10)
Read/4MiB-4            63.46Mi ± 1%   205.63Mi ± 0%  +224.05% (p=0.000 n=10)
Read/16MiB-4           60.19Mi ± 1%   235.76Mi ± 0%  +291.68% (p=0.000 n=10)
ReadAll/500KiB-4       57.23Mi ± 0%   146.18Mi ± 0%  +155.45% (p=0.000 n=10)
ReadAll/1MiB-4         55.46Mi ± 1%   145.59Mi ± 0%  +162.54% (p=0.000 n=10)
ReadAll/4MiB-4         52.14Mi ± 1%   189.28Mi ± 1%  +263.00% (p=0.000 n=10)
ReadAll/16MiB-4        50.06Mi ± 1%   220.99Mi ± 3%  +341.46% (p=0.000 n=10)
Write/500KiB-4         162.7Mi ± 1%    266.0Mi ± 0%   +63.54% (p=0.000 n=10)
Write/1MiB-4           164.1Mi ± 1%    270.2Mi ± 0%   +64.63% (p=0.000 n=10)
Write/4MiB-4           161.4Mi ± 1%    315.0Mi ± 0%   +95.16% (p=0.000 n=10)
Write/16MiB-4          159.7Mi ± 1%    328.0Mi ± 0%  +105.44% (p=0.000 n=10)
RoundTrip/500KiB-4     43.39Mi ± 1%    97.09Mi ± 0%  +123.77% (p=0.000 n=10)
RoundTrip/1MiB-4       43.04Mi ± 1%    98.06Mi ± 0%  +127.81% (p=0.000 n=10)
RoundTrip/4MiB-4       42.69Mi ± 2%   124.10Mi ± 1%  +190.70% (p=0.000 n=10)
RoundTrip/16MiB-4      41.13Mi ± 1%   135.32Mi ± 1%  +228.99% (p=0.000 n=10)
DOMRead/500KiB-4                       148.5Mi ± 0%
DOMRead/1MiB-4                         150.6Mi ± 0%
DOMRead/4MiB-4                         204.8Mi ± 0%
DOMRead/16MiB-4                        235.3Mi ± 0%
DOMWrite/500KiB-4                      270.2Mi ± 0%
DOMWrite/1MiB-4                        273.1Mi ± 0%
DOMWrite/4MiB-4                        316.3Mi ± 0%
DOMWrite/16MiB-4                       327.2Mi ± 0%
geomean                70.03Mi         195.6Mi       +156.84%

                   │  segmentio-json   │                yyjson                 │
                   │       B/op        │     B/op      vs base                 │
Read/500KiB-4          4160129.0 ±  0%     808.0 ± 0%   -99.98% (p=0.000 n=10)
Read/1MiB-4            8479841.0 ±  0%     808.0 ± 0%   -99.99% (p=0.000 n=10)
Read/4MiB-4           33246473.0 ±  0%     808.0 ± 0%  -100.00% (p=0.000 n=10)
Read/16MiB-4         130990766.0 ±  0%     808.0 ± 0%  -100.00% (p=0.000 n=10)
ReadAll/500KiB-4         4.937Mi ±  0%   1.008Mi ± 0%   -79.57% (p=0.000 n=10)
ReadAll/1MiB-4          10.057Mi ±  0%   2.126Mi ± 0%   -78.86% (p=0.000 n=10)
ReadAll/4MiB-4          39.677Mi ±  0%   9.532Mi ± 0%   -75.98% (p=0.000 n=10)
ReadAll/16MiB-4         156.89Mi ±  0%   34.55Mi ± 0%   -77.98% (p=0.000 n=10)
Write/500KiB-4           505.2Ki ±  0%   504.5Ki ± 0%         ~ (p=0.428 n=10)
Write/1MiB-4             1.005Mi ±  0%   1.000Mi ± 0%         ~ (p=0.459 n=10)
Write/4MiB-4             4.041Mi ±  1%   4.000Mi ± 0%         ~ (p=1.000 n=10)
Write/16MiB-4            17.69Mi ± 10%   16.00Mi ± 0%    -9.54% (p=0.021 n=10)
RoundTrip/500KiB-4      4663.6Ki ±  0%   505.0Ki ± 0%   -89.17% (p=0.000 n=10)
RoundTrip/1MiB-4         9.563Mi ±  1%   1.001Mi ± 0%   -89.53% (p=0.000 n=10)
RoundTrip/4MiB-4        42.503Mi ±  5%   4.001Mi ± 0%   -90.59% (p=0.000 n=10)
RoundTrip/16MiB-4       205.01Mi ±  5%   16.00Mi ± 0%   -92.19% (p=0.000 n=10)
DOMRead/500KiB-4                         2.398Ki ± 0%
DOMRead/1MiB-4                           2.398Ki ± 0%
DOMRead/4MiB-4                           2.398Ki ± 0%
DOMRead/16MiB-4                          2.398Ki ± 0%
DOMWrite/500KiB-4                        504.5Ki ± 0%
DOMWrite/1MiB-4                          1.000Mi ± 0%
DOMWrite/4MiB-4                          4.000Mi ± 0%
DOMWrite/16MiB-4                         16.00Mi ± 0%
geomean                  12.81Mi         228.6Ki        -96.98%

                   │  segmentio-json  │               yyjson                │
                   │    allocs/op     │ allocs/op   vs base                 │
Read/500KiB-4          91226.00 ±  0%   10.00 ± 0%   -99.99% (p=0.000 n=10)
Read/1MiB-4           186132.00 ±  0%   10.00 ± 0%   -99.99% (p=0.000 n=10)
Read/4MiB-4           729268.00 ±  0%   10.00 ± 0%  -100.00% (p=0.000 n=10)
Read/16MiB-4         2871615.00 ±  0%   10.00 ± 0%  -100.00% (p=0.000 n=10)
ReadAll/500KiB-4       91245.00 ±  0%   33.00 ± 0%   -99.96% (p=0.000 n=10)
ReadAll/1MiB-4        186156.00 ±  0%   35.00 ± 0%   -99.98% (p=0.000 n=10)
ReadAll/4MiB-4        729301.00 ±  0%   39.00 ± 0%   -99.99% (p=0.000 n=10)
ReadAll/16MiB-4      2871658.00 ±  0%   42.00 ± 0%  -100.00% (p=0.000 n=10)
Write/500KiB-4            1.000 ±  0%   7.000 ± 0%  +600.00% (p=0.000 n=10)
Write/1MiB-4              1.000 ±  0%   7.000 ± 0%  +600.00% (p=0.000 n=10)
Write/4MiB-4              1.000 ±  0%   7.000 ± 0%  +600.00% (p=0.000 n=10)
Write/16MiB-4             2.000 ± 50%   7.000 ± 0%  +250.00% (p=0.000 n=10)
RoundTrip/500KiB-4     91228.00 ±  0%   15.00 ± 0%   -99.98% (p=0.000 n=10)
RoundTrip/1MiB-4      186136.00 ±  0%   15.00 ± 0%   -99.99% (p=0.000 n=10)
RoundTrip/4MiB-4      729282.00 ±  0%   15.00 ± 0%  -100.00% (p=0.000 n=10)
RoundTrip/16MiB-4    2871646.50 ±  0%   15.00 ± 0%  -100.00% (p=0.000 n=10)
DOMRead/500KiB-4                        29.00 ± 0%
DOMRead/1MiB-4                          29.00 ± 0%
DOMRead/4MiB-4                          29.00 ± 0%
DOMRead/16MiB-4                         29.00 ± 0%
DOMWrite/500KiB-4                       7.000 ± 0%
DOMWrite/1MiB-4                         7.000 ± 0%
DOMWrite/4MiB-4                         7.000 ± 0%
DOMWrite/16MiB-4                        7.000 ± 0%
geomean                  17.67k         14.11        -99.92%
```
</details>

<details>
  <summary>vs. go-json</summary>

```
goos: linux
goarch: amd64
pkg: benchmarks
cpu: AMD EPYC 9V74 80-Core Processor
                   │   go-json    │               yyjson                │
                   │    sec/op    │   sec/op     vs base                │
Read/500KiB-4         7.234m ± 1%   3.192m ± 0%  -55.87% (p=0.000 n=10)
Read/1MiB-4          15.856m ± 1%   6.539m ± 0%  -58.76% (p=0.000 n=10)
Read/4MiB-4           69.72m ± 1%   19.45m ± 0%  -72.10% (p=0.000 n=10)
Read/16MiB-4         290.93m ± 2%   67.87m ± 0%  -76.67% (p=0.000 n=10)
ReadAll/500KiB-4      8.255m ± 1%   3.340m ± 0%  -59.54% (p=0.000 n=10)
ReadAll/1MiB-4       18.088m ± 0%   6.868m ± 0%  -62.03% (p=0.000 n=10)
ReadAll/4MiB-4        77.74m ± 1%   21.13m ± 1%  -72.82% (p=0.000 n=10)
ReadAll/16MiB-4      314.38m ± 2%   72.40m ± 3%  -76.97% (p=0.000 n=10)
Write/500KiB-4        5.024m ± 4%   1.835m ± 0%  -63.46% (p=0.000 n=10)
Write/1MiB-4         10.290m ± 4%   3.701m ± 0%  -64.03% (p=0.000 n=10)
Write/4MiB-4          41.42m ± 1%   12.70m ± 0%  -69.34% (p=0.000 n=10)
Write/16MiB-4        172.03m ± 3%   48.78m ± 0%  -71.65% (p=0.000 n=10)
RoundTrip/500KiB-4   13.569m ± 1%   5.029m ± 0%  -62.94% (p=0.000 n=10)
RoundTrip/1MiB-4      28.65m ± 2%   10.20m ± 0%  -64.41% (p=0.000 n=10)
RoundTrip/4MiB-4     122.06m ± 2%   32.23m ± 1%  -73.59% (p=0.000 n=10)
RoundTrip/16MiB-4     553.2m ± 8%   118.2m ± 1%  -78.62% (p=0.000 n=10)
DOMRead/500KiB-4                    3.288m ± 0%
DOMRead/1MiB-4                      6.642m ± 0%
DOMRead/4MiB-4                      19.53m ± 0%
DOMRead/16MiB-4                     67.99m ± 0%
DOMWrite/500KiB-4                   1.807m ± 0%
DOMWrite/1MiB-4                     3.661m ± 0%
DOMWrite/4MiB-4                     12.65m ± 0%
DOMWrite/16MiB-4                    48.89m ± 0%
geomean               41.65m        12.09m       -68.44%

                   │   go-json    │                 yyjson                 │
                   │     B/s      │      B/s       vs base                 │
Read/500KiB-4        67.50Mi ± 1%   152.96Mi ± 0%  +126.60% (p=0.000 n=10)
Read/1MiB-4          63.07Mi ± 1%   152.93Mi ± 0%  +142.47% (p=0.000 n=10)
Read/4MiB-4          57.38Mi ± 1%   205.63Mi ± 0%  +258.38% (p=0.000 n=10)
Read/16MiB-4         55.00Mi ± 2%   235.76Mi ± 0%  +328.66% (p=0.000 n=10)
ReadAll/500KiB-4     59.15Mi ± 1%   146.18Mi ± 0%  +147.15% (p=0.000 n=10)
ReadAll/1MiB-4       55.28Mi ± 0%   145.59Mi ± 0%  +163.35% (p=0.000 n=10)
ReadAll/4MiB-4       51.45Mi ± 1%   189.28Mi ± 1%  +267.88% (p=0.000 n=10)
ReadAll/16MiB-4      50.89Mi ± 2%   220.99Mi ± 3%  +334.22% (p=0.000 n=10)
Write/500KiB-4       97.20Mi ± 4%   266.04Mi ± 0%  +173.70% (p=0.000 n=10)
Write/1MiB-4         97.18Mi ± 4%   270.20Mi ± 0%  +178.03% (p=0.000 n=10)
Write/4MiB-4         96.58Mi ± 1%   315.03Mi ± 0%  +226.19% (p=0.000 n=10)
Write/16MiB-4        93.01Mi ± 3%   328.03Mi ± 0%  +252.69% (p=0.000 n=10)
RoundTrip/500KiB-4   35.98Mi ± 1%    97.09Mi ± 0%  +169.83% (p=0.000 n=10)
RoundTrip/1MiB-4     34.90Mi ± 2%    98.06Mi ± 0%  +180.93% (p=0.000 n=10)
RoundTrip/4MiB-4     32.77Mi ± 2%   124.10Mi ± 1%  +278.67% (p=0.000 n=10)
RoundTrip/16MiB-4    28.93Mi ± 8%   135.32Mi ± 1%  +367.68% (p=0.000 n=10)
DOMRead/500KiB-4                     148.5Mi ± 0%
DOMRead/1MiB-4                       150.6Mi ± 0%
DOMRead/4MiB-4                       204.8Mi ± 0%
DOMRead/16MiB-4                      235.3Mi ± 0%
DOMWrite/500KiB-4                    270.2Mi ± 0%
DOMWrite/1MiB-4                      273.1Mi ± 0%
DOMWrite/4MiB-4                      316.3Mi ± 0%
DOMWrite/16MiB-4                     327.2Mi ± 0%
geomean              56.77Mi         195.6Mi       +216.84%

                   │      go-json      │                yyjson                 │
                   │       B/op        │     B/op      vs base                 │
Read/500KiB-4          4626236.5 ±  0%     808.0 ± 0%   -99.98% (p=0.000 n=10)
Read/1MiB-4            9472684.0 ±  0%     808.0 ± 0%   -99.99% (p=0.000 n=10)
Read/4MiB-4           37423575.0 ±  0%     808.0 ± 0%  -100.00% (p=0.000 n=10)
Read/16MiB-4         147954270.5 ±  1%     808.0 ± 0%  -100.00% (p=0.000 n=10)
ReadAll/500KiB-4         4.566Mi ±  0%   1.008Mi ± 0%   -77.91% (p=0.000 n=10)
ReadAll/1MiB-4          10.337Mi ±  0%   2.126Mi ± 0%   -79.44% (p=0.000 n=10)
ReadAll/4MiB-4          40.833Mi ±  0%   9.532Mi ± 0%   -76.66% (p=0.000 n=10)
ReadAll/16MiB-4         161.41Mi ±  0%   34.55Mi ± 0%   -78.60% (p=0.000 n=10)
Write/500KiB-4           959.7Ki ±  0%   504.5Ki ± 0%   -47.43% (p=0.000 n=10)
Write/1MiB-4             1.920Mi ±  1%   1.000Mi ± 0%   -47.89% (p=0.000 n=10)
Write/4MiB-4             7.845Mi ±  1%   4.000Mi ± 0%   -49.01% (p=0.000 n=10)
Write/16MiB-4            36.46Mi ±  2%   16.00Mi ± 0%   -56.12% (p=0.000 n=10)
RoundTrip/500KiB-4      5653.7Ki ±  1%   505.0Ki ± 0%   -91.07% (p=0.000 n=10)
RoundTrip/1MiB-4        11.864Mi ±  2%   1.001Mi ± 0%   -91.56% (p=0.000 n=10)
RoundTrip/4MiB-4        53.870Mi ±  7%   4.001Mi ± 0%   -92.57% (p=0.000 n=10)
RoundTrip/16MiB-4       283.22Mi ± 15%   16.00Mi ± 0%   -94.35% (p=0.000 n=10)
DOMRead/500KiB-4                         2.398Ki ± 0%
DOMRead/1MiB-4                           2.398Ki ± 0%
DOMRead/4MiB-4                           2.398Ki ± 0%
DOMRead/16MiB-4                          2.398Ki ± 0%
DOMWrite/500KiB-4                        504.5Ki ± 0%
DOMWrite/1MiB-4                          1.000Mi ± 0%
DOMWrite/4MiB-4                          4.000Mi ± 0%
DOMWrite/16MiB-4                         16.00Mi ± 0%
geomean                  16.56Mi         228.6Ki        -97.67%

                   │     go-json     │               yyjson                │
                   │    allocs/op    │ allocs/op   vs base                 │
Read/500KiB-4         139221.00 ± 0%   10.00 ± 0%   -99.99% (p=0.000 n=10)
Read/1MiB-4           284077.50 ± 0%   10.00 ± 0%  -100.00% (p=0.000 n=10)
Read/4MiB-4          1113076.00 ± 0%   10.00 ± 0%  -100.00% (p=0.000 n=10)
Read/16MiB-4         4382973.50 ± 0%   10.00 ± 0%  -100.00% (p=0.000 n=10)
ReadAll/500KiB-4      153637.00 ± 0%   33.00 ± 0%   -99.98% (p=0.000 n=10)
ReadAll/1MiB-4        313482.00 ± 0%   35.00 ± 0%   -99.99% (p=0.000 n=10)
ReadAll/4MiB-4       1228239.00 ± 0%   39.00 ± 0%  -100.00% (p=0.000 n=10)
ReadAll/16MiB-4      4836402.00 ± 0%   42.00 ± 0%  -100.00% (p=0.000 n=10)
Write/500KiB-4         4803.000 ± 0%   7.000 ± 0%   -99.85% (p=0.000 n=10)
Write/1MiB-4           9798.000 ± 0%   7.000 ± 0%   -99.93% (p=0.000 n=10)
Write/4MiB-4          38385.000 ± 0%   7.000 ± 0%   -99.98% (p=0.000 n=10)
Write/16MiB-4        151143.000 ± 0%   7.000 ± 0%  -100.00% (p=0.000 n=10)
RoundTrip/500KiB-4    144025.00 ± 0%   15.00 ± 0%   -99.99% (p=0.000 n=10)
RoundTrip/1MiB-4      293880.00 ± 0%   15.00 ± 0%   -99.99% (p=0.000 n=10)
RoundTrip/4MiB-4     1151480.50 ± 0%   15.00 ± 0%  -100.00% (p=0.000 n=10)
RoundTrip/16MiB-4    4534174.00 ± 0%   15.00 ± 0%  -100.00% (p=0.000 n=10)
DOMRead/500KiB-4                       29.00 ± 0%
DOMRead/1MiB-4                         29.00 ± 0%
DOMRead/4MiB-4                         29.00 ± 0%
DOMRead/16MiB-4                        29.00 ± 0%
DOMWrite/500KiB-4                      7.000 ± 0%
DOMWrite/1MiB-4                        7.000 ± 0%
DOMWrite/4MiB-4                        7.000 ± 0%
DOMWrite/16MiB-4                       7.000 ± 0%
geomean                  295.2k        14.11       -100.00%
```
</details>

<details>
  <summary>vs. sonic</summary>

```
goos: linux
goarch: amd64
pkg: benchmarks
cpu: AMD EPYC 9V74 80-Core Processor
                   │    sonic     │               yyjson                │
                   │    sec/op    │   sec/op     vs base                │
Read/500KiB-4         3.937m ± 1%   3.192m ± 0%  -18.92% (p=0.000 n=10)
Read/1MiB-4           8.735m ± 1%   6.539m ± 0%  -25.14% (p=0.000 n=10)
Read/4MiB-4           40.13m ± 1%   19.45m ± 0%  -51.53% (p=0.000 n=10)
Read/16MiB-4         167.43m ± 1%   67.87m ± 0%  -59.47% (p=0.000 n=10)
ReadAll/500KiB-4      4.030m ± 1%   3.340m ± 0%  -17.11% (p=0.000 n=10)
ReadAll/1MiB-4       10.086m ± 1%   6.868m ± 0%  -31.90% (p=0.000 n=10)
ReadAll/4MiB-4        46.26m ± 1%   21.13m ± 1%  -54.31% (p=0.000 n=10)
ReadAll/16MiB-4      173.16m ± 2%   72.40m ± 3%  -58.19% (p=0.000 n=10)
Write/500KiB-4        2.307m ± 1%   1.835m ± 0%  -20.43% (p=0.000 n=10)
Write/1MiB-4          6.634m ± 1%   3.701m ± 0%  -44.21% (p=0.000 n=10)
Write/4MiB-4          26.64m ± 1%   12.70m ± 0%  -52.35% (p=0.000 n=10)
Write/16MiB-4        116.58m ± 1%   48.78m ± 0%  -58.16% (p=0.000 n=10)
RoundTrip/500KiB-4    7.784m ± 1%   5.029m ± 0%  -35.39% (p=0.000 n=10)
RoundTrip/1MiB-4      17.77m ± 1%   10.20m ± 0%  -42.60% (p=0.000 n=10)
RoundTrip/4MiB-4      69.27m ± 1%   32.23m ± 1%  -53.47% (p=0.000 n=10)
RoundTrip/16MiB-4     276.4m ± 2%   118.2m ± 1%  -57.23% (p=0.000 n=10)
DOMRead/500KiB-4                    3.288m ± 0%
DOMRead/1MiB-4                      6.642m ± 0%
DOMRead/4MiB-4                      19.53m ± 0%
DOMRead/16MiB-4                     67.99m ± 0%
DOMWrite/500KiB-4                   1.807m ± 0%
DOMWrite/1MiB-4                     3.661m ± 0%
DOMWrite/4MiB-4                     12.65m ± 0%
DOMWrite/16MiB-4                    48.89m ± 0%
geomean               23.63m        12.09m       -44.37%

                   │    sonic     │                 yyjson                 │
                   │     B/s      │      B/s       vs base                 │
Read/500KiB-4        124.0Mi ± 1%    153.0Mi ± 0%   +23.33% (p=0.000 n=10)
Read/1MiB-4          114.5Mi ± 1%    152.9Mi ± 0%   +33.58% (p=0.000 n=10)
Read/4MiB-4          99.67Mi ± 1%   205.63Mi ± 0%  +106.32% (p=0.000 n=10)
Read/16MiB-4         95.56Mi ± 1%   235.76Mi ± 0%  +146.70% (p=0.000 n=10)
ReadAll/500KiB-4     121.2Mi ± 1%    146.2Mi ± 0%   +20.64% (p=0.000 n=10)
ReadAll/1MiB-4       99.15Mi ± 1%   145.59Mi ± 0%   +46.84% (p=0.000 n=10)
ReadAll/4MiB-4       86.47Mi ± 1%   189.28Mi ± 1%  +118.89% (p=0.000 n=10)
ReadAll/16MiB-4      92.40Mi ± 2%   220.99Mi ± 3%  +139.17% (p=0.000 n=10)
Write/500KiB-4       211.7Mi ± 1%    266.0Mi ± 0%   +25.67% (p=0.000 n=10)
Write/1MiB-4         150.7Mi ± 1%    270.2Mi ± 0%   +79.25% (p=0.000 n=10)
Write/4MiB-4         150.1Mi ± 1%    315.0Mi ± 0%  +109.84% (p=0.000 n=10)
Write/16MiB-4        137.2Mi ± 1%    328.0Mi ± 0%  +139.01% (p=0.000 n=10)
RoundTrip/500KiB-4   62.73Mi ± 1%    97.09Mi ± 0%   +54.77% (p=0.000 n=10)
RoundTrip/1MiB-4     56.29Mi ± 1%    98.06Mi ± 0%   +74.21% (p=0.000 n=10)
RoundTrip/4MiB-4     57.74Mi ± 1%   124.10Mi ± 1%  +114.91% (p=0.000 n=10)
RoundTrip/16MiB-4    57.88Mi ± 2%   135.32Mi ± 1%  +133.80% (p=0.000 n=10)
DOMRead/500KiB-4                     148.5Mi ± 0%
DOMRead/1MiB-4                       150.6Mi ± 0%
DOMRead/4MiB-4                       204.8Mi ± 0%
DOMRead/16MiB-4                      235.3Mi ± 0%
DOMWrite/500KiB-4                    270.2Mi ± 0%
DOMWrite/1MiB-4                      273.1Mi ± 0%
DOMWrite/4MiB-4                      316.3Mi ± 0%
DOMWrite/16MiB-4                     327.2Mi ± 0%
geomean              100.0Mi         195.6Mi        +79.77%

                   │      sonic       │                yyjson                 │
                   │       B/op       │     B/op      vs base                 │
Read/500KiB-4          5314537.5 ± 0%     808.0 ± 0%   -99.98% (p=0.000 n=10)
Read/1MiB-4           10880919.5 ± 0%     808.0 ± 0%   -99.99% (p=0.000 n=10)
Read/4MiB-4           43157146.0 ± 0%     808.0 ± 0%  -100.00% (p=0.000 n=10)
Read/16MiB-4         166956150.0 ± 0%     808.0 ± 0%  -100.00% (p=0.000 n=10)
ReadAll/500KiB-4         5.070Mi ± 0%   1.008Mi ± 0%   -80.11% (p=0.000 n=10)
ReadAll/1MiB-4          13.828Mi ± 0%   2.126Mi ± 0%   -84.63% (p=0.000 n=10)
ReadAll/4MiB-4          58.528Mi ± 0%   9.532Mi ± 0%   -83.71% (p=0.000 n=10)
ReadAll/16MiB-4         217.75Mi ± 0%   34.55Mi ± 0%   -84.13% (p=0.000 n=10)
Write/500KiB-4           955.0Ki ± 0%   504.5Ki ± 0%   -47.18% (p=0.000 n=10)
Write/1MiB-4             5.888Mi ± 0%   1.000Mi ± 0%   -83.01% (p=0.000 n=10)
Write/4MiB-4            23.638Mi ± 0%   4.000Mi ± 0%   -83.08% (p=0.000 n=10)
Write/16MiB-4           111.71Mi ± 0%   16.00Mi ± 0%   -85.68% (p=0.000 n=10)
RoundTrip/500KiB-4      6270.9Ki ± 0%   505.0Ki ± 0%   -91.95% (p=0.000 n=10)
RoundTrip/1MiB-4        16.288Mi ± 0%   1.001Mi ± 0%   -93.85% (p=0.000 n=10)
RoundTrip/4MiB-4        64.876Mi ± 0%   4.001Mi ± 0%   -93.83% (p=0.000 n=10)
RoundTrip/16MiB-4       271.07Mi ± 0%   16.00Mi ± 0%   -94.10% (p=0.000 n=10)
DOMRead/500KiB-4                        2.398Ki ± 0%
DOMRead/1MiB-4                          2.398Ki ± 0%
DOMRead/4MiB-4                          2.398Ki ± 0%
DOMRead/16MiB-4                         2.398Ki ± 0%
DOMWrite/500KiB-4                       504.5Ki ± 0%
DOMWrite/1MiB-4                         1.000Mi ± 0%
DOMWrite/4MiB-4                         4.000Mi ± 0%
DOMWrite/16MiB-4                        16.00Mi ± 0%
geomean                  23.35Mi        228.6Ki        -98.35%

                   │      sonic      │               yyjson                │
                   │    allocs/op    │ allocs/op   vs base                 │
Read/500KiB-4          52816.00 ± 0%   10.00 ± 0%   -99.98% (p=0.000 n=10)
Read/1MiB-4           107762.00 ± 0%   10.00 ± 0%   -99.99% (p=0.000 n=10)
Read/4MiB-4           422211.00 ± 0%   10.00 ± 0%  -100.00% (p=0.000 n=10)
Read/16MiB-4         1662517.00 ± 0%   10.00 ± 0%  -100.00% (p=0.000 n=10)
ReadAll/500KiB-4       52818.00 ± 0%   33.00 ± 0%   -99.94% (p=0.000 n=10)
ReadAll/1MiB-4        107780.00 ± 0%   35.00 ± 0%   -99.97% (p=0.000 n=10)
ReadAll/4MiB-4        422234.00 ± 0%   39.00 ± 0%   -99.99% (p=0.000 n=10)
ReadAll/16MiB-4      1662544.00 ± 0%   42.00 ± 0%  -100.00% (p=0.000 n=10)
Write/500KiB-4         4803.000 ± 0%   7.000 ± 0%   -99.85% (p=0.000 n=10)
Write/1MiB-4           9820.000 ± 0%   7.000 ± 0%   -99.93% (p=0.000 n=10)
Write/4MiB-4          38413.000 ± 0%   7.000 ± 0%   -99.98% (p=0.000 n=10)
Write/16MiB-4        151178.000 ± 0%   7.000 ± 0%  -100.00% (p=0.000 n=10)
RoundTrip/500KiB-4     57620.00 ± 0%   15.00 ± 0%   -99.97% (p=0.000 n=10)
RoundTrip/1MiB-4      117583.00 ± 0%   15.00 ± 0%   -99.99% (p=0.000 n=10)
RoundTrip/4MiB-4      460627.00 ± 0%   15.00 ± 0%  -100.00% (p=0.000 n=10)
RoundTrip/16MiB-4    1813699.00 ± 0%   15.00 ± 0%  -100.00% (p=0.000 n=10)
DOMRead/500KiB-4                       29.00 ± 0%
DOMRead/1MiB-4                         29.00 ± 0%
DOMRead/4MiB-4                         29.00 ± 0%
DOMRead/16MiB-4                        29.00 ± 0%
DOMWrite/500KiB-4                      7.000 ± 0%
DOMWrite/1MiB-4                        7.000 ± 0%
DOMWrite/4MiB-4                        7.000 ± 0%
DOMWrite/16MiB-4                       7.000 ± 0%
geomean                  141.1k        14.11        -99.99%
```
</details>

<details>
  <summary>vs. simdjson-go</summary>

```
goos: linux
goarch: amd64
pkg: benchmarks
cpu: AMD EPYC 9V74 80-Core Processor
                   │  simdjson   │                yyjson                │
                   │   sec/op    │    sec/op     vs base                │
Read/500KiB-4        2.639m ± 1%    3.192m ± 0%  +20.96% (p=0.000 n=10)
Read/1MiB-4          5.333m ± 1%    6.539m ± 0%  +22.61% (p=0.000 n=10)
Read/4MiB-4          19.76m ± 2%    19.45m ± 0%        ~ (p=0.143 n=10)
Read/16MiB-4         77.00m ± 2%    67.87m ± 0%  -11.87% (p=0.000 n=10)
ReadAll/500KiB-4     2.715m ± 1%    3.340m ± 0%  +23.05% (p=0.000 n=10)
ReadAll/1MiB-4       5.664m ± 2%    6.868m ± 0%  +21.27% (p=0.000 n=10)
ReadAll/4MiB-4       20.17m ± 3%    21.13m ± 1%   +4.76% (p=0.000 n=10)
ReadAll/16MiB-4      81.88m ± 1%    72.40m ± 3%  -11.58% (p=0.000 n=10)
Write/500KiB-4       1.929m ± 0%    1.835m ± 0%   -4.84% (p=0.000 n=10)
Write/1MiB-4         3.896m ± 1%    3.701m ± 0%   -5.00% (p=0.000 n=10)
Write/4MiB-4         15.07m ± 0%    12.70m ± 0%  -15.76% (p=0.000 n=10)
Write/16MiB-4        59.09m ± 2%    48.78m ± 0%  -17.45% (p=0.000 n=10)
RoundTrip/500KiB-4   4.511m ± 1%    5.029m ± 0%  +11.48% (p=0.000 n=10)
RoundTrip/1MiB-4     9.140m ± 1%   10.198m ± 0%  +11.58% (p=0.000 n=10)
RoundTrip/4MiB-4     36.71m ± 1%    32.23m ± 1%  -12.21% (p=0.000 n=10)
RoundTrip/16MiB-4    140.1m ± 1%    118.2m ± 1%  -15.58% (p=0.000 n=10)
DOMRead/500KiB-4                    3.288m ± 0%
DOMRead/1MiB-4                      6.642m ± 0%
DOMRead/4MiB-4                      19.53m ± 0%
DOMRead/16MiB-4                     67.99m ± 0%
DOMWrite/500KiB-4                   1.807m ± 0%
DOMWrite/1MiB-4                     3.661m ± 0%
DOMWrite/4MiB-4                     12.65m ± 0%
DOMWrite/16MiB-4                    48.89m ± 0%
geomean              13.12m         12.09m        +0.19%

                   │   simdjson    │                yyjson                │
                   │      B/s      │     B/s       vs base                │
Read/500KiB-4         185.0Mi ± 1%   153.0Mi ± 0%  -17.33% (p=0.000 n=10)
Read/1MiB-4           187.5Mi ± 1%   152.9Mi ± 0%  -18.44% (p=0.000 n=10)
Read/4MiB-4           202.4Mi ± 2%   205.6Mi ± 0%        ~ (p=0.138 n=10)
Read/16MiB-4          207.8Mi ± 2%   235.8Mi ± 0%  +13.46% (p=0.000 n=10)
ReadAll/500KiB-4      179.9Mi ± 1%   146.2Mi ± 0%  -18.73% (p=0.000 n=10)
ReadAll/1MiB-4        176.6Mi ± 2%   145.6Mi ± 0%  -17.54% (p=0.000 n=10)
ReadAll/4MiB-4        198.3Mi ± 2%   189.3Mi ± 1%   -4.54% (p=0.000 n=10)
ReadAll/16MiB-4       195.4Mi ± 1%   221.0Mi ± 3%  +13.09% (p=0.000 n=10)
Write/500KiB-4        253.2Mi ± 0%   266.0Mi ± 0%   +5.08% (p=0.000 n=10)
Write/1MiB-4          256.7Mi ± 1%   270.2Mi ± 0%   +5.26% (p=0.000 n=10)
Write/4MiB-4          265.4Mi ± 0%   315.0Mi ± 0%  +18.70% (p=0.000 n=10)
Write/16MiB-4         270.8Mi ± 2%   328.0Mi ± 0%  +21.14% (p=0.000 n=10)
RoundTrip/500KiB-4   108.24Mi ± 1%   97.09Mi ± 0%  -10.30% (p=0.000 n=10)
RoundTrip/1MiB-4     109.41Mi ± 1%   98.06Mi ± 0%  -10.38% (p=0.000 n=10)
RoundTrip/4MiB-4      109.0Mi ± 1%   124.1Mi ± 1%  +13.90% (p=0.000 n=10)
RoundTrip/16MiB-4     114.2Mi ± 1%   135.3Mi ± 1%  +18.46% (p=0.000 n=10)
DOMRead/500KiB-4                     148.5Mi ± 0%
DOMRead/1MiB-4                       150.6Mi ± 0%
DOMRead/4MiB-4                       204.8Mi ± 0%
DOMRead/16MiB-4                      235.3Mi ± 0%
DOMWrite/500KiB-4                    270.2Mi ± 0%
DOMWrite/1MiB-4                      273.1Mi ± 0%
DOMWrite/4MiB-4                      316.3Mi ± 0%
DOMWrite/16MiB-4                     327.2Mi ± 0%
geomean               180.2Mi        195.6Mi        -0.19%

                   │     simdjson     │                yyjson                 │
                   │       B/op       │     B/op      vs base                 │
Read/500KiB-4          4449836.0 ± 0%     808.0 ± 0%   -99.98% (p=0.000 n=10)
Read/1MiB-4            9004590.5 ± 0%     808.0 ± 0%   -99.99% (p=0.000 n=10)
Read/4MiB-4           35489331.5 ± 0%     808.0 ± 0%  -100.00% (p=0.000 n=10)
Read/16MiB-4         141420072.0 ± 0%     808.0 ± 0%  -100.00% (p=0.000 n=10)
ReadAll/500KiB-4         5.251Mi ± 0%   1.008Mi ± 0%   -80.80% (p=0.000 n=10)
ReadAll/1MiB-4          10.712Mi ± 0%   2.126Mi ± 0%   -80.16% (p=0.000 n=10)
ReadAll/4MiB-4          43.376Mi ± 0%   9.532Mi ± 0%   -78.03% (p=0.000 n=10)
ReadAll/16MiB-4         169.42Mi ± 0%   34.55Mi ± 0%   -79.61% (p=0.000 n=10)
Write/500KiB-4          2478.7Ki ± 0%   504.5Ki ± 0%   -79.65% (p=0.000 n=10)
Write/1MiB-4             4.999Mi ± 0%   1.000Mi ± 0%   -79.99% (p=0.000 n=10)
Write/4MiB-4            20.132Mi ± 0%   4.000Mi ± 0%   -80.13% (p=0.000 n=10)
Write/16MiB-4            97.88Mi ± 0%   16.00Mi ± 0%   -83.65% (p=0.000 n=10)
RoundTrip/500KiB-4      6824.3Ki ± 0%   505.0Ki ± 0%   -92.60% (p=0.000 n=10)
RoundTrip/1MiB-4        13.586Mi ± 0%   1.001Mi ± 0%   -92.63% (p=0.000 n=10)
RoundTrip/4MiB-4        53.977Mi ± 0%   4.001Mi ± 0%   -92.59% (p=0.000 n=10)
RoundTrip/16MiB-4       232.75Mi ± 0%   16.00Mi ± 0%   -93.13% (p=0.000 n=10)
DOMRead/500KiB-4                        2.398Ki ± 0%
DOMRead/1MiB-4                          2.398Ki ± 0%
DOMRead/4MiB-4                          2.398Ki ± 0%
DOMRead/16MiB-4                         2.398Ki ± 0%
DOMWrite/500KiB-4                       504.5Ki ± 0%
DOMWrite/1MiB-4                         1.000Mi ± 0%
DOMWrite/4MiB-4                         4.000Mi ± 0%
DOMWrite/16MiB-4                        16.00Mi ± 0%
geomean                  21.35Mi        228.6Ki        -98.19%

                   │  simdjson   │               yyjson               │
                   │  allocs/op  │ allocs/op   vs base                │
Read/500KiB-4         16.00 ± 0%   10.00 ± 0%  -37.50% (p=0.000 n=10)
Read/1MiB-4           16.00 ± 0%   10.00 ± 0%  -37.50% (p=0.000 n=10)
Read/4MiB-4           16.00 ± 0%   10.00 ± 0%  -37.50% (p=0.000 n=10)
Read/16MiB-4          16.00 ± 0%   10.00 ± 0%  -37.50% (p=0.000 n=10)
ReadAll/500KiB-4      39.00 ± 0%   33.00 ± 0%  -15.38% (p=0.000 n=10)
ReadAll/1MiB-4        41.00 ± 0%   35.00 ± 0%  -14.63% (p=0.000 n=10)
ReadAll/4MiB-4        45.00 ± 0%   39.00 ± 0%  -13.33% (p=0.000 n=10)
ReadAll/16MiB-4       48.00 ± 0%   42.00 ± 0%  -12.50% (p=0.000 n=10)
Write/500KiB-4       30.000 ± 0%   7.000 ± 0%  -76.67% (p=0.000 n=10)
Write/1MiB-4         33.000 ± 0%   7.000 ± 0%  -78.79% (p=0.000 n=10)
Write/4MiB-4         39.000 ± 0%   7.000 ± 0%  -82.05% (p=0.000 n=10)
Write/16MiB-4        46.000 ± 0%   7.000 ± 0%  -84.78% (p=0.000 n=10)
RoundTrip/500KiB-4    46.00 ± 0%   15.00 ± 0%  -67.39% (p=0.000 n=10)
RoundTrip/1MiB-4      49.00 ± 0%   15.00 ± 0%  -69.39% (p=0.000 n=10)
RoundTrip/4MiB-4      55.00 ± 0%   15.00 ± 0%  -72.73% (p=0.000 n=10)
RoundTrip/16MiB-4     62.00 ± 0%   15.00 ± 0%  -75.81% (p=0.000 n=10)
DOMRead/500KiB-4                   29.00 ± 0%
DOMRead/1MiB-4                     29.00 ± 0%
DOMRead/4MiB-4                     29.00 ± 0%
DOMRead/16MiB-4                    29.00 ± 0%
DOMWrite/500KiB-4                  7.000 ± 0%
DOMWrite/1MiB-4                    7.000 ± 0%
DOMWrite/4MiB-4                    7.000 ± 0%
DOMWrite/16MiB-4                   7.000 ± 0%
geomean               33.93        14.11       -58.60%
```
</details>

### Summary

Run on an AMD EPYC 9V74, 10 iterations each, across payload sizes from 500 KiB to
16 MiB. Percentages are benchstat geomean deltas over rows present in both packages.
In non-std comparisons, [`DOMRead`](https://pkg.go.dev/go.dw1.io/yyjson#DOMRead)
and [`DOMWrite`](https://pkg.go.dev/go.dw1.io/yyjson#DOMWrite) are yyjson-only
rows and don't factor into those package deltas.

| pkg | Speed (geomean) | Go heap memory (geomean) | Go heap allocations (geomean) |
|---|---|---|---|
| `encoding/json` (std) | **-72.6%** | -98.7% | -100% |
| `json-iterator` | **-64.9%** | -97.8% | -100% |
| `segmentio/encoding/json` | **-61.1%** | -97.0% | -99.9% |
| `go-json` | **-68.4%** | -97.7% | -100% |
| `bytedance/sonic` | **-44.4%** | -98.4% | -100% |
| `simdjson-go` | **+0.2%** | -98.2% | -58.6% |

> [!NOTE]
> Memory and allocation columns come from Go heap bytes/op and
> allocs/op. Arena memory used internally by yyjson is not counted.

Key takeaways:

- **2-4x faster than the standard library** across document operations. The gap
  grows with payload size - at 16 MiB, reads are nearly **4x faster** and writes
  nearly **5x faster**.
- **Far fewer Go heap allocations.** [`Read`](https://pkg.go.dev/go.dw1.io/yyjson#Read)
  uses roughly **10 Go heap objects per call** regardless of payload size.
  Reflection-based libraries in this benchmark reach into the hundreds of thousands.
- **Competitive with SIMD-accelerated libraries.** yyjson is **~44% faster** than
  sonic (JIT-based) and essentially matches simdjson-go (SIMD-based) overall.

Run benchmarks yourself:

```bash
make -C benchmarks bench
make -C benchmarks benchstats
```

## Development

See [CONTRIBUTING.md](/docs/CONTRIBUTING.md) for contribution guidelines.
See [DEVELOPMENT.md](/docs/DEVELOPMENT.md) too.

## License

This package is licensed under the Apache License 2.0. See [LICENSE](/LICENSE).

Submodules are licensed under their own terms. See [`yyjson/`](/yyjson/) for the upstream
license.
