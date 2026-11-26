package scene

import (
	"strings"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
)

// UseInterfaceLanguage selects an optional artwork overlay before loading UI
// textures. Widgets retain their original texture handles until client restart.
func (s *Assets) UseInterfaceLanguage(code string) {
	s.interfaceOverlay = ""
	if code != "" {
		dir := "Languages/interface/interface_" + code
		if _, ok := s.Arc.LookupDir(dir); ok {
			s.interfaceOverlay = dir
		}
	}
}

func (s *Assets) interfaceTextureOverride(file string) string {
	if s.interfaceOverlay == "" {
		return ""
	}
	dir, name, _ := strings.Cut(strings.ReplaceAll(file, `\`, "/"), "/")
	if name == "" || !strings.EqualFold(dir, "interface") {
		return ""
	}
	return s.interfaceOverlay + "/" + name
}

func (s *Assets) HasTexture(file string) bool {
	if _, ok := s.Arc.Lookup(file); ok {
		return true
	}
	if override := s.interfaceTextureOverride(file); override != "" {
		_, ok := s.Arc.Lookup(override)
		return ok
	}
	return false
}

// ReadTexture decodes a single localized replacement when available, then
// falls back to the normal resource for missing or unreadable replacements.
func (s *Assets) ReadTexture(file string) (img.Texture, error) {
	if override := s.interfaceTextureOverride(file); override != "" {
		if raw, err := s.Arc.ReadFile(override); err == nil {
			if decoded, err := img.DecodeResource(file, raw); err == nil {
				return decoded, nil
			}
		}
	}
	raw, err := s.Arc.ReadFile(file)
	if err != nil {
		return img.Texture{}, err
	}
	return img.DecodeResource(file, raw)
}
