package scene

import (
	"cmp"
	"fmt"
	"math"
	"path"
	"slices"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type vaniPart struct {
	mesh        *render.Mesh
	frameMeshes []*render.Mesh
	texture     *render.Texture
	frames      [][]world.Vertex
	faces       []world.Face
	vertices    []render.Vertex
}

func loadVani(a *Assets, file string) (*vaniAsset, error) {
	data, err := a.Arc.ReadFile(file)
	if err != nil {
		return nil, err
	}
	parsed, err := world.ParseVANI(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return uploadVani(a, file, parsed)
}
func uploadVani(a *Assets, file string, parsed *world.VANI) (*vaniAsset, error) {
	asset := &vaniAsset{frame: -1, count: parsed.FrameCount, frameMilliseconds: parsed.FrameMilliseconds, center: render.V3(parsed.Center), radius: parsed.Radius}
	asset.unlit = strings.EqualFold(path.Base(file), "Login_effect_water01a.VANI")
	for _, m := range parsed.Meshes {
		part := vaniPart{texture: a.FindTexture(m.Texture)}
		if part.texture != nil {
			part.texture.SetWrap(true)
		}
		indices := make([]uint32, 0, len(m.Faces)*3)
		for _, face := range m.Faces {
			indices = append(indices, uint32(face[0]), uint32(face[1]), uint32(face[2]))
		}
		if len(m.Frames) > 0 {
			vertices := make([]render.Vertex, len(m.Frames[0]))
			part.frameMeshes = make([]*render.Mesh, len(m.Frames))
			for f, frame := range m.Frames {
				if len(vertices) != len(frame) {
					vertices = make([]render.Vertex, len(frame))
				}
				for j, src := range frame {
					vertices[j] = render.Vertex{Pos: render.V3(src.Pos), Normal: render.V3(src.Normal), UV: src.UV}
				}
				part.frameMeshes[f] = render.NewMesh(vertices, indices)
			}
			part.mesh = part.frameMeshes[0]
		}
		asset.parts = append(asset.parts, part)
	}
	return asset, nil
}

type vaniAsset struct {
	visibleWorlds     []render.M4
	parts             []vaniPart
	frameMilliseconds int32
	unlit             bool
	worlds            []render.M4
	frame             int
	count             int
	center            render.V3
	radius            float32
}

func (v *vaniAsset) draw(s *render.Scene3D, elapsed float64) {
	if len(v.worlds) == 0 || v.count == 0 {
		return
	}
	v.updatePose(elapsed)
	alpha := render.NativeWorldAlphaRef(false)
	if len(v.worlds) == 1 {
		for _, part := range v.parts {
			s.DrawWorldModel(part.mesh, part.texture, v.worlds[0], alpha)
		}
		return
	}
	s.SetInstances(v.worlds)
	for _, part := range v.parts {
		s.DrawWorldInstanced(part.mesh, part.texture, alpha)
	}
}

func (v *vaniAsset) updatePose(elapsed float64) {
	frame := v.frameAt(elapsed)
	if frame != v.frame {
		for i := range v.parts {
			p := &v.parts[i]
			if frame < len(p.frameMeshes) {
				p.mesh = p.frameMeshes[frame]
				continue
			}
			if frame < len(p.frames) {
				for j, src := range p.frames[frame] {
					p.vertices[j] = render.Vertex{Pos: render.V3(src.Pos), Normal: render.V3(src.Normal), UV: src.UV}
				}
				p.mesh.UpdateVertices(p.vertices)
			}
		}
		v.frame = frame
	}
}

func (v *vaniAsset) close() {
	for i := range v.parts {
		p := &v.parts[i]
		if len(p.frameMeshes) > 0 {
			for _, m := range p.frameMeshes {
				m.Delete()
			}
			p.frameMeshes = nil
			p.mesh = nil
		} else {
			p.mesh.Delete()
		}
	}
}

type vaniPlacement struct {
	asset  *vaniAsset
	matrix render.M4
	depth  float32
}

// Both WLD groups share one animated vertex stream per asset. Game.exe draws
// group A at 0x448a72 and group B with source-alpha blending at 0x44a769.
func (w *World) addVanis(a *Assets) {
	assets := make(map[string]*vaniAsset)
	for groupIndex, group := range []world.Group{w.WLD.VaniA, w.WLD.VaniB} {
		for _, inst := range group.Instances {
			if inst.Asset < 0 || int(inst.Asset) >= len(group.Assets) {
				continue
			}
			name := path.Base(strings.ReplaceAll(group.Assets[inst.Asset], `\`, "/"))
			file := "Entity/VAni/" + name
			key := strings.ToLower(file)
			asset, exists := assets[key]
			if !exists {
				asset, _ = loadVani(a, file)
				if asset != nil {
					w.vanis = append(w.vanis, asset)
				}
				assets[key] = asset
			}
			if asset != nil {
				matrix := render.Basis(render.V3(inst.Pos), render.V3(inst.Fwd), render.V3(inst.Up))
				if groupIndex == 0 {
					asset.worlds = append(asset.worlds, matrix)
				} else {
					w.blendedVani = append(w.blendedVani, vaniPlacement{asset: asset, matrix: matrix})
				}
			}
		}
	}
}

func (w *World) queueBlendedVani(s *render.Scene3D, asset *vaniAsset, matrix render.M4, center render.V3) {
	depth := center[0]*s.View[2] + center[1]*s.View[6] + center[2]*s.View[10]
	w.visibleBlendedVani = append(w.visibleBlendedVani, vaniPlacement{asset: asset, matrix: matrix, depth: depth})
}

// DrawTranslucentVANI runs after opaque geometry and actors so that transparent
// texels reveal the correct surface. All visible placements are sorted strictly
// back-to-front and submitted through vaniBatch so mixed foliage assets share
// indirect draw calls without changing compositing order.
func (w *World) DrawTranslucentVANI(s *render.Scene3D) {
	if s.Shadow {
		return
	}
	if len(w.blendedVani) != 0 {
		f := render.ViewFrustum(s.View, s.VisibilityProjection())
		for _, placement := range w.blendedVani {
			asset, matrix := placement.asset, placement.matrix
			center := transformPoint(asset.center, matrix)
			if asset.visible(s, f, matrix, center) {
				w.queueBlendedVani(s, asset, matrix, center)
			}
		}
	}
	visible := w.visibleBlendedVani
	if len(visible) == 0 {
		return
	}
	slices.SortStableFunc(visible, func(a, b vaniPlacement) int { return cmp.Compare(b.depth, a.depth) })
	batch := &w.vaniBatch
	batch.ResetTransforms()
	batch.Transforms = batch.Transforms[:0]
	w.vaniBlendIDs = w.vaniBlendIDs[:0]
	for i, placement := range visible {
		batch.Transforms = append(batch.Transforms, placement.matrix)
		w.vaniBlendIDs = append(w.vaniBlendIDs, uint32(i))
	}
	firstPlacement := batch.Placements(w.vaniBlendIDs)
	for i := 0; i < len(visible); {
		asset := visible[i].asset
		asset.updatePose(w.elapsed)
		runLen := 1
		if len(asset.parts) == 1 {
			for i+runLen < len(visible) && visible[i+runLen].asset == asset {
				runLen++
			}
		}
		for _, part := range asset.parts {
			batch.AddTranslucent(part.mesh, part.texture, !asset.unlit, firstPlacement+i, runLen)
		}
		i += runLen
	}
	batch.DrawTranslucent(s)
	clear(visible)
	w.visibleBlendedVani = visible[:0]
}

func (v *vaniAsset) frameAt(elapsed float64) int {
	if v.count <= 0 {
		return 0
	}
	interval := v.frameMilliseconds
	if interval <= 0 {
		interval = 33
	}
	return int(math.Mod(math.Floor(max(0, elapsed)*1000/float64(interval)), float64(v.count)))
}

// Eager dungeon placements keep all authored transforms and filter into scratch
// storage; animation bounds encompass every frame, not just the current pose.
func (v *vaniAsset) drawCulled(s *render.Scene3D, elapsed float64, f render.Frustum) {
	if v.radius <= 0 {
		v.draw(s, elapsed)
		return
	}
	all := v.worlds
	v.visibleWorlds = v.visibleWorlds[:0]
	for _, matrix := range all {
		if v.visible(s, f, matrix, transformPoint(v.center, matrix)) {
			v.visibleWorlds = append(v.visibleWorlds, matrix)
		}
	}
	v.worlds = v.visibleWorlds
	v.draw(s, elapsed)
	v.worlds = all
}

func (v *vaniAsset) visible(s *render.Scene3D, f render.Frustum, matrix render.M4, center render.V3) bool {
	radius := v.radius * sphereScale(matrix)
	return radius <= 0 || f.SphereVisible(center, radius) && render.ClipSphereVisible(s.ClipPlane, center, radius) && !s.OccludedSphere(center, radius)
}
