// Package assetfuzz corrupts real game assets and checks that no parser panics
// or hangs. It needs loose client data: set SHAIYA_DATA, otherwise it skips.
package assetfuzz

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/audio"
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
)

type parser struct {
	name string
	fn   func([]byte) error
}

func wrap[T any](f func([]byte) (T, error)) func([]byte) error {
	return func(b []byte) error { _, err := f(b); return err }
}

func TestAssetParsersNeverPanic(t *testing.T) {
	root := os.Getenv("SHAIYA_DATA")
	if root == "" {
		t.Skip("set SHAIYA_DATA to loose client data")
	}
	if _, err := os.Stat(root); err != nil {
		t.Skip(err)
	}
	byExt := map[string][]parser{
		".wld":  {{"ParseWLD", wrap(world.ParseWLD)}},
		".smod": {{"ParseSMOD", wrap(world.ParseSMOD)}, {"ParseSMODCollisions", wrap(world.ParseSMODCollisions)}, {"ParseTreeCollisions", wrap(world.ParseTreeCollisions)}},
		".dg":   {{"ParseDG", wrap(world.ParseDG)}, {"ParseDGCollisions", wrap(world.ParseDGCollisions)}},
		".wtr":  {{"ParseWTR", wrap(world.ParseWTR)}},
		".mani": {{"ParseMani", wrap(world.ParseMani)}, {"ParseVANICollisions", wrap(world.ParseVANICollisions)}},
		".vani": {{"ParseVANI", wrap(world.ParseVANI)}, {"ParseVANICollisions", wrap(world.ParseVANICollisions)}},
		".3do":  {{"Parse3DO", wrap(world.Parse3DO)}},
		".itm":  {{"ParseITM", wrap(model.ParseITM)}},
		".alt":  {{"ParseALT", wrap(model.ParseALT)}},
		".3dc":  {{"Parse3DC", wrap(model.Parse3DC)}},
		".ani":  {{"ParseANI", wrap(model.ParseANI)}},
		".mlt":  {{"ParseMLT", wrap(model.ParseMLT)}},
		".mon":  {{"ParseMON", wrap(model.ParseMON)}},
		".eft":  {{"ParseEFT", wrap(effect.ParseEFT)}},
		".seff": {{"ParseSEFF", wrap(effect.ParseSEFF)}},
		".3de":  {{"Parse3DE", wrap(effect.Parse3DE)}},
		".dds":  {{"Decode", wrap(img.Decode)}, {"DecodeDDS", wrap(img.DecodeDDS)}},
		".tga":  {{"DecodeTGA", wrap(img.DecodeTGA)}},
		".bmp":  {{"DecodeBMP", wrap(img.DecodeBMP)}},
		".png":  {{"DecodePNG", wrap(img.DecodePNG)}},
		".ogg":  {{"DecodeOGG", wrap(audio.DecodeOGG)}},
	}
	sdataParsers := map[string]parser{
		"skill.sdata":   {"ParseSkills", wrap(sdata.ParseSkills)},
		"item.sdata":    {"ParseItems", wrap(sdata.ParseItems)},
		"npc.sdata":     {"ParseNPCs", wrap(sdata.ParseNPCs)},
		"monster.sdata": {"ParseMonsters", wrap(sdata.ParseMonsters)},
	}
	rng := rand.New(rand.NewPCG(7, 9))
	files := map[string][]string{}
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if _, ok := byExt[ext]; ok {
			info, _ := d.Info()
			if info != nil && info.Size() < 3<<20 {
				files[ext] = append(files[ext], p)
			}
		} else if _, ok := sdataParsers[strings.ToLower(filepath.Base(p))]; ok {
			files["sdata"] = append(files["sdata"], p)
		}
		return nil
	})
	problems := map[string]string{}
	counts := map[string]int{}
	start := time.Now()
	run := func(pr parser, file string, data []byte) {
		done := make(chan string, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					st := string(debug.Stack())
					lines := strings.Split(st, "\n")
					loc := ""
					for i, l := range lines {
						if strings.Contains(l, "Shaiya-GX/internal") && !strings.Contains(l, "assetfuzz") && i+1 < len(lines) {
							loc = strings.TrimSpace(lines[i+1])
							break
						}
					}
					done <- fmt.Sprintf("panic %v at %s", r, loc)
					return
				}
				done <- ""
			}()
			pr.fn(data)
		}()
		select {
		case msg := <-done:
			if msg != "" {
				key := pr.name + ": " + msg
				if _, ok := problems[key]; !ok {
					problems[key] = file
				}
			}
		case <-time.After(10 * time.Second):
			problems[pr.name+": hang >10s"] = file
		}
		counts[pr.name]++
	}
	for ext, list := range files {
		sort.Strings(list)
		rng.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] })
		limit := 12
		if ext == "sdata" {
			limit = 8
		}
		if len(list) > limit {
			list = list[:limit]
		}
		for _, f := range list {
			orig, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			var ps []parser
			if ext == "sdata" {
				ps = []parser{sdataParsers[strings.ToLower(filepath.Base(f))]}
			} else {
				ps = byExt[ext]
			}
			for _, pr := range ps {
				run(pr, f, orig)
				for round := 0; round < 60; round++ {
					b := append([]byte(nil), orig...)
					switch rng.IntN(5) {
					case 0: // flip a few bytes
						for k := 0; k < 1+rng.IntN(8) && len(b) > 0; k++ {
							b[rng.IntN(len(b))] = byte(rng.IntN(256))
						}
					case 1: // truncate
						if len(b) > 1 {
							b = b[:rng.IntN(len(b))]
						}
					case 2: // 0xff run
						if len(b) > 8 {
							at := rng.IntN(len(b) - 4)
							for k := 0; k < 4; k++ {
								b[at+k] = 0xff
							}
						}
					case 3: // zero a header-ish region
						n := min(len(b), 64)
						at := rng.IntN(max(1, n-8))
						for k := 0; k < 8 && at+k < len(b); k++ {
							b[at+k] = 0
						}
					case 4: // huge count in the first 64 bytes
						if len(b) > 16 {
							at := rng.IntN(min(len(b)-4, 60))
							b[at], b[at+1], b[at+2], b[at+3] = 0xff, 0xff, 0xff, 0x7f
						}
					}
					run(pr, f, b)
				}
			}
		}
	}
	total := 0
	for _, c := range counts {
		total += c
	}
	t.Logf("%d parse runs over %d parsers in %v; %d distinct problems", total, len(counts), time.Since(start).Round(time.Millisecond), len(problems))
	keys := make([]string, 0, len(problems))
	for k := range problems {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Errorf("%s  (e.g. %s)", k, filepath.Base(problems[k]))
	}
}
