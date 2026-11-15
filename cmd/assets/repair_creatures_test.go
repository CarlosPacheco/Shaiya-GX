package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
)

func creatureFixture() []byte {
	var b bytes.Buffer
	write := func(v any) {
		if err := binary.Write(&b, binary.LittleEndian, v); err != nil {
			panic(err)
		}
	}
	write(uint32(0))
	write(uint32(1))
	write([16]float32{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1})
	write(uint32(3))
	for _, p := range [][3]float32{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}} {
		write(p)
		write(float32(1))
		write([4]byte{})
		write([3]float32{0, 0, -1})
		write([2]float32{})
	}
	write(uint32(1))
	write([3]uint16{0, 1, 2})
	return b.Bytes()
}
func texturedFixture(canonical []byte) []byte {
	name := []byte("Mob_Rend_01.TGA\x00")
	var b bytes.Buffer
	binary.Write(&b, binary.LittleEndian, uint32(len(name)))
	b.Write(name)
	b.Write(canonical[4:])
	return b.Bytes()
}
func TestOfflineCreatureRepairPreservesEntireMeshPayload(t *testing.T) {
	canonical := creatureFixture()
	original := texturedFixture(canonical)
	if _, err := model.Parse3DC(original); err == nil {
		t.Fatal("fixture did not reproduce runtime rejection")
	}
	before := append([]byte(nil), original...)
	repaired, texture, err := normalizeCreatureHeader(original)
	if err != nil || texture != "Mob_Rend_01.TGA" || !bytes.Equal(repaired, canonical) || !bytes.Equal(original, before) {
		t.Fatal("repair changed payload or input", texture, err)
	}
	mesh, err := model.Parse3DC(repaired)
	if err != nil || len(mesh.Bones) != 1 || len(mesh.Vertices) != 3 || len(mesh.Faces) != 1 {
		t.Fatal("unchanged parser rejected canonical data", mesh, err)
	}
	for _, bad := range [][]byte{original[:10], append([]byte{255, 255, 255, 255}, original[4:]...), append([]byte(nil), original...), original[:len(original)-1], append(original, 0)} {
		if bytes.Equal(bad, original) {
			bad[19] = 'X'
		} // Missing terminating zero.
		if _, _, err := normalizeCreatureHeader(bad); err == nil {
			t.Fatal("unrecognized damage accepted")
		}
	}
}
func TestCreatureRepairStagesBackupWithoutChangingSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	output := filepath.Join(root, "repair")
	if err := os.MkdirAll(filepath.Join(source, "Monster", "3DC"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(source, "Npc", "3DC"), 0755); err != nil {
		t.Fatal(err)
	}
	canonical := creatureFixture()
	original := texturedFixture(canonical)
	monster := filepath.Join(source, "Monster", "3DC", "broken.3DC")
	npc := filepath.Join(source, "Npc", "3DC", "valid.3DC")
	os.WriteFile(monster, original, 0644)
	os.WriteFile(npc, canonical, 0644)
	arc, err := archive.OpenDirectory(source)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	if err := repairCreatureModels(arc, source, filepath.Join(source, "nested")); err == nil {
		t.Fatal("allowed output inside source")
	}
	if err := repairCreatureModels(arc, source, output); err != nil {
		t.Fatal(err)
	}
	for file, want := range map[string][]byte{monster: original, npc: canonical, filepath.Join(output, "originals", "Monster", "3DC", "broken.3DC"): original, filepath.Join(output, "data", "Monster", "3DC", "broken.3DC"): canonical} {
		got, err := os.ReadFile(file)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatal("source/backup/payload changed", file, err)
		}
	}
	if _, err := os.Stat(filepath.Join(output, "data", "Npc", "3DC", "valid.3DC")); !os.IsNotExist(err) {
		t.Fatal("rewrote accepted NPC model")
	}
	if err := repairCreatureModels(arc, source, output); err == nil {
		t.Fatal("overwrote existing repair output")
	}
}
func TestUnknownCreatureDamageDoesNotWritePartialRepair(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	output := filepath.Join(root, "repair")
	os.MkdirAll(filepath.Join(source, "Monster", "3DC"), 0755)
	os.WriteFile(filepath.Join(source, "Monster", "3DC", "repairable.3DC"), texturedFixture(creatureFixture()), 0644)
	os.WriteFile(filepath.Join(source, "Monster", "3DC", "unknown.3DC"), []byte("not a supported model"), 0644)
	arc, err := archive.OpenDirectory(source)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	if err := repairCreatureModels(arc, source, output); err == nil || !strings.Contains(err.Error(), "unrecognized") {
		t.Fatal("unknown header guessed", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("partial repair written despite unknown corruption")
	}
}
