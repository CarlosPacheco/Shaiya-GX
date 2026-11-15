//go:build integration

package app

import (
	"errors"
	"github.com/Spelunkern/Shaiya-GX/internal/audio"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNativeConsumableAndNewsSoundDispatch(t *testing.T) {
	if os.Getenv("SHAIYA_AUDIO_TEST") != "1" {
		t.Skip("requires local audio device and original sounds")
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	// Record actual Engine.Play requests, but fail before creating any audible
	// voice. Independently decode each original OGG to validate local resources.
	requested := map[string]int{}
	engine, err := audio.New(func(name string) ([]byte, error) {
		requested[strings.ToLower(name)]++
		return nil, errors.New("silent dispatch probe")
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()
	a := inputApp()
	a.arc = arc
	a.sound = engine
	a.soundErrors = map[string]bool{}
	a.state.SelfID = 1
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindUser, X: 10}
	a.state.LastItemUse = proto.ItemUseResult{ID: 2, Bag: 1}
	// The native ten-unit cutoff excludes its boundary.
	a.confirmedSound(proto.NewWriter(proto.UseItem).Bytes())
	if len(requested) != 0 {
		t.Fatal("out-of-range consumable was audible", requested)
	}
	a.state.Entities[2].X = 9
	deadline := time.Now().Add(time.Second)
	for requested["sound/ch_potion01.ogg"] == 0 {
		a.confirmedSound(proto.NewWriter(proto.UseItem).Bytes())
		if time.Now().After(deadline) {
			t.Fatal("consumable sound not requested", requested)
		}
		time.Sleep(time.Millisecond)
	}
	a.state.LastItemUse = proto.ItemUseResult{ID: 1, Bag: 0}
	a.confirmedSound(proto.NewWriter(proto.UseItem).Bytes())
	a.playNewsSound()
	for _, name := range []string{"ch_potion01.ogg", "ui_error001.ogg", "ui_news001.ogg"} {
		if requested["sound/"+name] != 1 {
			t.Fatal("missing or duplicate native sound", name, requested)
		}
		raw, e := arc.ReadFile("Sound/" + name)
		if e != nil {
			t.Fatal(e)
		}
		pcm, e := audio.DecodeOGG(raw)
		if e != nil || len(pcm) == 0 {
			t.Fatal("invalid original OGG", name, e)
		}
	}
}
