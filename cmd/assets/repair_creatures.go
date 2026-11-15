package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
)

type creatureRepair struct {
	File                string `json:"file"`
	EmbeddedTexture     string `json:"embedded_texture"`
	OriginalSHA256      string `json:"original_sha256"`
	CanonicalSHA256     string `json:"canonical_sha256"`
	PayloadSHA256       string `json:"payload_sha256"`
	Bones               int    `json:"bones"`
	Vertices            int    `json:"vertices"`
	Faces               int    `json:"faces"`
	original, canonical []byte
}

func digest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

// normalizeCreatureHeader is an offline repair only. Parse3DC and runtime
// loaders retain their current format. Every byte after the legacy texture
// header is preserved, including inverse bind matrices and vertex weights.
func normalizeCreatureHeader(raw []byte) ([]byte, string, error) {
	if len(raw) < 12 {
		return nil, "", fmt.Errorf("short model")
	}
	n := int(binary.LittleEndian.Uint32(raw))
	if n < 2 || n > 512 || 4+n+4 > len(raw) || raw[4+n-1] != 0 {
		return nil, "", fmt.Errorf("not a terminated texture-name header")
	}
	name := strings.TrimRight(string(raw[4:4+n]), "\x00")
	for _, b := range []byte(name) {
		if b < 32 || b > 126 {
			return nil, "", fmt.Errorf("invalid embedded texture name")
		}
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".dds", ".tga", ".png", ".bmp", ".jpg", ".jpeg":
	default:
		return nil, "", fmt.Errorf("embedded name is not a known texture filename")
	}
	normalized := make([]byte, 4, len(raw)-n)
	normalized = append(normalized, raw[4+n:]...)
	mesh, err := model.Parse3DC(normalized)
	if err != nil {
		return nil, "", fmt.Errorf("canonical payload: %w", err)
	}
	if len(mesh.Bones) == 0 || len(mesh.Vertices) == 0 || len(mesh.Faces) == 0 || len(normalized) != 16+len(mesh.Bones)*64+len(mesh.Vertices)*40+len(mesh.Faces)*6 {
		return nil, "", fmt.Errorf("incomplete or trailing canonical mesh payload")
	}
	for _, face := range mesh.Faces {
		for _, index := range face {
			if int(index) >= len(mesh.Vertices) {
				return nil, "", fmt.Errorf("invalid face index")
			}
		}
	}
	return normalized, name, nil
}

// repairCreatureModels stages canonical copies and exact originals separately.
// It never mutates the supplied data directory or guesses an unknown layout.
func repairCreatureModels(a *archive.Archive, source, output string) error {
	src, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	dest, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(src, dest)
	if err != nil || rel == "." || !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("repair output must be outside source data")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		return fmt.Errorf("repair output must not already exist")
	}
	scanned := 0
	var repairs []creatureRepair
	var problems []string
	a.Walk(func(file string, entry *archive.File) {
		lower := strings.ToLower(file)
		if !strings.HasSuffix(lower, ".3dc") || !(strings.HasPrefix(lower, "monster/") || strings.HasPrefix(lower, "npc/")) {
			return
		}
		scanned++
		raw, err := a.Read(entry)
		if err != nil {
			problems = append(problems, file+": "+err.Error())
			return
		}
		if mesh, err := model.Parse3DC(raw); err == nil && len(mesh.Vertices) > 0 && len(mesh.Faces) > 0 {
			return
		}
		normalized, name, err := normalizeCreatureHeader(raw)
		if err != nil {
			problems = append(problems, file+": "+err.Error())
			return
		}
		mesh, _ := model.Parse3DC(normalized)
		repairs = append(repairs, creatureRepair{File: file, EmbeddedTexture: name, OriginalSHA256: digest(raw), CanonicalSHA256: digest(normalized), PayloadSHA256: digest(normalized[4:]), Bones: len(mesh.Bones), Vertices: len(mesh.Vertices), Faces: len(mesh.Faces), original: raw, canonical: normalized})
	})
	if scanned == 0 {
		return fmt.Errorf("no monster/NPC 3DC files found")
	}
	if len(problems) > 0 {
		return fmt.Errorf("unrecognized creature model damage; no output written:\n%s", strings.Join(problems, "\n"))
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}
	for _, r := range repairs {
		for _, copy := range []struct {
			dir  string
			data []byte
		}{{"originals", r.original}, {"data", r.canonical}} {
			target := filepath.Join(dest, copy.dir, filepath.FromSlash(r.File))
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(target, copy.data, 0644); err != nil {
				return err
			}
		}
	}
	report := struct {
		Scanned   int              `json:"scanned"`
		Unchanged int              `json:"unchanged"`
		Repairs   []creatureRepair `json:"repairs"`
	}{scanned, scanned - len(repairs), repairs}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dest, "report.json"), append(raw, '\n'), 0644); err != nil {
		return err
	}
	fmt.Printf("Creature meshes: %d scanned, %d unchanged, %d canonical repairs staged; source data untouched.\n", scanned, report.Unchanged, len(repairs))
	return nil
}
