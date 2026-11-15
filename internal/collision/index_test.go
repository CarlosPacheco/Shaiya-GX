package collision

import (
	"math"
	"math/rand/v2"
	"reflect"
	"sync"
	"testing"
)

func TestIndexMatchesOriginalOrderedQueries(t *testing.T) {
	index, reference := NewIndex(), newReferenceIndex()
	add := func(triangle Triangle) { index.Add(triangle); reference.Add(triangle) }
	random := rand.New(rand.NewPCG(123, 456))
	for n := range 3000 {
		p := Vec{random.Float64()*80 - 40, random.Float64()*30 - 15, random.Float64()*80 - 40}
		size := .25 + random.Float64()*4
		if n%100 == 0 {
			size = 200
		} // Global references and duplicates across cells.
		add(Triangle{p, p.Add(Vec{size, 0, size}), p.Add(Vec{size, size, 0})})
	}
	add(Triangle{{1e10, 0, 0}, {1e10 + 1, 0, 1}, {1e10 + 1, 1, 0}})
	// Distinct IDs with identical geometry must still be visited separately.
	add(index.triangles[0])
	queries := []Box{
		{Vec{-16, -16, -16}, Vec{16, 16, 16}},
		{Vec{0, 0, 0}, Vec{0, 0, 0}},
		{Vec{-10000, -10000, -10000}, Vec{10000, 10000, 10000}}, // Large fallback.
		{Vec{math.Inf(-1), math.Inf(-1), math.Inf(-1)}, Vec{math.Inf(1), math.Inf(1), math.Inf(1)}},
	}
	for range 200 {
		p := Vec{random.Float64()*100 - 50, random.Float64()*40 - 20, random.Float64()*100 - 50}
		queries = append(queries, Box{p, p.Add(Vec{3, 2, 3})})
	}
	for _, query := range queries {
		var got, want []Triangle
		index.Visit(query, func(triangle Triangle) { got = append(got, triangle) })
		reference.Visit(query, func(triangle Triangle) { want = append(want, triangle) })
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("query %v changed triangle order or results: got %d, want %d", query, len(got), len(want))
		}
	}
	// Prepared indexes share no query scratch, including queries over many cells.
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for _, query := range queries[:4] {
				index.Visit(query, func(Triangle) {})
			}
		})
	}
	wg.Wait()
}

func TestIndexPartialBlocksAndInclusiveBounds(t *testing.T) {
	for _, count := range []int{1, 31, 32, 33, 63, 64, 65} {
		index := NewIndex()
		for n := range count {
			y := float64(n)
			index.Add(Triangle{{0, y, 0}, {1, y, 1}, {1, y, 0}})
		}
		for n := range count {
			query := Box{Vec{1, float64(n), 1}, Vec{1, float64(n), 1}}
			hits := 0
			index.Visit(query, func(Triangle) { hits++ })
			if hits != 1 {
				t.Fatalf("count=%d triangle=%d: %d boundary hits", count, n, hits)
			}
		}
	}
	var empty *Index
	empty.Visit(Box{}, func(Triangle) { t.Fatal("nil index visited") })
}

func denseObstacleIndexes() (*Index, *referenceIndex) {
	index, reference := NewIndex(), newReferenceIndex()
	add := func(triangle Triangle) { index.Add(triangle); reference.Add(triangle) }
	// Dense, spatially grouped mesh references share a cell but most are far
	// above the player's sweep, as with a detailed building or dungeon mesh.
	for y := range 128 {
		for n := range 64 {
			p := Vec{float64(n%8) * 1.5, float64(y) + 10, float64(n/8) * 1.5}
			add(Triangle{p, p.Add(Vec{1, 0, 1}), p.Add(Vec{1, 0, 0})})
		}
	}
	for _, triangle := range BlockerBox(Vec{8, 0, 4}, Vec{9, 4, 12}, 0) {
		add(triangle)
	}
	return index, reference
}

func TestIndexHeldObstacleMovementMatchesOriginal(t *testing.T) {
	index, reference := denseObstacleIndexes()
	for _, velocity := range []Vec{{.08, 0, 0}, {.08, 0, .03}, {20, 0, 0}, {0, -1, 0}} {
		got, want := Vec{7, 1.5, 8}, Vec{7, 1.5, 8}
		for frame := range 360 {
			p, hit := Move(got, velocity, index.Visit)
			expected, expectedHit := Move(want, velocity, reference.Visit)
			if p != expected || hit != expectedHit {
				t.Fatalf("frame %d velocity %v: %v/%v != %v/%v", frame, velocity, p, hit, expected, expectedHit)
			}
			got, want = p, expected
		}
	}
}

func BenchmarkIndexHeldObstacle(b *testing.B) {
	index, reference := denseObstacleIndexes()
	for _, variant := range []struct {
		name  string
		visit Visit
	}{{"original", reference.Visit}, {"blocks32", index.Visit}} {
		b.Run(variant.name, func(b *testing.B) {
			position := Vec{7.65, 1.5, 8}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				position, _ = Move(position, Vec{.08, 0, 0}, variant.visit)
			}
		})
	}
}
