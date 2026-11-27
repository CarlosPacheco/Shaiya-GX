package world

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"testing"
)

// The vectors under testdata/svnav were produced by executing the original
// ps_game.exe instructions (CMap::EnableLine 0042ab80/0042ade0, IsBlockSafe
// 0041b420, AStar::FindPath 0048eda0) over synthetic masks.

func readSvnavVectors(t *testing.T, name string, each func(kind string, f []string)) {
	t.Helper()
	file, err := os.Open("testdata/svnav/" + name)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	s := bufio.NewScanner(file)
	s.Buffer(make([]byte, 1<<20), 1<<24)
	for s.Scan() {
		line := s.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		f := strings.Fields(line)
		each(f[0], f[1:])
	}
	if err := s.Err(); err != nil {
		t.Fatal(err)
	}
}

func vectorInt(t *testing.T, v string) int32 {
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	return int32(n)
}

func vectorGrid(t *testing.T, f []string) (int32, svGrid) {
	var mask []byte
	if f[2] != "-" {
		var err error
		if mask, err = hex.DecodeString(f[2]); err != nil {
			t.Fatal(err)
		}
	}
	return vectorInt(t, f[0]), svGrid{vectorInt(t, f[1]), mask}
}

func TestSvmapEnableLineMatchesOriginal(t *testing.T) {
	maps := map[int32]svGrid{}
	cases := 0
	readSvnavVectors(t, "lines.tsv", func(kind string, f []string) {
		switch kind {
		case "N":
			id, g := vectorGrid(t, f)
			maps[id] = g
		case "L":
			g := maps[vectorInt(t, f[0])]
			start := svTile{vectorInt(t, f[1]), vectorInt(t, f[2])}
			end := svTile{vectorInt(t, f[3]), vectorInt(t, f[4])}
			previous := svTile{-777, 888}
			ok := g.enableLine(start, end, &previous)
			var b bytes.Buffer
			b.WriteByte(boolByte(ok))
			binary.Write(&b, binary.LittleEndian, previous.x)
			binary.Write(&b, binary.LittleEndian, previous.z)
			b.WriteByte(boolByte(g.enableLine(start, end, nil)))
			b.WriteByte(boolByte(g.blocked(start)))
			b.WriteByte(boolByte(g.blocked(end)))
			if got := hex.EncodeToString(b.Bytes()); got != f[5] {
				t.Fatalf("case %d %v->%v: got %s want %s", cases, start, end, got, f[5])
			}
			cases++
		}
	})
	if cases != 4096 {
		t.Fatalf("covered %d line cases", cases)
	}
}

func TestSvmapAStarMatchesOriginal(t *testing.T) {
	maps := map[int32]svGrid{}
	cases := 0
	a := &svAStar{}
	readSvnavVectors(t, "astar.tsv", func(kind string, f []string) {
		switch kind {
		case "N":
			id, g := vectorGrid(t, f)
			maps[id] = g
		case "F":
			g := maps[vectorInt(t, f[0])]
			from := svTile{vectorInt(t, f[1]), vectorInt(t, f[2])}
			to := svTile{vectorInt(t, f[3]), vectorInt(t, f[4])}
			want, err := hex.DecodeString(f[5])
			if err != nil {
				t.Fatal(err)
			}
			path, ok := a.path(g, from, to, nil)
			if ok != (want[0] == 1) {
				t.Fatalf("case %d %v->%v: result %v", cases, from, to, ok)
			}
			// Skip result, found, last and the scan bounds, then the cursor.
			r := want[3+16:]
			if r[0] == 1 {
				r = r[9:]
			} else {
				r = r[1:]
			}
			n := int(binary.LittleEndian.Uint32(r))
			var chain []svTile
			for i := range n {
				chain = append(chain, svTile{int32(binary.LittleEndian.Uint32(r[4+8*i:])), int32(binary.LittleEndian.Uint32(r[8+8*i:]))})
			}
			if len(chain) > 0 {
				chain = chain[1:] // the original chain includes the start
			}
			if len(chain) != len(path) {
				t.Fatalf("case %d %v->%v: path %v want %v", cases, from, to, path, chain)
			}
			for i := range chain {
				if chain[i] != path[i] {
					t.Fatalf("case %d %v->%v: path %v want %v", cases, from, to, path, chain)
				}
			}
			cases++
		}
	})
	if cases != 1536 {
		t.Fatalf("covered %d A* cases", cases)
	}
}

func TestSvNextRoutesAroundWall(t *testing.T) {
	// 32x32 mask with a wall at x=16 except a gap at z=10.
	const size = 32
	mask := make([]byte, size*size/8)
	for z := range int32(size) {
		if z == 10 {
			continue
		}
		i := z*size + 16
		mask[i/8] |= 1 << (i % 8)
	}
	g := svGrid{size, mask}
	var r svRoute
	pos, goal := [3]float32{4.5, 0, 4.5}, [3]float32{28.5, 0, 20.5}
	for range 200 {
		next, ok := svNext(g, &r, pos, goal, 2)
		if !ok {
			t.Fatalf("stuck at %v", pos)
		}
		if g.blocked(tileOf(next[0], next[2])) {
			t.Fatalf("entered wall at %v", next)
		}
		pos = next
		if DistanceXZ(pos[0], pos[2], goal[0], goal[2]) < .02 {
			return
		}
	}
	t.Fatalf("did not arrive, at %v", pos)
}
