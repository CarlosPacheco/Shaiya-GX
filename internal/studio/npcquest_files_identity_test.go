package studio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNPCQuestFilesMustHaveSeparateIdentities(t *testing.T) {
	dir := t.TempDir()
	client, alias := filepath.Join(dir, "client"), filepath.Join(dir, "alias")
	if err := os.WriteFile(client, []byte("npcquest"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(client, alias); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	if err := distinctNPCQuestFiles(client, alias); err == nil {
		t.Fatal("an alias passed as a separate destination")
	}
	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(alias, []byte("npcquest"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := distinctNPCQuestFiles(client, alias); err != nil {
		t.Fatal(err)
	}
}
