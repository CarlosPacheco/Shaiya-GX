package app

import (
	"log"
	"math/rand/v2"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// loadingArtRules reads MapScreens.INI and lists the random pool once.
func (a *App) loadingArtRules() {
	if a.loadingRulesLoaded || a.arc == nil {
		return
	}
	a.loadingRulesLoaded = true
	if _, ok := a.arc.Lookup(ui.MapScreensFile); ok {
		if raw, err := a.arc.ReadFile(ui.MapScreensFile); err == nil {
			a.mapScreens = ui.ParseMapScreens(raw)
		} else {
			log.Printf("%s: %v", ui.MapScreensFile, err)
		}
	}
	if dir, ok := a.arc.LookupDir("interface"); ok {
		files := make([]string, 0, len(dir.Files))
		for _, f := range dir.Files {
			files = append(files, f.Name)
		}
		a.randomLoading = ui.RandomLoadingNames(files)
	}
}

// loadingArt picks the picture for a map load: the map's MapScreens.INI entry
// when its file exists, otherwise any random_loadingNN picture. Nil keeps the
// built-in picture. The character-select load has no map.
func (a *App) loadingArt(mapID uint16, selection bool) *render.Texture {
	a.loadingArtRules()
	name := ""
	if file, ok := a.mapScreens[mapID]; ok && !selection {
		if _, found := a.arc.Lookup("interface/" + file); found {
			name = file
		}
	}
	if name == "" && len(a.randomLoading) > 0 {
		name = a.randomLoading[rand.IntN(len(a.randomLoading))]
	}
	if name == "" {
		name = "loading.png"
		if _, found := a.arc.Lookup("interface/" + name); !found {
			return nil
		}
	}
	tex, err := a.Interface(name)
	if err != nil {
		return nil
	}
	return tex
}
