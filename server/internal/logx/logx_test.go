package logx

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestStructuredRotationAndConcurrentTags(t *testing.T) {
	t.Chdir(t.TempDir())
	log, err := New("rotation")
	if err != nil {
		t.Fatal(err)
	}
	log.maxBytes = 400
	log.backups = 3
	tag := log.Tagged("first")
	var wg sync.WaitGroup
	for i := range 4 {
		wg.Go(func() {
			for range 10 {
				tag.SetTag(string(rune('a' + i)))
				tag.Printf("message with newline\nvalue")
			}
		})
	}
	wg.Wait()
	if err := log.Close(); err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob("logs/rotation.log*")
	if err != nil || len(files) != 4 {
		t.Fatal(files, err)
	}
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			var row map[string]any
			if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
				t.Fatal(err)
			}
			if row["component"] != "rotation" || row["session"] == nil || row["time"] == nil {
				t.Fatal(row)
			}
		}
		f.Close()
		if err := scanner.Err(); err != nil {
			t.Fatal(err)
		}
	}
}
