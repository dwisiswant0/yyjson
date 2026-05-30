package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDropHostCompilerDefinitions(t *testing.T) {
	in := []byte(
		"const YYJSON_VERSION_MAJOR = 0\n" +
			"const __GNUC__ = 13\n" +
			"const __VERSION__ = \"13.3.0\"\n" +
			"const FP_FAST_FMA = 1\n" +
			"type T__predefined_wchar_t = uint32\n" +
			"const _GNU_SOURCE = 1\n" +
			"func keep() {}\n",
	)
	want := []byte(
		"const YYJSON_VERSION_MAJOR = 0\n" +
			"const _GNU_SOURCE = 1\n" +
			"func keep() {}\n",
	)

	got := dropHostCompilerDefinitions(in)
	if !bytes.Equal(got, want) {
		t.Fatalf("dropHostCompilerDefinitions() = %q, want %q", got, want)
	}
}

func TestDisableGeneratedCheckptr(t *testing.T) {
	in := []byte(
		"package yyjsonc\n\n" +
			"func plain() {}\n\n" +
			"func first() {\n" +
			"\t_ = unsafe.Pointer(addr)\n" +
			"}\n\n" +
			"//go:nocheckptr\n" +
			"func alreadyMarked() {\n" +
			"\t_ = unsafe.Pointer(addr)\n" +
			"}\n",
	)
	want := []byte(
		"package yyjsonc\n\n" +
			"func plain() {}\n\n" +
			"//go:nocheckptr\n" +
			"func first() {\n" +
			"\t_ = unsafe.Pointer(addr)\n" +
			"}\n\n" +
			"//go:nocheckptr\n" +
			"func alreadyMarked() {\n" +
			"\t_ = unsafe.Pointer(addr)\n" +
			"}\n",
	)

	got := disableGeneratedCheckptr(in)
	if !bytes.Equal(got, want) {
		t.Fatalf("disableGeneratedCheckptr() = %q, want %q", got, want)
	}
}

func TestParseTargets(t *testing.T) {
	got, err := parseTargets(" linux/amd64,linux/arm64 ")
	if err != nil {
		t.Fatalf("parseTargets() error = %v", err)
	}
	want := []generationTarget{
		{goos: "linux", goarch: "amd64"},
		{goos: "linux", goarch: "arm64"},
	}
	if len(got) != len(want) {
		t.Fatalf("parseTargets() length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseTargets()[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestParseTargetsRejectsInvalidTargets(t *testing.T) {
	for _, in := range []string{"", "linux", "linux/", "/amd64", "linux/amd64/extra"} {
		if _, err := parseTargets(in); err == nil {
			t.Fatalf("parseTargets(%q) error = nil", in)
		}
	}
}

func TestRemoveFiles(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "sidecar.go")
	missing := filepath.Join(dir, "missing.go")
	if err := os.WriteFile(existing, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := removeFiles([]string{existing, missing}); err != nil {
		t.Fatalf("removeFiles() error = %v", err)
	}
	if _, err := os.Stat(existing); !os.IsNotExist(err) {
		t.Fatalf("removed file still exists or stat failed with unexpected error: %v", err)
	}
}

func TestCcgoToolIgnoresGoToolDownloadStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake go command is a POSIX shell script")
	}

	root := t.TempDir()
	fakeBin := t.TempDir()
	fakeGo := filepath.Join(fakeBin, "go")
	script := "#!/bin/sh\n" +
		"printf '%s\\n' 'go: downloading modernc.org/ccgo/v4 v4.34.0' >&2\n" +
		"printf '%s\\n' \"$PWD/ccgo\"\n"
	if err := os.WriteFile(fakeGo, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	got, err := ccgoTool(root)
	if err != nil {
		t.Fatalf("ccgoTool() error = %v", err)
	}
	want := filepath.Join(root, "ccgo")
	if got != want {
		t.Fatalf("ccgoTool() = %q, want %q", got, want)
	}
}

func TestCopyModuleFiles(t *testing.T) {
	root := t.TempDir()
	tmp := t.TempDir()
	for _, name := range []string{"go.mod", "go.sum"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := copyModuleFiles(root, tmp); err != nil {
		t.Fatalf("copyModuleFiles() error = %v", err)
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(tmp, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != name+"\n" {
			t.Fatalf("%s = %q", name, data)
		}
	}
}
