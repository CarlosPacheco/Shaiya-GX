package world

import (
	"path"
	"reflect"
	"strings"
	"testing"
)

func TestCollisionOnlyReadersMatchOriginalResources(t *testing.T) {
	arc := openArchive(t)
	for _, id := range []string{"1", "2", "3"} {
		raw, err := arc.ReadFile("world/" + id + ".wld")
		if err != nil {
			t.Fatal(err)
		}
		layout, err := ParseWLD(raw)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range []struct {
			directory string
			group     Group
		}{{"Entity/Building", layout.Buildings}, {"Entity/Shape", layout.Shapes}, {"Entity/Tree", layout.Trees}, {"Entity/Grass", layout.Grass}} {
			for _, name := range entry.group.Assets[:min(2, len(entry.group.Assets))] {
				file := entry.directory + "/" + name
				data, err := arc.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				full, err := ParseSMOD(data)
				if err != nil {
					t.Fatal(err)
				}
				only, err := ParseSMODCollisions(data)
				if err != nil {
					t.Fatal(err)
				}
				expected := full.Colliders
				vertices := 0
				for _, m := range expected {
					vertices += len(m.Vertices)
				}
				if vertices == 0 {
					total := 0
					for _, m := range full.Meshes {
						total += len(m.Vertices)
					}
					if total <= 65535 {
						expected = nil
						for _, m := range full.Meshes {
							c := CollisionMesh{Faces: m.Faces}
							for _, v := range m.Vertices {
								c.Vertices = append(c.Vertices, v.Pos)
							}
							if len(c.Vertices) > 0 {
								c.Bounds = Box{c.Vertices[0], c.Vertices[0]}
								for _, v := range c.Vertices {
									for a := range 3 {
										c.Bounds.Min[a] = min(c.Bounds.Min[a], v[a])
										c.Bounds.Max[a] = max(c.Bounds.Max[a], v[a])
									}
								}
							}
							expected = append(expected, c)
						}
					}
				}
				if len(only) != len(expected) || len(only) > 0 && !reflect.DeepEqual(only, expected) {
					t.Fatalf("collision-only reader changed %s", file)
				}
			}
		}
		for _, group := range []Group{layout.VaniA, layout.VaniB} {
			for _, name := range group.Assets[:min(2, len(group.Assets))] {
				file := "Entity/VAni/" + path.Base(strings.ReplaceAll(name, `\`, "/"))
				data, err := arc.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				full, err := ParseVANI(data)
				if err != nil {
					t.Fatal(err)
				}
				only, err := ParseVANICollisions(data)
				if err != nil {
					t.Fatal(err)
				}
				if len(only) > 0 {
					if len(only) != 1 || len(only[0].Faces) != 12 {
						t.Fatal("VANI-A did not construct the native box", file)
					}
					for _, v := range only[0].Vertices {
						for a := range 3 {
							if v[a] != full.FinalBounds.Min[a] && v[a] != full.FinalBounds.Max[a] {
								t.Fatal("VANI-A box differs from authored bounds", file, v, full.FinalBounds)
							}
						}
					}
				}
			}

		}
		if layout.Dungeon {
			file := "world/dungeon/" + layout.Layout
			data, err := arc.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			full, err := ParseDG(data)
			if err != nil {
				t.Fatal(err)
			}
			only, err := ParseDGCollisions(data)
			if err != nil {
				t.Fatal(err)
			}
			var expected []CollisionMesh
			full.Root.Walk(func(n *DGNode) {
				if len(n.Collider.Vertices) > 0 || len(n.Collider.Faces) > 0 {
					expected = append(expected, n.Collider)
				}
			})
			if !reflect.DeepEqual(only, expected) {
				t.Fatal("DG collision-only reader changed explicit geometry", file)
			}
		}
	}
}
func TestCollisionReadersRejectTruncatedData(t *testing.T) {
	for _, read := range []func([]byte) ([]CollisionMesh, error){ParseSMODCollisions, ParseVANICollisions, ParseDGCollisions} {
		for _, length := range []int{0, 5, 35} {
			if _, err := read(make([]byte, length)); err == nil {
				t.Fatalf("truncated %d-byte file accepted", length)
			}
		}
	}
}
