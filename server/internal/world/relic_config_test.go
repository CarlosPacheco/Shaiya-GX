package world

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOriginalObeliskConfiguration(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(gameDataDir(), "Obelisk.ini"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := parseRelicConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.relics) != 9 || len(cfg.bosses) != 25 {
		t.Fatalf("relics=%d bosses=%d", len(cfg.relics), len(cfg.bosses))
	}
	if r := cfg.relics[0]; r.key != 1 || r.mobs != [3]uint16{52, 252, 152} || r.pos != [3]float32{155.10, 39.43, 190.38} {
		t.Fatalf("first relic %+v", r)
	}
	if cfg.relics[5].initial != 1 || cfg.relics[6].initial != 2 || cfg.relics[7].key != 18<<16|1 {
		t.Fatal("country/key decoding")
	}
	if b := cfg.bosses[23]; !b.random || len(b.mobs) != 1 || b.mobs[0] != 2414 {
		t.Fatalf("random boss %+v", b)
	}
	t.Logf("original configuration: %d relics, %d bosses, %d portal declarations", len(cfg.relics), len(cfg.bosses), len(cfg.portals))
}
