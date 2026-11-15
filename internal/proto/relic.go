package proto

import (
	"fmt"
	"math"
)

// EP5.4 SHA256 8e5ef14a...bb3263 readers 0x594d20/0x594d80/0x594dd0 and
// 0x594e30: lists contain runtime IDs, not map/INI persistence keys.
// Owner states 0/1/2 are neutral/Light/Fury; portal states 100..103 are
// closed/Light/Fury/both. A list replaces the current map snapshot.
const (
	RelicList    Opcode = 0x2101
	RelicOwner   Opcode = 0x2102
	RelicNotice  Opcode = 0x2103
	PortalNotice Opcode = 0x2104
)

type Relic struct {
	ID    uint32
	Owner uint8
	X, Z  float32
}

func DecodeRelicList(p []byte) ([]Relic, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > 100 || r.Remaining() != n*13 {
		return nil, fmt.Errorf("invalid relic list length")
	}
	out := make([]Relic, 0, n)
	seen := make(map[uint32]bool, n)
	for range n {
		v := Relic{ID: r.U32(), Owner: r.U8(), X: r.F32(), Z: r.F32()}
		if seen[v.ID] || !validRelicOwner(v.Owner) || math.IsNaN(float64(v.X)) || math.IsNaN(float64(v.Z)) || math.IsInf(float64(v.X), 0) || math.IsInf(float64(v.Z), 0) {
			return nil, fmt.Errorf("invalid relic list record")
		}
		seen[v.ID] = true
		out = append(out, v)
	}
	if err := r.Err(); err != nil {
		return nil, err
	}
	if r.Remaining() != 0 {
		return nil, fmt.Errorf("relic list: %d trailing bytes", r.Remaining())
	}
	return out, nil
}

type RelicOwnerChange struct {
	ID    uint32
	Owner uint8
}

func DecodeRelicOwner(p []byte) (RelicOwnerChange, error) {
	r := NewReader(p)
	v := RelicOwnerChange{ID: r.U32(), Owner: r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 || !validRelicOwner(v.Owner) {
		return v, fmt.Errorf("relic owner: %d trailing bytes", r.Remaining())
	}
	return v, nil
}

type RelicDestroyed struct {
	MobID              uint16
	OldOwner, NewOwner uint8
}

func DecodeRelicNotice(p []byte) (RelicDestroyed, error) {
	r := NewReader(p)
	v := RelicDestroyed{MobID: r.U16(), OldOwner: r.U8(), NewOwner: r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 || v.OldOwner > 2 || v.NewOwner > 2 {
		return v, fmt.Errorf("relic notice: %d trailing bytes", r.Remaining())
	}
	return v, nil
}

func validRelicOwner(v uint8) bool { return v <= 2 || v >= 100 && v <= 103 }

type PortalOwnerNotice struct {
	PortalID           uint16
	OldOwner, NewOwner uint8
}

func DecodePortalNotice(p []byte) (PortalOwnerNotice, error) {
	r := NewReader(p)
	v := PortalOwnerNotice{r.U16(), r.U8(), r.U8()}
	if err := r.Err(); err != nil {
		return v, err
	}
	if r.Remaining() != 0 || v.OldOwner < 100 || v.OldOwner > 103 || v.NewOwner < 100 || v.NewOwner > 103 {
		return v, fmt.Errorf("invalid portal notice")
	}
	return v, nil
}
