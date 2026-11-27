package render

import "github.com/Spelunkern/Shaiya-GX/internal/render/gl"

// Anisotropic filtering (GL_ARB/EXT_texture_filter_anisotropic) for the
// mipmapped world textures. Bindless handles freeze texture state, so each
// sampling view exists in a plain and an anisotropic variant and draws pick
// the one matching the current setting; the base texture is updated in place.
const (
	textureMaxAnisotropy    = 0x84fe
	maxTextureMaxAnisotropy = 0x84ff
	anisotropyTarget        = 16
)

var (
	anisotropySupported bool
	anisotropyMax       int32
	anisotropyLevel     int32 = 1
	worldTextures           = map[*Texture]struct{}{}
)

func configureAnisotropy(supported bool) {
	anisotropySupported, anisotropyMax, anisotropyLevel = supported, 1, 1
	if supported {
		var v float32
		gl.GetFloatv(maxTextureMaxAnisotropy, &v)
		anisotropyMax = max(1, min(anisotropyTarget, int32(v)))
	}
}

func registerAnisotropic(t *Texture) {
	worldTextures[t] = struct{}{}
	if anisotropySupported {
		gl.TextureParameteri(t.ID, textureMaxAnisotropy, anisotropyLevel)
	}
}

// SetAnisotropic turns 16x anisotropic filtering (or the driver maximum) on
// or off for every world texture, including ones loaded later.
func SetAnisotropic(on bool) {
	level := int32(1)
	if on && anisotropySupported {
		level = anisotropyMax
	}
	if level == anisotropyLevel {
		return
	}
	anisotropyLevel = level
	for t := range worldTextures {
		gl.TextureParameteri(t.ID, textureMaxAnisotropy, level)
	}
}
