package render

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
)

func TestSkinGroupOrderBatchesCullAndIndexState(t *testing.T) {
	short, long := &SkinMesh{indexType: gl.UNSIGNED_SHORT}, &SkinMesh{indexType: gl.UNSIGNED_INT}
	tex := func(id uint32) *Texture { return &Texture{ID: id} }
	// Actor order: body, helmet (two-sided), body, cloak (two-sided), 32-bit body.
	groups := []SkinGroup{
		{Mesh: short, Texture: tex(3)}, {Mesh: short, Texture: tex(1), TwoSided: true},
		{Mesh: short, Texture: tex(2)}, {Mesh: short, Texture: tex(1), TwoSided: true},
		{Mesh: long, Texture: tex(5)},
	}
	order := skinGroupOrder(nil, groups)
	runs := 0
	for i, g := range order {
		if i == 0 || groups[g].TwoSided != groups[order[i-1]].TwoSided || groups[g].Mesh != groups[order[i-1]].Mesh && groups[g].Mesh.indexType != groups[order[i-1]].Mesh.indexType {
			runs++
		}
	}
	if runs != 3 {
		t.Fatalf("order %v gives %d state runs, want 3", order, runs)
	}
	if want := []int{2, 0, 1, 3, 4}; !equalInts(order, want) {
		t.Fatalf("order %v, want %v (stable within equal keys)", order, want)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
