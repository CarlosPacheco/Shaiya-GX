package scene

import (
	"fmt"
	"path"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// CharacterModels reports the native MLT indices with available meshes and
// textures for an appearance. Missing records leave their indices unset.
func (s *Assets) CharacterModels(a Appearance, part string) [256]bool {
	var available [256]bool
	folder := raceFolders[a.Race%4]
	table, err := s.mlt(folder, a.Prefix(), part)
	if err != nil {
		return available
	}
	for i, record := range table.Records[:min(len(table.Records), len(available))] {
		if record.Mesh < 0 || int(record.Mesh) >= len(table.Meshes) ||
			record.Texture < 0 || int(record.Texture) >= len(table.Textures) {
			continue
		}
		_, mesh := s.Arc.Lookup(fmt.Sprintf("Character/%s/3DC/%s", folder, path.Base(table.Meshes[record.Mesh])))
		_, texture := s.Arc.Lookup(fmt.Sprintf("Character/%s/DDS/%s", folder, DDSName(path.Base(table.Textures[record.Texture]))))
		available[i] = mesh && texture
	}
	return available
}

// PrepareCharacterItem loads and validates a weapon or shield attachment.
func (s *Assets) PrepareCharacterItem(typ, model uint8) bool {
	part, _, _, err := s.itemPart(typ, model)
	return err == nil && part.Tex != nil
}

// PrepareCharacterCloak validates and caches the native cloak for this outfit.
func (s *Assets) PrepareCharacterCloak(a Appearance) bool {
	_, err := s.cloakGeometry(a)
	return err == nil
}

// CloseCharacterResources releases an Assets cache dedicated to characters and
// their effects. Call on the GL thread after dropping its characters and effects.
func (s *Assets) CloseCharacterResources() {
	for _, part := range s.parts {
		part.mesh.Delete()
	}
	for _, part := range s.itemParts {
		part.Mesh.Delete()
	}
	for _, cloak := range s.cloaks {
		cloak.mesh.Delete()
	}
	for _, mesh := range []*render.Mesh{s.effectQuad, s.effectMirrorQuad, s.weaponQuad} {
		if mesh != nil {
			mesh.Delete()
		}
	}
	for _, texture := range s.textures {
		if texture != nil {
			texture.Delete()
		}
	}
	*s = Assets{}
}
