package mapnav

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const MobDungeonDirectory = "mob-dungeons"

type Catalog struct {
	Version int      `json:"version"`
	Maps    []uint16 `json:"maps"`
}

func ReadCatalog(directory string) (Catalog, error) {
	var catalog Catalog
	raw, err := os.ReadFile(filepath.Join(directory, "catalog.json"))
	if err != nil {
		return catalog, err
	}
	if err = json.Unmarshal(raw, &catalog); err != nil {
		return catalog, err
	}
	if catalog.Version != FormatVersion || len(catalog.Maps) == 0 {
		return catalog, fmt.Errorf("invalid navigation catalog")
	}
	seen := map[uint16]bool{}
	for _, id := range catalog.Maps {
		if seen[id] {
			return catalog, fmt.Errorf("duplicate navigation map %d", id)
		}
		seen[id] = true
	}
	return catalog, nil
}

// WriteCatalog publishes the completed map set. A listed missing map is an
// installation error, never permission to silently revert to straight lines.
func WriteCatalog(directory string, ids []uint16) error {
	if len(ids) == 0 {
		return fmt.Errorf("cannot publish an empty navigation catalog")
	}
	ids = slices.Clone(ids)
	slices.Sort(ids)
	for index, id := range ids {
		if index > 0 && id == ids[index-1] {
			return fmt.Errorf("duplicate navigation map %d", id)
		}
		raw, err := os.ReadFile(filepath.Join(directory, fmt.Sprintf("%d.manifest.json", id)))
		if err != nil {
			return err
		}
		var m Manifest
		if err = json.Unmarshal(raw, &m); err != nil {
			return err
		}
		if m.MapID != id || m.Version != FormatVersion || m.CollisionRules != CollisionRulesVersion {
			return fmt.Errorf("invalid map generation %d", id)
		}
	}
	raw, err := json.MarshalIndent(Catalog{FormatVersion, ids}, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(directory, ".catalog-")
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
	return os.Rename(f.Name(), filepath.Join(directory, "catalog.json"))
}

// ValidateCatalog checks installation completeness before any map is populated.
// File contents are checksum-verified when a map is decoded.
func ValidateCatalog(directory string) (bool, error) {
	catalog, err := ReadCatalog(directory)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, id := range catalog.Maps {
		raw, err := os.ReadFile(filepath.Join(directory, fmt.Sprintf("%d.manifest.json", id)))
		if err != nil {
			return true, err
		}
		var manifest Manifest
		if err = json.Unmarshal(raw, &manifest); err != nil {
			return true, err
		}
		if manifest.MapID != id || manifest.Version != FormatVersion {
			return true, fmt.Errorf("invalid navigation manifest for map %d", id)
		}
		if manifest.CollisionRules != CollisionRulesVersion {
			return true, fmt.Errorf("map %d uses collision rules %d, expected %d; rebuild the geometry catalog", id, manifest.CollisionRules, CollisionRulesVersion)
		}
		if manifest.PlayerPhysics != PlayerPhysicsVersion {
			return true, fmt.Errorf("map %d is missing player physics; run gx_mapbuild -physics-only", id)
		}
		for _, extension := range []string{"gxnav", "gxcol", "gxphysics"} {
			name := fmt.Sprintf("%d.%s", id, extension)
			if len(manifest.Files[name]) != 64 {
				return true, fmt.Errorf("missing navigation checksum for %s", name)
			}
			info, err := os.Stat(filepath.Join(directory, name))
			if err != nil {
				return true, err
			}
			if !info.Mode().IsRegular() || info.Size() == 0 {
				return true, fmt.Errorf("invalid navigation file %s", name)
			}
		}
	}
	return true, nil
}
