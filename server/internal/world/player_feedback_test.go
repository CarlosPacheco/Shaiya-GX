package world

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestPlayerFeedbackReachesOnlyCurrentViewers(t *testing.T) {
	s, _ := potentialSession(t)
	const mapID = 61225
	s.char.MapID, s.char.Level, s.char.Class, s.char.SkillPoint = mapID, 22, 5, 5
	s.player.SetMap(mapID)
	if err := db.SaveCharacter(s.char); err != nil {
		t.Fatal(err)
	}
	observer := func(id uint32, faction uint8, known bool) (*ConnectedPlayer, <-chan []byte) {
		o, wire := gmWireSession(t, db.Character{ID: id, MapID: mapID}, false)
		p := RegisterPlayer(*o.char, nil, false, faction, o.out, o.logger)
		t.Cleanup(p.Unregister)
		if known {
			p.know(viewUser, s.char.ID)
		}
		return p, wire
	}
	_, light := observer(6122501, FactionLight, true)
	_, dark := observer(6122502, FactionDark, true)
	_, unknown := observer(6122503, FactionDark, false)
	assertCue := func(wire <-chan []byte, want uint8) {
		t.Helper()
		id, kind, err := proto.DecodePlayerFeedback(waitWorldPacket(t, wire, proto.GXPlayerFeedback))
		if err != nil || id != s.char.ID || kind != want {
			t.Fatalf("feedback = %d/%d, %v", id, kind, err)
		}
	}

	s.sendPlayerFeedback(proto.PlayerSpawnFeedback)
	assertCue(light, proto.PlayerSpawnFeedback)
	assertCue(dark, proto.PlayerSpawnFeedback)
	s.handleLearnSkill(proto.LearnSkillPacket(125, 1))
	assertCue(light, proto.PlayerSkillFeedback)
	assertCue(dark, proto.PlayerSkillFeedback)
	s.handleLearnSkill(proto.LearnSkillPacket(125, 1)) // Already learned.
	s.player.hidden.Store(true)
	s.sendPlayerFeedback(proto.PlayerSpawnFeedback)
	assertCue(light, proto.PlayerSpawnFeedback)
	timer := time.NewTimer(30 * time.Millisecond)
	defer timer.Stop()
	for {
		var p []byte
		select {
		case p = <-light:
		case p = <-dark:
		case p = <-unknown:
		case <-timer.C:
			return
		}
		if proto.Op(p) == proto.GXPlayerFeedback {
			t.Fatalf("unexpected player feedback: %x", p)
		}
	}
}
