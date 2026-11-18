package world

import (
	"fmt"
)

// forcedTeleport is a GM's /asummon: where the user is taken, at once.
type forcedTeleport struct {
	mapID uint16
	pos   [3]float32
}

func (p *ConnectedPlayer) queueForcedTeleport(f forcedTeleport) {
	p.mailMu.Lock()
	p.forced = &f
	p.mailMu.Unlock()
}

func (p *ConnectedPlayer) takeForcedTeleport() *forcedTeleport {
	p.mailMu.Lock()
	defer p.mailMu.Unlock()
	f := p.forced
	p.forced = nil
	return f
}

// tickForcedTeleport carries out a GM's /asummon on this user: no consumable,
// no cast and no confirmation.
func (s *Session) tickForcedTeleport() {
	if s.player == nil || s.char == nil {
		return
	}
	f := s.player.takeForcedTeleport()
	if f == nil || s.dead || s.char.CurrentHP == 0 {
		return
	}
	pos := f.pos
	if g, err := s.playerGeometry(f.mapID); err == nil {
		if dest, ok := g.Spawn(pos); ok {
			pos = dest
		}
	}
	s.cancelMove()
	s.cast, s.autoAttack = nil, false
	s.gmForced = true
	s.applyTeleport(f.mapID, pos[0], pos[1], pos[2], "GM summon", true)
	s.gmForced = false
}

// gmTarget finds the character a GM command names.
func (s *Session) gmTarget(name string) (*ConnectedPlayer, error) {
	p := FindPlayerByName(name)
	if p == nil || p == s.player {
		return nil, fmt.Errorf("%s is not online", name)
	}
	if !p.IsAlive() {
		return nil, fmt.Errorf("%s is dead", p.CharacterName)
	}
	return p, nil
}

// gmGoTo is /amove: the movement rune without the item or the cast, to where
// another character is, on any map.
func (s *Session) gmGoTo(name string) error {
	if s.dead || s.char.CurrentHP == 0 {
		return fmt.Errorf("you are dead")
	}
	p, err := s.gmTarget(name)
	if err != nil {
		return err
	}
	ch := p.Snapshot()
	pos := [3]float32{ch.PosX, ch.PosY, ch.PosZ}
	if g, e := s.playerGeometry(ch.MapID); e == nil {
		if dest, ok := g.Spawn(pos); ok {
			pos = dest
		}
	}
	s.cancelMove()
	s.cast, s.autoAttack = nil, false
	s.applyTeleport(ch.MapID, pos[0], pos[1], pos[2], "GM move", true)
	s.gmReply("GM: moved to " + p.CharacterName + ".")
	return nil
}

// gmBring is /asummon: the summon rune without the item, the cast or the
// confirmation; the character is taken to the GM at once.
func (s *Session) gmBring(name string) error {
	p, err := s.gmTarget(name)
	if err != nil {
		return err
	}
	if tradeFor(p.CharacterID) != nil {
		return fmt.Errorf("%s is trading", p.CharacterName)
	}
	p.queueForcedTeleport(forcedTeleport{mapID: s.char.MapID, pos: [3]float32{s.char.PosX, s.char.PosY, s.char.PosZ}})
	s.gmReply("GM: summoned " + p.CharacterName + ".")
	return nil
}
