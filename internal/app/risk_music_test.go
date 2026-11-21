package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/audio"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestRiskMusicReplacesOnlyMapMusicWhileEnemiesAreNearby(t *testing.T) {
	root := t.TempDir()
	sound := filepath.Join(root, "Sound", "Music")
	if err := os.MkdirAll(sound, 0700); err != nil {
		t.Fatal(err)
	}
	for _, file := range riskMusic {
		if err := os.WriteFile(filepath.Join(sound, file), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	arc, err := archive.OpenDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := &App{arc: arc, state: game.New(), wv: &worldView{mapID: 1}}
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Race: 0}
	enemy := &game.Entity{ID: 2, Kind: game.KindUser, Shape: &proto.UserShape{Race: 2}}

	mix := func() []audio.LoopTarget {
		return a.mixRiskMusic([]audio.LoopTarget{
			{File: "town.ogg", Gain: .72, Music: true},
			{File: "wind.ogg", Gain: .86},
		}, a.enemyNearby())
	}
	if got := mix(); got[0].File != "town.ogg" {
		t.Fatalf("idle music = %+v", got)
	}
	a.state.Entities[2] = enemy
	got := mix()
	if len(got) != 2 || got[0].File != "wind.ogg" || got[1].File != a.mapAudio.riskTrack || !got[1].Music {
		t.Fatalf("alert mix = %+v", got)
	}
	if next := mix(); next[1] != got[1] {
		t.Fatal("battle music switched during the encounter")
	}
	a.gameMenu = &ui.GameMenu{Settings: ui.DefaultClientSettings()}
	a.gameMenu.Settings.RiskSensor = false
	if got := mix(); got[0].File != "town.ogg" || a.mapAudio.riskTrack != "" {
		t.Fatalf("disabled Risk Sensor replaced the map music: %+v", got)
	}
	a.gameMenu.Settings.RiskSensor = true
	if got := mix(); got[1].File != a.mapAudio.riskTrack || a.mapAudio.riskTrack == "" {
		t.Fatalf("enabled Risk Sensor did not resume battle music: %+v", got)
	}
	enemy.Dead = true
	if got := mix(); got[0].File != "town.ogg" || a.mapAudio.riskTrack != "" {
		t.Fatalf("music did not return after enemy died: %+v", got)
	}
	enemy.Dead = false
	a.wv.mapID = 40
	if got := mix(); got[0].File != "town.ogg" {
		t.Fatalf("exempt map played battle music: %+v", got)
	}
}
