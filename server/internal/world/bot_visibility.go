package world

import "time"

const botEntrancesPerTick = 32

type botObserver struct {
	visible []bool
	pending []byte
}

func clearWorldBots() {
	if a := worldBots.arena; a != nil {
		for i := range a.bots {
			a.bots[i].wire.close()
		}
		for viewer, observer := range a.visible {
			if viewer.MapID() != a.bots[0].char.MapID {
				continue
			}
			for i, shown := range observer.visible {
				if shown {
					observer.add(buildUserLeave(a.bots[i].char.ID))
				}
			}
		}
		a.flush()
	}
	worldBots.arena = nil
}

func tickWorldBots(now time.Time) {
	worldBots.Lock()
	defer worldBots.Unlock()
	a := worldBots.arena
	if a == nil {
		return
	}
	if FindPlayerByID(a.owner) == nil {
		clearWorldBots()
		return
	}
	a.updateObservers(AllPlayers())
	for i := range a.bots {
		a.advance(i, now)
		if a.failed {
			clearWorldBots()
			return
		}
	}
	a.flush()
}

func (a *worldBotArena) updateObservers(players []*ConnectedPlayer) {
	for viewer := range a.visible {
		if FindPlayerByID(viewer.CharacterID) != viewer || viewer.MapID() != a.bots[0].char.MapID {
			delete(a.visible, viewer)
		}
	}
	for _, viewer := range players {
		if viewer.MapID() != a.bots[0].char.MapID {
			continue
		}
		observer := a.visible[viewer]
		if observer == nil {
			observer = &botObserver{visible: make([]bool, len(a.bots))}
			a.visible[viewer] = observer
		}
		a.updateObserver(observer, viewer)
	}
}

func (a *worldBotArena) updateObserver(observer *botObserver, viewer *ConnectedPlayer) {
	x, z := viewer.PositionXZ()
	added := 0
	for i := range a.bots {
		b := &a.bots[i]
		dx, dz := x-b.char.PosX, z-b.char.PosZ
		show := dx*dx+dz*dz <= BroadcastRadius*BroadcastRadius
		switch {
		case show && !observer.visible[i] && added < botEntrancesPerTick:
			observer.add(buildCharacterEnteredMapGuild(&b.char, false, false, 0))
			observer.add(buildCharacterShapeGuild(&b.char, b.faction, b.items, 0, ""))
			if b.char.CurrentHP == 0 {
				observer.add(buildCharacterDeathState(b.char.ID))
			}
			observer.visible[i] = true
			added++
		case !show && observer.visible[i]:
			observer.add(buildUserLeave(b.char.ID))
			observer.visible[i] = false
		}
	}
}

func (observer *botObserver) add(frame []byte) {
	observer.pending = appendWorldFrame(observer.pending, frame)
}

func (a *worldBotArena) emit(frame []byte, subject, target int) {
	for _, observer := range a.visible {
		if observer.visible[subject] && (target < 0 || observer.visible[target]) {
			observer.add(frame)
		}
	}
}

func (a *worldBotArena) flush() {
	for viewer, observer := range a.visible {
		if len(observer.pending) != 0 && viewer.MapID() == a.bots[0].char.MapID {
			// Each frame retains its own prefix and XOR. The batch occupies one
			// writer slot and cannot be interleaved with other network traffic.
			if err := viewer.out.write(observer.pending); err != nil {
				delete(a.visible, viewer)
			}
		}
		observer.pending = observer.pending[:0]
	}
}
