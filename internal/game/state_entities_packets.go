package game

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func (s *State) applyEntitiesPacket(p []byte, ev Event) (Event, error) {
	switch ev.Op {
	case proto.CharacterEnteredMap:
		u, err := proto.DecodeUserEnter(p)
		if err != nil {
			return ev.packetError(err)
		}
		if u.ID == s.SelfID {
			s.Self.X, s.Self.Y, s.Self.Z = u.X, u.Y, u.Z
			s.Self.IsGM = u.Marker == 2
			if !s.Self.IsGM {
				s.GMTurbo = false
			}
			s.Guild.ID = u.GuildID
			if info, ok := s.Guild.Directory[u.GuildID]; ok {
				s.Guild.Remark, s.Guild.RemarkKnown = info.Remark, true
			}
			s.Phase = PhaseInWorld
		} else {
			e := s.Entities[u.ID]
			if e == nil {
				e = &Entity{Kind: KindUser, ID: u.ID}
				s.Entities[u.ID] = e
			}
			e.X, e.Y, e.Z, e.GuildID = u.X, u.Y, u.Z, u.GuildID
			e.IsGM = u.Marker == 2
			e.PositionResetSerial++
			e.ClimbMode = 0
		}
		ev.Text = fmt.Sprintf("user %d enters", u.ID)
	case proto.UserLeave, proto.MobLeave, proto.MapNpcLeave:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return ev.packetError(err)
		}
		delete(s.Entities, id)
		if s.Target == id {
			s.Target = 0
		}
	case proto.ChangeAppearance:
		m, err := proto.DecodeAppearanceChange(p)
		if err != nil {
			return ev.packetError(err)
		}
		if m.ID == s.SelfID {
			for slot, c := range s.Slots {
				if c.ID == m.ID {
					if c.Gender != m.Gender {
						s.QuestMapRevision++
					}
					c.Hair, c.Face, c.Height, c.Gender = m.Hair, m.Face, m.Height, m.Gender
					s.Slots[slot] = c
				}
			}
		}
		e := s.Entities[m.ID]
		if m.ID == s.SelfID {
			e = &s.Self
		}
		if e != nil && e.Shape != nil {
			shape := *e.Shape
			shape.Hair, shape.Face, shape.Height, shape.Gender = m.Hair, m.Face, m.Height, m.Gender
			e.Shape = &shape
		}
	case proto.CharacterEquipment:
		m, err := proto.DecodeEquipment(p)
		if err != nil {
			return ev.packetError(err)
		}
		e := s.Entities[m.ID]
		if m.ID == s.SelfID {
			e = &s.Self
		}
		if e != nil && e.Shape != nil && m.Slot < 8 {
			copyShape := *e.Shape
			copyShape.Equipment[m.Slot] = [3]uint8{m.Type, m.TypeID, m.Enchant}
			if m.Slot == 7 {
				copyShape.Cloak = m.Cloak
			}
			e.Shape = &copyShape
		}
	case proto.CharacterShape:
		shape, err := proto.DecodeUserShape(p)
		if err != nil {
			return ev.packetError(err)
		}
		e := &s.Self
		if shape.ID != s.SelfID {
			e = s.Entities[shape.ID]
			if e == nil {
				e = &Entity{Kind: KindUser, ID: shape.ID}
				s.Entities[shape.ID] = e
			}
		}
		e.Shape = &shape
		e.PvPKills, e.PvPKillsKnown = shape.Kills, true
		e.Dead = shape.Dead != 0
	case proto.CharacterMove:
		m, err := proto.DecodeUserMove(p)
		if err != nil {
			return ev.packetError(err)
		}
		e := &s.Self
		if m.ID != s.SelfID {
			if e = s.Entities[m.ID]; e == nil {
				e = &Entity{Kind: KindUser, ID: m.ID}
				s.Entities[m.ID] = e
			}
		}
		e.Motion, e.Dir, e.X, e.Y, e.Z = m.Motion, m.Dir, m.X, m.Y, m.Z
		e.MoveSerial++
	case proto.MobEnter, proto.GXMobEnter:
		m, err := proto.DecodeMobEnter(p)
		if err != nil {
			return ev.packetError(err)
		}
		s.Entities[m.ID] = &Entity{Kind: KindMob, ID: m.ID, TypeID: m.MobID, X: m.X, Y: m.Y, Z: m.Z, MobHeightKnown: m.HeightKnown, MobCountry: m.Country, MobCountryKnown: m.CountryKnown}
	case proto.MobMove, proto.GXMobMove:
		m, err := proto.DecodeMobMove(p)
		if err != nil {
			return ev.packetError(err)
		}
		if e := s.Entities[m.ID]; e != nil {
			e.Motion, e.X, e.Y, e.Z, e.MobHeightKnown = m.Motion, m.X, m.Y, m.Z, m.HeightKnown
			e.MobMoveDurationMS = m.DurationMS
			e.MoveSerial++
		}
	case proto.MapNpcEnter:
		n, err := proto.DecodeNpcEnter(p)
		if err != nil {
			return ev.packetError(err)
		}
		s.Entities[n.ID] = &Entity{Kind: KindNpc, Summoned: n.Summoned, ID: n.ID, NpcType: n.Type, TypeID: n.TypeID, X: n.X, Y: n.Y, Z: n.Z, Dir: n.Angle}
	case proto.MapNpcMove:
		n, err := proto.DecodeNpcMove(p)
		if err != nil {
			return ev.packetError(err)
		}
		if e := s.Entities[n.ID]; e != nil && e.Kind == KindNpc {
			e.Motion, e.X, e.Y, e.Z = n.Motion, n.X, n.Y, n.Z
			e.MoveSerial++
		}
	case proto.MapNpcAttackPlayer, proto.MapNpcAttackMob:
		hit, err := proto.DecodeNpcAttack(p)
		if err != nil {
			return ev.packetError(err)
		}
		if e := s.Entities[hit.Source]; e != nil && e.Kind == KindNpc {
			e.NpcAttackTarget = hit.Target
			e.NpcAttackSerial++
		}
	}
	return ev, nil
}
