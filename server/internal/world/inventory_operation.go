package world

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"slices"
	"time"
)

// inventoryOperation is an application unit of work. It stages character,
// equipment and packets while the session lock excludes other mutations.
// No database write or player publication occurs until its guarded commit.
type inventoryOperation struct {
	packets []operationPacket
	effects []func()
	player  *ConnectedPlayer
	err     error
}
type operationPacket struct {
	frame []byte
	label string
	view  bool
}

func (op *inventoryOperation) packet(frame []byte, label string, view bool) {
	if len(op.packets) >= 256 {
		op.err = fmt.Errorf("inventory publication budget exceeded")
		return
	}
	op.packets = append(op.packets, operationPacket{append([]byte(nil), frame...), label, view})
}
func (s *Session) inventoryOperation(run func()) bool {
	return s.inventoryOperationWithCommit(run, nil)
}
func (s *Session) inventoryOperationWithCommit(run func(), commit func(db.CharacterChange) error) (ok bool) {
	if s.char == nil {
		return false
	}
	if s.operation != nil {
		run()
		return s.operation.err == nil
	}
	original := s.char
	before := *original
	after := before
	items, buffs := s.items, s.buffs
	elements, movement := s.elements, s.movement
	speedKnown, speeds := s.speedKnown, s.lastSpeeds
	player := s.player
	op := &inventoryOperation{player: player}
	s.char = &after
	s.items = slices.Clone(items)
	s.buffs = slices.Clone(buffs)
	s.player = nil
	s.operation = op
	committed := false
	defer func() {
		s.operation = nil
		s.player = player
		s.char = original
		if !committed {
			s.items = items
			s.buffs = buffs
			s.elements = elements
			s.movement = movement
			s.speedKnown = speedKnown
			s.lastSpeeds = speeds
		}
	}()
	run()
	if op.err != nil {
		s.logErr("inventory plan", op.err)
		return false
	}
	if commit != nil || before != after || !slices.Equal(items, s.items) {
		change := db.CharacterChange{Before: before, After: after, ItemsBefore: items, ItemsAfter: s.items}
		var err error
		if commit != nil {
			err = commit(change)
		} else if before == after {
			err = db.CommitItems(before.ID, items, s.items)
		} else {
			err = db.CommitGameplay([]db.CharacterChange{change}, nil)
		}
		if err != nil {
			s.logErr("inventory transaction", err)
			return false
		}
	}
	committed = true
	*original = after
	s.char = original
	s.player = player
	s.operation = nil
	if player != nil {
		player.UpdateItems(s.items)
		s.syncPlayer()
		s.syncVisibility()
		s.publishPlayerEffects()
		player.mu.Lock()
		player.speeds, player.speedKnown = s.lastSpeeds, s.speedKnown
		player.mu.Unlock()
	}
	for _, packet := range op.packets {
		if packet.view {
			s.sendToViewers(packet.frame, packet.label)
		} else {
			s.send(packet.frame, packet.label)
		}
	}
	for _, effect := range op.effects {
		effect()
	}
	s.partyPools()
	return true
}
func (s *Session) saveInventoryItem(it *db.Item) error {
	if s.operation != nil {
		return nil
	}
	return db.SaveCharacterItem(s.char.ID, it)
}
func (s *Session) addInventoryItem(it *db.Item) error {
	if s.operation == nil {
		return db.AddCharacterItem(s.char.ID, it)
	}
	if it.UID == 0 {
		it.UID = db.NewItemUID()
	}
	if it.MakeTime.IsZero() {
		it.MakeTime = time.Now()
	}
	return nil
}

func (s *Session) operationPlayer() *ConnectedPlayer {
	if s.operation != nil {
		return s.operation.player
	}
	return s.player
}
func (s *Session) guildPublication(g *guild, frame []byte, label string, except *ConnectedPlayer) {
	if s.operation == nil {
		g.send(frame, label, except)
		return
	}
	if len(s.operation.effects)+len(s.operation.packets) >= 256 {
		s.operation.err = fmt.Errorf("inventory publication budget exceeded")
		return
	}
	s.operation.effects = append(s.operation.effects, func() { g.send(frame, label, except) })
}
