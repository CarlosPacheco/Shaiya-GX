// Package mapnav bakes and queries versioned, layered ground navigation.
package mapnav

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/collision"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
)

const FormatVersion = 1
const CollisionRulesVersion = 2

type Manifest struct {
	CollisionRules                                           int               `json:"collision_rules"`
	PlayerPhysics                                            int               `json:"player_physics,omitempty"`
	Version                                                  int               `json:"version"`
	MapID                                                    uint16            `json:"map_id"`
	Files                                                    map[string]string `json:"files"`
	Sources                                                  map[string]string `json:"sources"`
	AgentHeight, AgentRadius, MaxClimb, CellSize, CellHeight float32
	TileSize                                                 int `json:"tile_size"`
	Triangles, Polygons                                      int
}
type physicsFile struct {
	Version   int
	MapID     uint16
	Triangles []collision.Triangle
	Dynamic   []mapgeometry.Dynamic
	MapSize   uint32
	Heights   []uint16
}

func digest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

// Bake writes one complete generation into a new directory, then publishes its
// manifest last. Sources are authored collision plus the native terrain fans.
type WalkMask struct {
	Size uint32
	Bits []byte
}

func Bake(g *mapgeometry.Geometry, id uint16, directory string, masks ...WalkMask) error {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	var obj bytes.Buffer
	writer := bufio.NewWriter(&obj)
	vertices := 0
	emit := func(t collision.Triangle) {
		if !t.Valid() {
			return
		}
		for _, p := range t {
			fmt.Fprintf(writer, "v %.9g %.9g %.9g\n", p[0], p[1], p[2])
		}
		fmt.Fprintf(writer, "f %d %d %d\n", vertices+1, vertices+2, vertices+3)
		vertices += 3
	}
	for _, t := range g.Triangles {
		emit(t)
	}
	for _, dynamic := range g.Dynamic {
		mapgeometry.Visit(dynamic.Meshes, dynamic.Base, emit)
	}
	g.Terrain(emit)
	if err := writer.Flush(); err != nil {
		return err
	}
	if vertices == 0 {
		return fmt.Errorf("map %d has no navigation input geometry", id)
	}

	temporary, err := os.MkdirTemp(directory, ".bake-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	navName := fmt.Sprintf("%d.gxnav", id)
	colName := fmt.Sprintf("%d.gxcol", id)
	input := filepath.Join(temporary, "source.obj")
	if err = os.WriteFile(input, obj.Bytes(), 0600); err != nil {
		return err
	}
	cellSize := float32(.25)
	if g.HasTerrain() {
		cellSize = .4
	}
	baker := os.Getenv("GX_RECAST_BAKER")
	if baker == "" {
		baker = "gx_recast_baker"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	arguments := []string{input, filepath.Join(temporary, navName), fmt.Sprint(cellSize)}
	if len(masks) > 0 && masks[0].Size > 0 {
		mask := masks[0]
		if uint64(len(mask.Bits)) != uint64(mask.Size)*uint64(mask.Size)/8 {
			return fmt.Errorf("invalid walk mask")
		}
		raw := make([]byte, 4+len(mask.Bits))
		binary.LittleEndian.PutUint32(raw, mask.Size)
		copy(raw[4:], mask.Bits)
		file := filepath.Join(temporary, "walk.mask")
		if err = os.WriteFile(file, raw, 0600); err != nil {
			return err
		}
		arguments = append(arguments, file)
	}
	var output []byte
	tileSize := 128
	for _, candidate := range []int{128, 256, 512} {
		tileSize = candidate
		command := exec.CommandContext(ctx, baker, arguments...)
		for _, entry := range os.Environ() {
			if !strings.HasPrefix(entry, "GX_RECAST_TILE_SIZE=") {
				command.Env = append(command.Env, entry)
			}
		}
		command.Env = append(command.Env, fmt.Sprintf("GX_RECAST_TILE_SIZE=%d", candidate))
		output, err = command.CombinedOutput()
		if err == nil {
			break
		}
		if !bytes.Contains(output, []byte("tile polygon capacity exceeded")) {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("map %d Recast bake: %w: %s", id, err, output)
	}
	polygons, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil || polygons < 0 {
		return fmt.Errorf("invalid baker polygon count: %s", output)
	}
	f, err := os.Create(filepath.Join(temporary, colName))
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(f)
	err = gob.NewEncoder(gz).Encode(physicsFile{FormatVersion, id, g.Triangles, g.Dynamic, g.MapSize, g.Heights})
	closeZip := gz.Close()
	closeFile := f.Close()
	if err != nil {
		return err
	}
	if closeZip != nil {
		return closeZip
	}
	if closeFile != nil {
		return closeFile
	}
	manifest := Manifest{TileSize: tileSize, CollisionRules: CollisionRulesVersion, Version: FormatVersion, MapID: id, Files: map[string]string{}, Sources: g.Sources, AgentHeight: 1.8, AgentRadius: .3, MaxClimb: .5, CellSize: cellSize, CellHeight: .1, Triangles: vertices / 3, Polygons: polygons}
	for _, name := range []string{navName, colName} {
		data, err := os.ReadFile(filepath.Join(temporary, name))
		if err != nil {
			return err
		}
		manifest.Files[name] = digest(data)
		if err = os.Rename(filepath.Join(temporary, name), filepath.Join(directory, name)); err != nil {
			return err
		}
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%d.manifest.json", id)
	if err = os.WriteFile(filepath.Join(temporary, name), append(raw, '\n'), 0644); err != nil {
		return err
	}
	if err = os.Rename(filepath.Join(temporary, name), filepath.Join(directory, name)); err != nil {
		return err
	}
	return WritePlayerPhysics(g, id, directory)
}
