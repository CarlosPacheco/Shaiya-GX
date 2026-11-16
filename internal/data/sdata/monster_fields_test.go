package sdata

import (
	"encoding/binary"
	"testing"
)

func TestMonsterNativeFields(t *testing.T) {
	// Two independent records catch a shifted tail/stride as well as offsets.
	raw := make([]byte, 4+2*(4+31))
	binary.LittleEndian.PutUint32(raw, 2)
	for i := range 2 {
		p := raw[8+i*35:]
		binary.LittleEndian.PutUint16(p, uint16(101+i))
		binary.LittleEndian.PutUint16(p[2:], 65)
		p[4], p[9], p[10], p[11] = 3, uint8(5+i), 8, 4
		binary.LittleEndian.PutUint32(p[5:], 123456)
		binary.LittleEndian.PutUint32(p[12:], 1200)
		p[16] = 2
		binary.LittleEndian.PutUint32(p[17:], 700)
		p[21] = 4
		copy(p[22:28], []byte{0, 1, 6, 2, 11, 3})
		p[28] = 1
		binary.LittleEndian.PutUint16(p[29:], 1139)
	}
	mobs, err := ParseMonsters(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(mobs) != 2 {
		t.Fatal(len(mobs))
	}
	for i, m := range mobs {
		if m.Model != uint16(101+i) || m.AI != 3 || m.HP != 123456 || m.ScaleCode != uint8(5+i) || m.Size != 8 || m.AttackPlus3 != 1 || m.QuestItemID != 1139 || m.AttackAnimations != [3]uint8{1, 2, 3} || m.ChaseTimeMS != 700 {
			t.Fatalf("bad field offsets: %+v", m)
		}
	}
}

func TestMonsterNativeVisualScale(t *testing.T) {
	for code, want := range []float32{.6, 1, 1.3, 1.6, 2, 5, .4, 1} {
		if got := (Monster{ScaleCode: uint8(code), Size: 8}).VisualScale(); got != want {
			t.Fatalf("code %d: %v, want %v", code, got, want)
		}
	}
}
