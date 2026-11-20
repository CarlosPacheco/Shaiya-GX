package render

import "testing"

func TestCullEffectEmitterAndPreserveInstanceOrder(t *testing.T) {
	occlusion := NewTerrainOcclusion(32, 32)
	occlusion.Reset(Identity(), Identity())
	occlusion.covered = true
	for y := range occlusion.Height {
		for x := range occlusion.Width / 2 {
			occlusion.depth[y*occlusion.Width+x] = 2
		}
	}
	mesh := &Mesh{boundsValid: true, minimum: V3{-.05, -.05, 0}, maximum: V3{.05, .05, 0}}
	worlds, colors := make([]M4, 10), make([][4]float32, 10)
	for i := range worlds {
		worlds[i] = Identity()
		worlds[i][12] = -.45
		colors[i][0] = float32(i)
	}
	if got := occlusion.CullEffectInstances(mesh, worlds, colors); got != 0 {
		t.Fatalf("fully occluded emitter kept %d particles", got)
	}
	worlds[2][12], worlds[5][12], worlds[9][12] = .45, .03, .35
	if got := occlusion.CullEffectInstances(mesh, worlds, colors); got != 3 || colors[0][0] != 2 || colors[1][0] != 5 || colors[2][0] != 9 {
		t.Fatalf("visible particle order or bounds changed: count=%d colors=%v", got, colors[:3])
	}
	if occlusion.Culled != 0 {
		t.Fatal("worker changed shared camera state")
	}
}
