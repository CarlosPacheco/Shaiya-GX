//go:build integration

package app

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func TestObeliskAgainstIsolatedGX(t *testing.T) {
	phase := os.Getenv("SHAIYA_TEST_OBELISK_PHASE")
	if phase == "" {
		t.Skip("requires disposable obelisk world fixture")
	}
	st, _, send, wait := testworld.Enter(t)
	wait(func() bool { return st.Self.IsGM && st.MapID == 0 && len(st.Relics) == 7 })
	send(proto.ChatPacket(proto.ChannelNormal, false, "/turbo"))
	wait(func() bool { return st.GMTurbo }) // Barrier after the complete entry/view snapshot.
	initialBless := st.Blessing.Count
	if expected := os.Getenv("SHAIYA_TEST_BLESSING_COUNT"); expected != "" {
		count, _ := strconv.ParseUint(expected, 10, 32)
		if initialBless != uint32(count) {
			t.Fatal("entry blessing snapshot", st.Blessing)
		}
	}
	owner := uint8(0)
	if phase == "restored" {
		owner = 1
	}
	if st.Relics[0].ID != 0 || st.Relics[0].Owner != owner {
		t.Fatalf("runtime identity/owner %+v", st.Relics)
	}
	relicMob := uint16(52)
	if owner == 1 {
		relicMob = 252
	}
	var id uint32
	wait(func() bool {
		for _, e := range st.Entities {
			if e.Kind == game.KindMob && e.TypeID == relicMob && !e.Dead {
				id = e.ID
				return true
			}
		}
		return false
	})
	if phase == "capture" {
		var guards []uint32
		for _, e := range st.Entities {
			if e.Kind == game.KindMob && e.ID != id {
				guards = append(guards, e.ID)
			}
		}
		if len(guards) == 0 {
			t.Fatal("no original guards visible")
		}
		send(proto.MobAttackPacket(id))
		wait(func() bool {
			return st.Relics[0].Owner == 1 && st.LastRelicNotice.NewOwner == 1 && st.LastRelicNotice.MobID == 52
		})
		if os.Getenv("SHAIYA_TEST_BLESSING_COUNT") != "" {
			wait(func() bool { return st.Blessing.Count == initialBless+1366 })
			t.Logf("altar shared blessing counter: %d -> %d", initialBless, st.Blessing.Count)
		}
		for _, guard := range guards {
			if st.Entities[guard] != nil {
				t.Fatal("dependent guard survived capture", guard)
			}
		}
		wait(func() bool { e := st.Entities[id]; return e != nil && !e.Dead && e.TypeID == 252 })
		if st.Details.Kills != 0 {
			t.Fatal("relic incorrectly granted a player kill")
		}
	} else if st.LastRelicNotice != (proto.RelicDestroyed{}) {
		t.Fatal("restart replayed a capture notice")
	}
	st.LastAttack = proto.AttackResult{}
	send(proto.MobAttackPacket(id)) // Own-country replacement must reject damage.
	deadline := time.Now().Add(1500 * time.Millisecond)
	wait(func() bool { return time.Now().After(deadline) })
	send(proto.StopAttackPacket())
	if st.LastAttack.Source == st.SelfID {
		t.Fatal("own-country relic accepted an attack")
	}
	send(proto.ChatPacket(proto.ChannelNormal, false, "/cmove 1"))
	wait(func() bool { return st.MapID == 1 })
	if len(st.Relics) != 0 {
		t.Fatal("map without relics retained stale icons")
	}
	send(proto.ChatPacket(proto.ChannelNormal, false, "/cmove 0"))
	wait(func() bool { return st.MapID == 0 && len(st.Relics) == 7 })
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
	t.Logf("obelisk %s: encrypted snapshot/capture/guards/owner variant/friendly protection/teleport state verified", phase)
}
