package yyjson_test

import (
	"bytes"
	"fmt"

	"go.dw1.io/yyjson"
)

func ExampleReadString() {
	doc, err := yyjson.ReadString(`{"name":"yyjson","items":[1,2]}`)
	if err != nil {
		panic(err)
	}
	defer doc.Close()

	root := doc.Root()
	fmt.Println(root.Get("name").String())
	fmt.Println(root.Get("items").Index(1).Uint())
	// Output:
	// yyjson
	// 2
}

func ExampleMutDoc() {
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
	// Output:
	// {"name":"yyjson","ok":true}
}

func ExampleDoc_Pointer() {
	doc, err := yyjson.ReadString(`{"items":[{"name":"first"},{"name":"second"}]}`)
	if err != nil {
		panic(err)
	}
	defer doc.Close()

	val, err := doc.Pointer("/items/1/name")
	if err != nil {
		panic(err)
	}
	fmt.Println(val.String())
	// Output:
	// second
}

func ExampleMutDoc_Patch() {
	orig, err := yyjson.ReadString(`{"name":"old"}`)
	if err != nil {
		panic(err)
	}
	defer orig.Close()
	patch, err := yyjson.ReadString(`[{"op":"replace","path":"/name","value":"new"}]`)
	if err != nil {
		panic(err)
	}
	defer patch.Close()

	doc, err := yyjson.NewMutDoc()
	if err != nil {
		panic(err)
	}
	defer doc.Close()
	patched, err := doc.Patch(orig.Root(), patch.Root())
	if err != nil {
		panic(err)
	}
	doc.SetRoot(patched)
	out, err := doc.WriteString()
	if err != nil {
		panic(err)
	}
	fmt.Println(out)
	// Output:
	// {"name":"new"}
}

func ExampleMarshal() {
	out, err := yyjson.Marshal(struct {
		Name string `json:"name"`
	}{Name: "yyjson"})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(out))
	// Output:
	// {"name":"yyjson"}
}

func ExampleUnmarshal() {
	var value struct {
		Name string `json:"name"`
	}
	if err := yyjson.Unmarshal([]byte(`{"name":"yyjson"}`), &value); err != nil {
		panic(err)
	}
	fmt.Println(value.Name)
	// Output:
	// yyjson
}

func ExampleDoc_WriteTo() {
	doc, err := yyjson.ReadString(`{"ok":true}`)
	if err != nil {
		panic(err)
	}
	defer doc.Close()

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		panic(err)
	}
	fmt.Println(buf.String())
	// Output:
	// {"ok":true}
}

func ExampleNewIncrReader() {
	reader, err := yyjson.NewIncrReader(len(`{"ok":true}`))
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	if err := reader.Append([]byte(`{"ok":true`)); err != nil {
		panic(err)
	}
	if _, err := reader.Read(); err == nil {
		panic("expected more data")
	}
	if err := reader.Append([]byte(`}`)); err != nil {
		panic(err)
	}
	doc, err := reader.Read()
	if err != nil {
		panic(err)
	}
	defer doc.Close()
	fmt.Println(doc.Root().Get("ok").Bool())
	// Output:
	// true
}
