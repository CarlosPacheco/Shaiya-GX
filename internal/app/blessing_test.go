package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	gametext "github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestBlessingTooltipUsesCumulativeLocalizedTypes(t *testing.T) {
	a := &App{state: game.New(), blessingLoaded: true, msg: gametext.SysMsg{5136: "HP recovery localized", 5137: "SP recovery localized", 5150: "Linking localized"}}
	a.blessingTiers = []sdata.BlessingTier{{Country: 0, Kills: 150, Effects: [6]sdata.BlessingEffect{{Type: 11, Value: 5}}}, {Country: 0, Kills: 300, Effects: [6]sdata.BlessingEffect{{Type: 24, Value: 1}, {Type: 10, Value: 5}}}, {Country: 1, Kills: 150, Effects: [6]sdata.BlessingEffect{{Type: 4, Value: 10}}}}
	a.state.Blessing = game.BlessingState{Country: 0, Count: 300, Known: true}
	rows := a.blessingLines()
	if len(rows) != 3 || rows[0] != "HP recovery localized" || rows[1] != "SP recovery localized" || rows[2] != "Linking localized" {
		t.Fatal(rows)
	}
	a.state.Blessing.Count = 149
	if len(a.blessingLines()) != 0 {
		t.Fatal("future benefits displayed")
	}
}

func TestBlessingActivationOnlyOnConfirmedMaximumBroadcast(t *testing.T) {
	a := &App{state: game.New(), blessingLoaded: true, blessingMax: [2]uint32{12288, 12288}, clock: 10}
	a.state.Blessing = game.BlessingState{Country: 0, Count: 12288, RemainingMS: 120000, Known: true}
	a.confirmedBlessing(proto.BlessInit)
	a.confirmedBlessing(proto.CountryKillCount)
	if a.state.Blessing.RemainingMS != 120000 || a.blessingBannerUntil != 0 {
		t.Fatal("reconnect replayed activation")
	}
	a.state.Blessing.Count = 0
	a.state.Blessing.RemainingMS = 0
	a.confirmedBlessing(proto.CountryKillCount)
	if a.state.Blessing.RemainingMS != 0 {
		t.Fatal("reset activated blessing")
	}
	a.state.Blessing.Count = 12288
	a.confirmedBlessing(proto.CountryKillCount)
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
		a := &App{state: game.New(), blessingLoaded: true, blessingMax: [2]uint32{12288, 12288}, clock: 10}
		a.state.Faction.Faction = faction
		a.state.Blessing = game.BlessingState{Country: 0, Count: 12288, Known: true}
		a.confirmedBlessing(proto.CountryKillCount)
		if (a.blessingBannerUntil == 15) != (faction == 0) {
			t.Fatal("activation escaped winning faction", faction)
		}
	}
}
