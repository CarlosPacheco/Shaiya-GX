package mapnav

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
)

const PlayerPhysicsVersion = 1

type playerPhysicsFile struct {
	Version int
	MapID   uint16
	Physics physicsFile
	Water   bool
}

// WritePlayerPhysics publishes a derived collision resource without replacing
// existing mob files. Older running worlds can continue using their generation.
func WritePlayerPhysics(g *mapgeometry.Geometry, id uint16, directory string) error {
	manifestName := filepath.Join(directory, fmt.Sprintf("%d.manifest.json", id))
	raw, err := os.ReadFile(manifestName)
	if err != nil {
		return err
	}
	var manifest Manifest
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	if manifest.MapID != id || manifest.Version != FormatVersion || manifest.CollisionRules != CollisionRulesVersion {
		return fmt.Errorf("incompatible map generation %d: run a full navigation bake", id)
	}
	for _, extension := range []string{"gxnav", "gxcol"} {
		name := fmt.Sprintf("%d.%s", id, extension)
		if len(manifest.Files[name]) != 64 {
			return fmt.Errorf("missing navigation checksum for %s: run a full navigation bake", name)
		}
	}
	if len(g.Sources) != len(manifest.Sources) {
		return fmt.Errorf("map %d source set changed: regenerate navigation", id)
	}
	for source, old := range manifest.Sources {
		if g.Sources[source] != old {
			return fmt.Errorf("map %d source %s changed: regenerate navigation instead of updating physics only", id, source)
		}
	}
	// Index contains query acceleration only; rebuild it from immutable triangles.
	exported := physicsFile{FormatVersion, id, g.Triangles, g.Dynamic, g.MapSize, g.Heights}
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	if err = gob.NewEncoder(gz).Encode(playerPhysicsFile{PlayerPhysicsVersion, id, exported, g.Water}); err != nil {
		return err
	}
	if err = gz.Close(); err != nil {
		return err
	}
	name := fmt.Sprintf("%d.gxphysics", id)
	f, err := os.CreateTemp(directory, ".player-physics-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(buffer.Bytes()); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), filepath.Join(directory, name)); err != nil {
		return err
	}
	manifest.PlayerPhysics = PlayerPhysicsVersion
	manifest.Files[name] = digest(buffer.Bytes())
	raw, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	f, err = os.CreateTemp(directory, ".player-manifest-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(append(raw, '\n')); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), manifestName)
}

func loadPlayerPhysics(data []byte, id uint16) (*mapgeometry.Geometry, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	var stored playerPhysicsFile
	if err = gob.NewDecoder(io.LimitReader(gz, 512<<20)).Decode(&stored); err != nil {
		return nil, err
	}
	if stored.Version != PlayerPhysicsVersion || stored.MapID != id {
		return nil, fmt.Errorf("invalid player physics for map %d", id)
	}
	p := stored.Physics
	if p.Version != FormatVersion || p.MapID != id {
		return nil, fmt.Errorf("invalid player collision payload")
	}
	g := &mapgeometry.Geometry{Triangles: p.Triangles, Dynamic: p.Dynamic, MapSize: p.MapSize, Heights: p.Heights, Water: stored.Water}
	if g.MapSize > 0 && !g.HasTerrain() {
		return nil, fmt.Errorf("invalid player terrain samples")
	}
	g.Reindex()
	return g, nil
}

func (m *Map) PlayerGeometry() (*mapgeometry.Geometry, bool) { return m.geometry, m.playerPhysics }
