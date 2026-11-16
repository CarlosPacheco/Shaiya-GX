package sdata

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/element"
)

// Monster is an EP5 Monster.SData record (Parsec MonsterRecord).
// IDs are the zero-based record indices used by the server.
type Monster struct {
	Name                          string
	Model, Level                  uint16
	AI, ScaleCode                 uint8 // Day is the native client visual scale selector.
	HP                            uint32
	Size, Element                 uint8
	NormalTimeMS, ChaseTimeMS     uint32
	NormalStep, ChaseStep         uint8
	AttackTypes, AttackAnimations [3]uint8
	AttackPlus3                   uint8
	QuestItemID                   uint16
}

func ParseMonsters(raw []byte) ([]Monster, error) {
	b, err := Decrypt(raw)
	if err != nil {
		return nil, err
	}
	c := &cursor{b: b}
	n := c.count(35)
	out := make([]Monster, 0, n)
	for i := 0; i < n && c.err == nil; i++ {
		m := Monster{Name: c.str(), Model: c.u16(), Level: c.u16()}
		m.AI = c.u8()
		m.HP = c.u32()
		m.ScaleCode = c.u8()
		m.Size = c.u8()
		m.Element = element.MobElement(c.u8())
		m.NormalTimeMS, m.NormalStep = c.u32(), c.u8()
		m.ChaseTimeMS, m.ChaseStep = c.u32(), c.u8()
		for j := range 3 {
			m.AttackTypes[j], m.AttackAnimations[j] = c.u8(), c.u8()
		}
		m.AttackPlus3, m.QuestItemID = c.u8(), c.u16()
		out = append(out, m)
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.off != len(b) {
		return nil, fmt.Errorf("monster sdata: %d trailing bytes", len(b)-c.off)
	}
	return out, nil
}

// VisualScale is Game.exe 0x431770's Monster.SData Day selector.
// Size remains a separate combat radius, not a model scale.
func (m Monster) VisualScale() float32 {
	switch m.ScaleCode {
	case 0:
		return .6
	case 1:
		return 1
	case 2:
		return 1.3
	case 3:
		return 1.6
	case 4:
		return 2
	case 5:
		return 5
	case 6:
		return .4
	default:
		return 1
	}
}
