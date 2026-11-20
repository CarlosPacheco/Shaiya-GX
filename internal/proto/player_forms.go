package proto

import "fmt"

const (
	GXFormAppearance Opcode = 0x7F21
	GXEvolutionBar   Opcode = 0x7F22
	GXEvolutionCast  Opcode = 0x7F23
)

type PlayerFormKind uint8

const (
	FormNone PlayerFormKind = iota
	FormMonsterCopy
	FormDisguise
	FormEvolution
	FormDegeneration
)

func SkillForm(effect uint8, detail, change uint16) PlayerFormKind {
	if detail != 90 {
		return FormNone
	}
	if effect == 3 && change == 101 {
		return FormDegeneration
	}
	if effect != 4 {
		return FormNone
	}
	switch change {
	case 100:
		return FormDisguise
	case 102:
		return FormMonsterCopy
	case 10, 11, 12:
		return FormEvolution
	default:
		return FormNone
	}
}

type PlayerFormAppearance struct {
	User    uint32
	Kind    PlayerFormKind
	Skill   uint16
	Rank    uint8
	Shape   UserShape
	Attacks [2]FormAttack
}

type FormAttack struct {
	Skill uint16
	Slot  uint8
}

func DecodePlayerForm(p []byte) (v PlayerFormAppearance, err error) {
	r := NewReader(p)
	v.User, v.Kind = r.U32(), PlayerFormKind(r.U8())
	switch v.Kind {
	case FormNone:
		if r.Remaining() == 0 {
			return v, r.Err()
		}
	case FormEvolution:
		v.Skill, v.Rank = r.U16(), r.U8()
		for i := range v.Attacks {
			v.Attacks[i] = FormAttack{Skill: r.U16(), Slot: r.U8()}
		}
		if r.Remaining() == 0 {
			return v, r.Err()
		}
	case FormDisguise:
		if len(p) >= 9 && Op(p[7:]) == CharacterShape {
			v.Shape, err = DecodeUserShape(p[7:])
			if err == nil && v.Shape.ID != v.User {
				err = fmt.Errorf("invalid disguise character")
			}
			return v, err
		}
	}
	return v, fmt.Errorf("invalid player form")
}

type EvolutionAction struct {
	Skill              uint16
	Range              uint8
	CooldownMS, LeftMS uint32
}

type EvolutionBar struct {
	Generation uint32
	Actions    [2]EvolutionAction
}

func DecodeEvolutionBar(p []byte) (v EvolutionBar, err error) {
	if len(p) != 28 {
		return v, fmt.Errorf("invalid evolution bar length")
	}
	r := NewReader(p)
	v.Generation = r.U32()
	for i := range v.Actions {
		a := &v.Actions[i]
		a.Skill, a.Range, a.CooldownMS, a.LeftMS = r.U16(), r.U8(), r.U32(), r.U32()
	}
	return v, r.Err()
}

func EvolutionCastPacket(generation uint32, cell uint8, target uint32) []byte {
	w := NewWriter(GXEvolutionCast)
	w.U32(generation)
	w.U8(cell)
	w.U32(target)
	return w.Bytes()
}
