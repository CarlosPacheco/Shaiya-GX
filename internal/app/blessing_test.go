package app

import (
	gametext "github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestBlessingTooltipUsesCumulativeLocalizedTypes(t *testing.T) {
	a := &App{state: game.New(), msg: gametext.SysMsg{5136: "HP recovery localized", 5137: "SP recovery localized", 5150: "Linking localized"}}
	a.state.Blessing = game.BlessingState{Country: 0, Count: 300, Maximum: 12288, Known: true, Bonuses: [27]uint16{9: 5, 10: 5, 23: 1}}
	rows := a.blessingLines()
	if len(rows) != 3 || rows[0] != "HP recovery localized" || rows[1] != "SP recovery localized" || rows[2] != "Linking localized" {
		t.Fatal(rows)
	}
	a.state.Blessing.Count = 149
	a.state.Blessing.Bonuses = [27]uint16{}
	if len(a.blessingLines()) != 0 {
		t.Fatal("future benefits displayed")
	}
}

func TestBlessingActivationOnlyOnConfirmedMaximumBroadcast(t *testing.T) {
	a := &App{state: game.New(), clock: 10}
	a.state.Blessing = game.BlessingState{Country: 0, Count: 12288, Maximum: 12288, Entry: true, RemainingMS: 120000, Known: true}
	a.confirmedBlessing(proto.GXBlessingSnapshot)
	a.confirmedBlessing(proto.CountryKillCount)
	a.state.Blessing.Entry = false
	if a.state.Blessing.RemainingMS != 120000 || a.blessingBannerUntil != 0 {
		t.Fatal("reconnect replayed activation")
	}
	a.state.Blessing.Count = 0
	a.state.Blessing.RemainingMS = 0
	a.confirmedBlessing(proto.GXBlessingSnapshot)
	if a.state.Blessing.RemainingMS != 0 {
		t.Fatal("reset activated blessing")
	}
	a.state.Blessing.Count = 12288
	a.state.Blessing.RemainingMS = 600000
	a.confirmedBlessing(proto.GXBlessingSnapshot)
	if a.state.Blessing.RemainingMS != 600000 || a.blessingBannerUntil != 15 {
		t.Fatal("missing native activation")
	}
	a.state.Blessing.Advance(2)
	a.clock = 12
	a.confirmedBlessing(proto.CountryKillCount)
	if a.state.Blessing.RemainingMS != 598000 || a.blessingBannerUntil != 15 {
		t.Fatal("duplicate broadcast extended/replayed blessing")
	}
	a.confirmedBlessing(proto.Logout)
	if a.blessingBannerUntil != 0 || a.blessingFeedbackActive {
		t.Fatal("stale activation after logout")
	}
}

func TestBlessingFeedbackIsLocalToEachWinningFactionMember(t *testing.T) {
	for _, faction := range []uint8{0, 0, 1} {
		a := &App{state: game.New(), clock: 10}
		a.state.Faction.Faction = faction
		a.state.Blessing = game.BlessingState{Country: 0, Count: 12288, Maximum: 12288, RemainingMS: 600000, Known: true}
		a.confirmedBlessing(proto.GXBlessingSnapshot)
		if (a.blessingBannerUntil == 15) != (faction == 0) {
			t.Fatal("activation escaped winning faction", faction)
		}
	}
}
