package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/logx"
)

func TestDeadCharacterRevivesAtBindWhenLeaving(t *testing.T) {
	old := mapConfig
	t.Cleanup(func() { mapConfig = old })
	log, err := logx.New("gx_revive_on_exit_test")
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	if err := loadMapProperties(log); err != nil {
		t.Fatal(err)
	}
	for _, dead := range []bool{true, false} { // flagged dead, or saved at 0 HP
		ch := &db.Character{Race: 3, MapID: 65500, Level: 20, Exp: 1000, CurrentHP: 0, PosX: 5, PosZ: 5}
		s := &Session{char: ch, dead: dead, deathLoss: deathExpPercent, log: log.Tagged("t")}
		s.reviveOnExit(s.exitDeathLoss())
		if s.dead || ch.CurrentHP == 0 {
			t.Fatalf("character stayed dead (flag=%v): dead=%v hp=%d", dead, s.dead, ch.CurrentHP)
		}
		if ch.MapID == 65500 && ch.PosX == 5 && ch.PosZ == 5 {
			t.Fatal("character was not moved to its bind point")
		}
	}
	alive := &db.Character{MapID: 65500, CurrentHP: 50, PosX: 5}
	s := &Session{char: alive, log: log.Tagged("t")}
	s.reviveOnExit(s.exitDeathLoss())
	if alive.MapID != 65500 || alive.CurrentHP != 50 || alive.PosX != 5 {
		t.Fatal("living character was changed on exit")
	}
}
