package game

import (
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"
)

// Every opcode with random bodies, alone and in long sequences on one
// accumulating state: a malformed or hostile server must never crash the client.
func TestApplyNeverPanicsOnRandomPackets(t *testing.T) {
	src, err := os.ReadFile("../proto/opcodes.go")
	if err != nil {
		t.Skip(err)
	}
	ops := map[uint16]bool{}
	for _, m := range regexp.MustCompile(`Opcode = (0x[0-9A-Fa-f]+)`).FindAllStringSubmatch(string(src), -1) {
		v, _ := strconv.ParseUint(m[1], 0, 16)
		ops[uint16(v)] = true
	}
	for _, f := range []string{"../proto/world.go", "../proto/vehicle.go"} {
		if b, err := os.ReadFile(f); err == nil {
			for _, m := range regexp.MustCompile(`Opcode = (0x[0-9A-Fa-f]+)`).FindAllStringSubmatch(string(b), -1) {
				v, _ := strconv.ParseUint(m[1], 0, 16)
				ops[uint16(v)] = true
			}
		}
	}
	rng := rand.New(rand.NewPCG(1, 2))
	panics := map[string]int{}
	start := time.Now()
	n := 0
	for op := range ops {
		for round := 0; round < 400; round++ {
			size := []int{0, 1, 2, 3, 4, 8, 16, 33, 64, 200, 700}[rng.IntN(11)]
			p := make([]byte, 2+size)
			binary.LittleEndian.PutUint16(p, op)
			for i := 2; i < len(p); i++ {
				switch rng.IntN(4) {
				case 0:
					p[i] = 0
				case 1:
					p[i] = 0xff
				default:
					p[i] = byte(rng.IntN(256))
				}
			}
			st := New()
			func() {
				defer func() {
					if r := recover(); r != nil {
						panics[fmt.Sprintf("op %#x: %v", op, r)]++
					}
				}()
				st.Apply(p)
				n++
			}()
		}
	}
	// Sequences on one accumulating state.
	list := make([]uint16, 0, len(ops))
	for op := range ops {
		list = append(list, op)
	}
	for seq := 0; seq < 600; seq++ {
		st := New()
		for step := 0; step < 400; step++ {
			op := list[rng.IntN(len(list))]
			size := []int{0, 1, 2, 4, 8, 16, 33, 64, 120}[rng.IntN(9)]
			p := make([]byte, 2+size)
			binary.LittleEndian.PutUint16(p, op)
			for i := 2; i < len(p); i++ {
				if rng.IntN(3) == 0 {
					p[i] = byte(rng.IntN(4))
				} else {
					p[i] = byte(rng.IntN(256))
				}
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						panics[fmt.Sprintf("seq op %#x: %v", op, r)]++
					}
				}()
				st.Apply(p)
				n++
			}()
		}
	}
	t.Logf("%d ops, %d applies in %v, %d distinct panics", len(ops), n, time.Since(start), len(panics))
	for k, v := range panics {
		t.Errorf("%s (x%d)", k, v)
	}
}
