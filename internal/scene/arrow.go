package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// ArrowModel loads the official loose 3DO/DDS pair once, shared by all shots.
func (s *Assets) ArrowModel() (*Part, error) {
	if p := s.itemParts["physical:arrow"]; p != nil {
		return p, nil
	}
	raw, err := s.Arc.ReadFile("Item/3DO/arrow.3DO")
	if err != nil {
		return nil, err
	}
	obj, err := world.Parse3DO(raw)
	if err != nil {
		return nil, err
	}
	vertices := make([]render.Vertex, len(obj.Vertices))
	for i, v := range obj.Vertices {
		vertices[i] = render.Vertex{Pos: render.V3(v.Pos), Normal: render.V3(v.Normal), UV: v.UV}
	}
	indices := make([]uint32, 0, len(obj.Faces)*3)
	for _, f := range obj.Faces {
		for _, i := range f {
			indices = append(indices, uint32(i))
		}
	}
	p := &Part{Mesh: render.NewMesh(vertices, indices), Tex: s.modelTexture("Item/DDS/arrow.dds"), Alpha: true}
	if s.itemParts == nil {
		s.itemParts = map[string]*Part{}
	}
	s.itemParts["physical:arrow"] = p
	return p, nil
}
