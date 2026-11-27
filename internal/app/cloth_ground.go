package app

import "github.com/Spelunkern/Shaiya-GX/internal/scene"

// installClothGround gives cloak collisions the terrain under each actor
// (native 0x410580). Maps without terrain keep the cloth unclamped.
func installClothGround(w *scene.World) {
	if !w.HasTerrain() {
		scene.SetClothGround(nil)
		return
	}
	scene.SetClothGround(w.Height)
}
