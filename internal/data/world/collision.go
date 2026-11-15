package world

import "github.com/Spelunkern/Shaiya-GX/internal/collision"

// ParseSMODCollisions follows 0x5aeec0/0x5ae590: authored collision wins;
// absent collision is built from display triangles when the model fits uint16.
// Group exclusions (Grass/VANI-B) belong to the world dispatcher, not the file.
func ParseSMODCollisions(data []byte) ([]CollisionMesh, error) {
	return parseSMODCollisionPolicy(data, false)
}

// ParseTreeCollisions uses the original group mode 3 (0x5ae540).
func ParseTreeCollisions(data []byte) ([]CollisionMesh, error) {
	return parseSMODCollisionPolicy(data, true)
}

func parseSMODCollisionPolicy(data []byte, tree bool) ([]CollisionMesh, error) {
	r := &reader{b: data}
	r.take(40)
	var display []CollisionMesh
	total := 0
	for i, n := 0, r.count(12); i < n && r.err == nil; i++ {
		r.lstring()
		nv := r.count(36)
		total += nv
		m := CollisionMesh{Vertices: make([]Vec3, nv)}
		for j := range m.Vertices {
			m.Vertices[j] = r.vec3()
			r.take(24)
		}
		m.Faces = r.faces()
		if len(m.Vertices) > 0 {
			m.Bounds = Box{Min: m.Vertices[0], Max: m.Vertices[0]}
			for _, v := range m.Vertices {
				for a := range 3 {
					m.Bounds.Min[a] = min(m.Bounds.Min[a], v[a])
					m.Bounds.Max[a] = max(m.Bounds.Max[a], v[a])
				}
			}
		}
		display = append(display, m)
	}
	bounds := r.box()
	out := make([]CollisionMesh, 0)
	for i, n := 0, r.count(8); i < n && r.err == nil; i++ {
		out = append(out, r.collisionMesh())
	}
	if r.err != nil {
		return nil, r.err
	}
	vertices := 0
	for _, m := range out {
		vertices += len(m.Vertices)
	}
	if vertices == 0 && tree && len(display) > 0 {
		return collisionBox(bounds), nil
	}
	if vertices == 0 && total <= 65535 {
		return display, nil
	}
	return out, nil
}

// ParseVANICollisions follows VANI-A group mode 3 (0x4466c0/0x5adbc0).
// The original uses the authored collision bounds, not pose-zero display faces.
func ParseVANICollisions(data []byte) ([]CollisionMesh, error) {
	r := &reader{b: data}
	r.take(40)
	meshes := r.count(12)
	frames := r.count(0)
	r.i32()
	if frames <= 0 || frames > 65536 {
		return nil, ErrTruncated
	}
	for i := 0; i < meshes && r.err == nil; i++ {
		r.lstring()
		r.take(r.count(6) * 6)
		r.take(r.count(frames*36) * frames * 36)
	}
	bounds := r.box()
	r.i32()
	if r.err != nil {
		return nil, r.err
	}
	if meshes == 0 {
		return nil, nil
	}
	return collisionBox(bounds), nil
}

func collisionBox(b Box) []CollisionMesh {
	for a := range 3 {
		if b.Min[a] > b.Max[a] {
			return nil
		}
	}
	m := CollisionMesh{Bounds: b}
	for _, t := range collision.BlockerBox(collision.Vec{float64(b.Min[0]), float64(b.Min[1]), float64(b.Min[2])}, collision.Vec{float64(b.Max[0]), float64(b.Max[1]), float64(b.Max[2])}, 0) {
		i := uint16(len(m.Vertices))
		for _, p := range t {
			m.Vertices = append(m.Vertices, Vec3{float32(p[0]), float32(p[1]), float32(p[2])})
		}
		m.Faces = append(m.Faces, Face{i, i + 1, i + 2})
	}
	return []CollisionMesh{m}
}

// ParseDGCollisions skips lightmapped display buffers while retaining every
// explicit DG node collision mesh. Node boxes are broad phase, never solids.
func ParseDGCollisions(data []byte) ([]CollisionMesh, error) {
	r := &reader{b: data}
	r.box()
	r.names()
	r.i32()
	var meshes []CollisionMesh
	var node func(int)
	node = func(depth int) {
		if depth > 32 {
			r.err = errDepth
			return
		}
		r.vec3()
		r.box()
		r.box()
		for i, n := 0, r.count(8); i < n && r.err == nil; i++ {
			r.i32()
			for j, m := 0, r.count(12); j < m && r.err == nil; j++ {
				r.i32()
				r.take(r.count(44) * 44)
				r.take(r.count(6) * 6)
			}
		}
		if r.i32() == 1 {
			meshes = append(meshes, r.collisionMesh())
		}
		for i := 0; i < 8 && r.err == nil; i++ {
			if r.i32() > 0 {
				node(depth + 1)
			}
		}
	}
	if r.i32() > 0 {
		node(0)
	}
	return meshes, r.err
}
