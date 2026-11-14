package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiagnosticsFallbackRotationAndFailure(t *testing.T) {
	directory := t.TempDir()
	blocked := filepath.Join(directory, "not-a-directory")
	if err := os.WriteFile(blocked, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(directory, "logs")
	for _, text := range []string{"first session", "second session", "third session"} {
		d := openDiagnostics([]string{filepath.Join(blocked, "logs"), target})
		if d.path != filepath.Join(target, "client.log") {
			t.Fatalf("fallback path: %q", d.path)
		}
		log.Print(text)
		d.data("resources")
		d.fail(errors.New("interface/example.png unavailable"), false)
		d.Close()
	}
	current, err := os.ReadFile(filepath.Join(target, "client.log"))
	if err != nil {
		t.Fatal(err)
	}
	previous, err := os.ReadFile(filepath.Join(target, "client.previous.log"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"third session", "Data directory:", "interface/example.png unavailable", "goroutine"} {
		if !strings.Contains(string(current), expected) {
			t.Fatalf("missing diagnostic %q", expected)
		}
	}
	if !strings.Contains(string(previous), "second session") || strings.Contains(string(previous), "first session") {
		t.Fatal("rotation did not retain the preceding session")
	}
	if strings.Contains(string(current), "second session") {
		t.Fatal("current log was not reset")
	}
}

// Windows GUI executables can have no usable standard-error handle.
type unavailableConsole struct{}

func (unavailableConsole) Write([]byte) (int, error) { return 0, errors.New("no console") }

func TestDiagnosticsWithoutConsole(t *testing.T) {
	previous := log.Writer()
	log.SetOutput(unavailableConsole{})
	defer log.SetOutput(previous)
	d := openDiagnostics([]string{t.TempDir()})
	log.Print("diagnostic survives absent console")
	d.Close()
	contents, err := os.ReadFile(d.path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "diagnostic survives absent console") {
		t.Fatal("GUI launch lost its diagnostic")
	}
}
