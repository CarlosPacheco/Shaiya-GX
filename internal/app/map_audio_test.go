package app

import (
	"fmt"
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/audio"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func TestMapAudioFadeAndDeduplication(t *testing.T) {
	w := &world.WLD{MusicAssets: []string{"track.wav"}, MusicZones: []world.MusicZone{{Box: world.Box{Min: world.Vec3{10, -1, 10}, Max: world.Vec3{0, 1, 0}}, Asset: 0}}, SoundAssets: []string{"birds.wav"}, Sounds: []world.SoundInstance{{Asset: 0, Center: world.Vec3{5, 0, 5}, Radius: 10}, {Asset: 0, Center: world.Vec3{20, 0, 5}, Radius: 10}, {Asset: 5, Radius: 10}}}
	got := mapAudioTargets(w, render.V3{5, 0, 5})
	if len(got) != 2 || !got[0].Music || got[0].Gain != .72 || got[1].Gain != .86 {
		t.Fatalf("inside box/strongest emitter: %+v", got)
	}
	got = mapAudioTargets(w, render.V3{34, 0, 5})
	if len(got) != 2 || got[0].Gain >= .72 || got[0].Gain <= 0 {
		t.Fatalf("outside-box fade missing: %+v", got)
	}
	if len(mapAudioTargets(w, render.V3{200, 0, 200})) != 0 {
		t.Fatal("distant tracks retained")
	}
	for i := 0; i < 40; i++ {
		w.SoundAssets = append(w.SoundAssets, fmt.Sprint(i))
		w.Sounds = append(w.Sounds, world.SoundInstance{Asset: int32(i + 1), Center: world.Vec3{5, 0, 5}, Radius: 10})
	}
	if len(mapAudioTargets(w, render.V3{5, 0, 5})) != 25 {
		t.Fatal("ambient cap lost")
	}
}

func TestTerrainFootstepLayer(t *testing.T) {
	w := &world.WLD{MapSize: 4, TextureMap: []byte{0, 1, 0, 1, 0, 1, 0, 1, 0}, Layers: []world.TerrainLayer{{Sound: "grass.wav"}, {Sound: "stone.wav"}}}
	if terrainWalkSound(w, 2, 0) != "stone.wav" || terrainWalkSound(w, 0, 0) != "grass.wav" || terrainWalkSound(w, -1, 0) != "" {
		t.Fatal("incorrect terrain sample")
	}
	w.TextureMap[0] = 9
	if terrainWalkSound(w, 0, 0) != "" {
		t.Fatal("invalid layer accepted")
	}
}

func TestRealMapAudioCatalog(t *testing.T) {
	root := os.Getenv("SHAIYA_DATA")
	if root == "" {
		t.Skip("requires assets")
	}
	arc, err := archive.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := &App{arc: arc}
	decoded := map[string]bool{}
	for _, id := range []string{"1", "2", "19", "select_a", "select_b", "Login"} {
		b, err := arc.ReadFile("world/" + id + ".wld")
		if err != nil {
			t.Fatal(err)
		}
		w, err := world.ParseWLD(b)
		if err != nil {
			t.Fatal(err)
		}
		if (id == "1" || id == "2") && len(w.MusicZones) == 0 {
			t.Fatal(id, "has no parsed music")
		}
		missing := 0
		for _, name := range w.MusicAssets {
			file := a.audioPath(name, true)
			if _, ok := arc.Lookup(file); !ok {
				t.Errorf("%s music missing: %s", id, file)
			}
		}
		for _, name := range w.SoundAssets {
			if name == "" {
				continue
			}
			file := a.audioPath(name, false)
			if _, ok := arc.Lookup(file); !ok {
				missing++
				t.Logf("map %s absent ambient asset: %q", id, name)
				continue
			}
			if !decoded[file] && len(decoded) < 3 {
				raw, err := arc.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				pcm, err := audio.DecodeOGG(raw)
				if err != nil || len(pcm) == 0 {
					t.Fatal(file, err)
				}
				decoded[file] = true
			}
		}
		for _, layer := range w.Layers {
			if layer.Sound != "" {
				if _, ok := arc.Lookup(a.audioPath(layer.Sound, false)); !ok {
					t.Errorf("%s footstep missing: %s", id, layer.Sound)
				}
			}
		}
		t.Logf("map %s: %d music zones, %d emitters; missing authored ambience=%d", id, len(w.MusicZones), len(w.Sounds), missing)
	}
	if len(decoded) == 0 {
		t.Fatal("no real ambience decoded")
	}
}

func TestRealMapAudioRegions(t *testing.T) {
	root := os.Getenv("SHAIYA_DATA")
	if root == "" {
		t.Skip("assets")
	}
	arc, err := archive.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	b, err := arc.ReadFile("world/1.wld")
	if err != nil {
		t.Fatal(err)
	}
	w, err := world.ParseWLD(b)
	if err != nil {
		t.Fatal(err)
	}
	targets := mapAudioTargets(w, render.V3{803.786, .876, 1434.47})
	if len(targets) == 0 {
		t.Fatal("river should have audible ambience")
	}
	music := false
	for _, v := range mapAudioTargets(w, render.V3{520, 78, 1762}) {
		music = music || v.Music
	}
	if !music {
		t.Fatal("town music zone must be audible")
	}
	b, err = arc.ReadFile("world/19.wld")
	if err != nil {
		t.Fatal(err)
	}
	w, err = world.ParseWLD(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.MusicAssets) != 0 || len(w.MusicZones) != 0 {
		t.Fatal("map19 audio fixture changed")
	}
	targets = mapAudioTargets(w, render.V3{635.946, 7.56, 1841.597})
	if len(targets) != 1 || targets[0].Music || targets[0].File != "bg_League of light_01.ogg" || targets[0].Gain <= 0 {
		t.Fatalf("map19 continuous field ambience missing: %+v", targets)
	}
}

// A dungeon is stone. Dry ground at any height uses its own terrain layer.
func TestFootstepFileSurfaces(t *testing.T) {
	land := &scene.World{WLD: &world.WLD{MapSize: 4, TextureMap: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0}, Layers: []world.TerrainLayer{{Sound: "ch_walkg002.ogg"}}}}
	for range 20 {
		if got := footstepFile(land, 1, 5, 1, false); got != "ch_walkg001.ogg" && got != "ch_walkg002.ogg" && got != "ch_walkg003.ogg" {
			t.Fatalf("terrain step %q", got)
		}
		for _, y := range []float32{.5, -.5} {
			if got := footstepFile(land, 1, y, 1, false); got < "ch_walkg001.ogg" || got > "ch_walkg003.ogg" {
				t.Fatalf("dry terrain below 0.9 played %q", got)
			}
		}
		if got := footstepFile(&scene.World{WLD: &world.WLD{Dungeon: true}}, 1, 5, 1, false); got < "ch_walks001.ogg" || got > "ch_walks003.ogg" {
			t.Fatalf("dungeon step %q", got)
		}
	}
	if got := footstepFile(land, 1, 5, 1, true); got < "ch_walks001.ogg" || got > "ch_walks003.ogg" {
		t.Fatal("a step on an object must use stone, not the terrain layer", got)
	}
	if got := mountStepFile(land, 0, 1, 5, 1, true); got[:len("run_horse_st0")] != "run_horse_st0" {
		t.Fatal("a hoof on an object must use stone", got)
	}
	if footstepFile(nil, 0, 0, 0, false) != "" || footstepFile(land, -5, 5, 1, false) != "" {
		t.Fatal("steps outside the map or without a world must be silent")
	}
}

// Game.exe 0x40f7e0: the mount's footstep surface follows the terrain layer's
// own step file; water and dungeons override it.
func TestMountStepFileSurfaces(t *testing.T) {
	layers := []world.TerrainLayer{{Sound: "ch_walks001.ogg"}, {Sound: "ch_walkg001.ogg"}, {Sound: "ch_walke001.ogg"}}
	w := &scene.World{WLD: &world.WLD{MapSize: 4, TextureMap: []byte{0, 1, 2, 0, 1, 2, 0, 1, 2}, Layers: layers}}
	for range 15 {
		for _, tc := range []struct {
			race uint8
			x, y float32
			want string
		}{
			{0, 0, 5, "run_horse_st0"}, {1, 2, 5, "run_puma_gr0"}, {3, 4, 5, "run_baslk_ea0"}, {2, 2, .5, "run_puma_gr0"}, {2, 2, -.5, "run_puma_gr0"},
		} {
			if got := mountStepFile(w, tc.race, tc.x, tc.y, 0, false); len(got) != len(tc.want)+5 || got[:len(tc.want)] != tc.want {
				t.Fatalf("race %d at (%v,%v): %q want prefix %q", tc.race, tc.x, tc.y, got, tc.want)
			}
		}
		if got := mountStepFile(&scene.World{WLD: &world.WLD{Dungeon: true}}, 0, 1, 5, 1, false); got[:len("run_horse_st0")] != "run_horse_st0" {
			t.Fatalf("dungeon mount step %q", got)
		}
	}
	if mountStepFile(nil, 0, 0, 0, 0, false) != "" || mountStepFile(w, 4, 0, 5, 0, false) != "" {
		t.Fatal("no world or unknown race must be silent")
	}
}
