//go:build integration

package app

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/audio"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestNativeCharacterVoiceDispatch(t *testing.T) {
	if os.Getenv("SHAIYA_AUDIO_TEST") != "1" {
		t.Skip("requires local audio device and original sounds")
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	requested := map[string]int{}
	engine, err := audio.New(func(name string) ([]byte, error) {
		requested[strings.ToLower(name)]++
		return nil, errors.New("silent dispatch probe")
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	a := &App{arc: arc, sound: engine, soundErrors: map[string]bool{}, state: game.New(), wv: &worldView{},
		gameMenu: &ui.GameMenu{Settings: ui.DefaultClientSettings()}}
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Race: 2, Gender: 1}
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindUser, X: 5, Shape: &proto.UserShape{Race: 3}}
	death := proto.NewWriter(proto.CharacterDeath)
	death.U32(2)

	a.gameMenu.Settings.CharacterSound = false
	a.playCharacterVoice(1, voiceAttack)
	a.confirmedSound(death.Bytes())
	if len(requested) != 0 {
		t.Fatalf("voice played while disabled: %v", requested)
	}
	a.gameMenu.Settings.CharacterSound = true
	deadline := time.Now().Add(time.Second)
	for requested["sound/ch_ohatw003.ogg"] == 0 {
		a.playCharacterVoice(1, voiceAttack)
		if time.Now().After(deadline) {
			t.Fatalf("local attack voice not requested: %v", requested)
		}
		time.Sleep(time.Millisecond)
	}
	a.playCharacterDamageVoice(1, 1, 10)
	a.confirmedSound(death.Bytes())
	for _, file := range []string{"sound/ch_ohatw003.ogg", "sound/ch_viw_dam.ogg", "sound/ch_dem_die.ogg"} {
		if requested[file] != 1 {
			t.Fatalf("missing native cue %s: %v", file, requested)
		}
	}
	for race := range uint8(4) {
		for gender := range uint8(2) {
			for _, cue := range []characterVoice{voiceAttack, voiceDamage, voiceDeath} {
				name := characterVoiceFile(race, gender, cue)
				raw, err := arc.ReadFile("Sound/" + name)
				if err != nil {
					t.Fatal(err)
				}
				pcm, err := audio.DecodeOGG(raw)
				if err != nil || len(pcm) == 0 {
					t.Fatalf("invalid original voice %s: %v", name, err)
				}
			}
		}
	}
}
