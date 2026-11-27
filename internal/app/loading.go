package app

import (
	"errors"
	"log"
	"math/rand/v2"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/gameini"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type mapLoading struct {
	active, presented, started, drawn bool
	selection                         bool
	mapID                             uint16
	progress                          float32
	art                               *render.Texture
	artMap                            uint16
	artSet                            bool
	tip                               string
}

func (a *App) beginMapLoading(mapID uint16, selection bool) {
	a.loading = mapLoading{active: true, selection: selection, mapID: mapID, tip: a.randomLoadingTip()}
}

func (a *App) randomLoadingTip() string {
	const name = "Tip.INI"
	if a.arc == nil {
		return ""
	}
	raw, err := a.arc.ReadFile(name)
	if errors.Is(err, archive.ErrNotFound) {
		return ""
	}
	if err != nil {
		log.Printf("%s: %v", name, err)
		return ""
	}
	config, err := gameini.Parse(raw)
	if err != nil {
		log.Printf("%s: %v", name, err)
		return ""
	}
	var tips []string
	for _, tip := range config["TIPS"] {
		if strings.TrimSpace(tip) != "" {
			tips = append(tips, tip)
		}
	}
	if len(tips) == 0 {
		return ""
	}
	return tips[rand.IntN(len(tips))]
}

// Present the overlay before synchronous map setup, then keep it up until
// the initial visible cells/resources are ready. Network packets still pump.
func (a *App) updateMapLoading() bool {
	if a.world == nil || a.state == nil {
		return false
	}
	phase := a.state.Phase
	if phase == game.PhaseCharSelect && a.screen == screenCharacters && !a.creating {
		fury := a.state.Faction.Faction == 1
		key := "select_a"
		if fury {
			key = "select_b"
		}
		if _, loaded := a.worlds[key]; !loaded {
			if !a.loading.active || !a.loading.selection {
				a.beginMapLoading(0, true)
			}
			if !a.loading.presented {
				return true
			}
			if a.selectWorld(fury) == nil {
				a.backToLoginID(12134)
				return false
			}
			a.loading = mapLoading{}
			return false
		}
	}
	if phase != game.PhaseLoading && phase != game.PhaseInWorld {
		a.loading = mapLoading{}
		return false
	}
	needsWorld := phase == game.PhaseInWorld && (a.screen != screenWorld || a.wv == nil || a.wv.mapID != a.state.MapID)
	if !a.loading.active && (phase == game.PhaseLoading || needsWorld) {
		a.beginMapLoading(a.state.MapID, false)
		a.menuOpen = false
	}
	// A transfer received during initial streaming supersedes the old plan.
	if a.loading.active && a.loading.started && a.wv != nil && a.wv.mapID != a.state.MapID {
		// Same loading screen, new plan: the picture and tip stay, so the
		// restart is not seen as a second loading screen.
		old := a.loading
		a.beginMapLoading(a.state.MapID, false)
		a.loading.art, a.loading.artMap, a.loading.artSet, a.loading.tip = old.art, old.artMap, old.artSet, old.tip
		a.loading.presented, a.loading.progress = true, old.progress
	}
	l := &a.loading
	if !l.active {
		return false
	}
	l.mapID = a.state.MapID
	if phase == game.PhaseLoading || !l.presented {
		return true
	}
	if !l.started {
		a.screen = screenWorld
		a.enterWorld()
		if a.wv == nil || a.wv.world == nil {
			a.backToLoginID(12135)
			return false
		}
		l.started = true
		l.progress = .1
	}
	if l.drawn && a.wv != nil && a.wv.world != nil && a.wv.world.Ready() {
		a.loading = mapLoading{}
		return false
	}
	return true
}
func (a *App) drawMapLoading(w, h int) {
	l := &a.loading
	if l.started && a.wv != nil && a.wv.world != nil {
		l.progress = max(l.progress, .1+.9*a.wv.world.LoadProgress())
		l.drawn = true
	}
	a.sprites.Begin(w, h)
	if !l.artSet {
		l.art, l.artMap, l.artSet = a.loadingArt(l.mapID, l.selection), l.mapID, true
	} else if l.artMap != l.mapID {
		// The map id often arrives after the first frame (0 = unknown). Keep
		// the picture on screen unless the map has a picture of its own.
		if file, ok := a.mapScreens[l.mapID]; ok && !l.selection {
			if _, found := a.arc.Lookup("interface/" + file); found {
				l.art = a.loadingArt(l.mapID, false)
			}
		}
		l.artMap = l.mapID
	}
	a.loadingScreen.Draw(a.sprites, a.font, a.white, w, h, l.progress, l.tip, l.art)
	a.sprites.Flush()
	l.presented = true
}
