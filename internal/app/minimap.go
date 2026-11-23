package app

import (
	"fmt"
	"log"
	"math"
	"time"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type mapView struct {
	compactHidden          bool
	atlas                  worldMapView
	image                  *render.Texture
	loaded, full, dragging bool
	smallZoom, zoom        float32
	u, v                   float32
	lastX, lastY           int
	markers                []ui.MapMarker
	click                  mapClick
}

// Quest readiness is only a presentation hint; Rapture validates turn-in.
func questMapReady(q sdata.Quest, active proto.ActiveQuest, s *game.State) bool {
	switch q.EndType {
	case 0:
		return q.EndNPCType != 0
	case 1:
		if !s.InventoryReady {
			return false
		}
		var counts [3]int
		for key, item := range s.Inventory {
			if key[0] < 1 || key[0] > 5 {
				continue
			}
			for i, want := range q.FarmItems {
				if want.Type != 0 && item.Type == want.Type && item.TypeID == want.ID {
					counts[i] += int(item.Count)
					break
				}
			}
		}
		for i, want := range q.FarmItems {
			if counts[i] < int(want.Count) {
				return false
			}
		}
		return true
	case 2:
		return active.Counters[0] >= q.MobCounts[0] && active.Counters[1] >= q.MobCounts[1]
	case 3:
		return active.Counters[0] >= q.PVPKills
	default:
		return false
	}
}

func (a *App) npcQuestMapKind(e *game.Entity) (ui.MapMarkerKind, bool) {
	return a.npcQuestMapKindAt(e, time.Now())
}

func (a *App) npcQuestMapKindAt(e *game.Entity, now time.Time) (ui.MapMarkerKind, bool) {
	def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
	for _, id := range def.EndQuests {
		if active, ok := a.state.Quests[id]; ok && id > 0 && int(id) <= len(a.questDefs) {
			q := a.questDefs[id-1]
			started, known := a.questStartedAt[id]
			if (!known || !now.Before(started.Add(time.Duration(q.MinMinutes)*time.Minute))) && questMapReady(q, active, a.state) {
				return ui.MapQuestEnd, true
			}
		}
	}
	for _, id := range def.StartQuests {
		if _, ok := a.state.Quests[id]; ok {
			continue
		}
		if _, ok := a.state.FinishedQuests[id]; ok {
			continue
		}
		if id > 0 && int(id) <= len(a.questDefs) && a.questStartVisible(a.questDefs[id-1], e) {
			return ui.MapQuestStart, true
		}
	}
	return 0, false
}

func (a *App) visibleQuestMapMarkers(mapSize float32, markers []ui.MapMarker) []ui.MapMarker {
	// Build objectives once, rather than scanning all quests for every creature.
	objectives := map[uint16]bool{}
	for id, active := range a.state.Quests {
		if id == 0 || int(id) > len(a.questDefs) {
			continue
		}
		q := a.questDefs[id-1]
		if q.EndType != 2 {
			continue
		}
		for i, mob := range q.MobIDs {
			if mob != 0 && active.Counters[i] < q.MobCounts[i] {
				objectives[mob] = true
			}
		}
	}
	for _, e := range a.state.Entities {
		kind, show := ui.MapParty, false
		if e.Kind == game.KindNpc {
			kind, show = a.npcQuestMapKind(e)
		}
		if e.Kind == game.KindMob && !e.Dead && objectives[e.TypeID] {
			kind, show = ui.MapQuestObjective, true
		}
		if !show {
			continue
		}
		if u, v, ok := ui.MapPosition(e.X, e.Z, mapSize); ok {
			markers = append(markers, ui.MapMarker{U: u, V: v, Kind: kind})
		}
	}
	return markers
}

// partyMapMarkers is the party radar of 0x4c73a0: every ally of the party, or in
// a raid only the members of your own group (the six slots of DAT_021fd7f4),
// on this map. The server keeps their positions current (0x0C07 / 0x0C12), so
// allies out of view still show; a visible ally uses its live position.
func partyMapMarkers(s *game.State, mapID uint16, mapSize float32, markers []ui.MapMarker) []ui.MapMarker {
	group := -1
	if s.Party.Raid {
		for slot, id := range s.Party.RaidSlots {
			if id == s.SelfID {
				group = int(slot) / 6
			}
		}
	}
	inGroup := func(id uint32) bool {
		if group < 0 {
			return !s.Party.Raid
		}
		for slot, member := range s.Party.RaidSlots {
			if member == id {
				return int(slot)/6 == group
			}
		}
		return false
	}
	for _, member := range s.Party.Members {
		if member.ID == s.SelfID || !inGroup(member.ID) {
			continue
		}
		x, z, known := member.X, member.Z, mapID != 0 && member.Instance == mapID
		if e := s.Entities[member.ID]; e != nil && e.Kind == game.KindUser {
			x, z, known = e.X, e.Z, true
		}
		if !known {
			continue
		}
		if u, v, ok := ui.MapPosition(x, z, mapSize); ok {
			markers = append(markers, ui.MapMarker{U: u, V: v})
		}
	}
	return markers
}

func (a *App) mapState() *mapView {
	if a.wv == nil {
		return nil
	}
	v := &a.wv.mapView
	if v.smallZoom == 0 {
		v.smallZoom, v.zoom, v.u, v.v = 1, 1, .5, .5
	}
	return v
}

func (a *App) loadMapImage(v *mapView) {
	if v.loaded {
		return
	}
	v.loaded = true
	file := fmt.Sprintf("interface/MiniMap/%d.png", a.wv.mapID)
	if _, ok := a.arc.Lookup(file); !ok {
		return // Not every original map includes a minimap image.
	}
	b, err := a.arc.ReadFile(file)
	if err == nil {
		var decoded img.Texture
		decoded, err = img.DecodeResource(file, b)
		if err == nil {
			v.image = render.Upload(decoded)
		}
	}
	if err != nil {
		log.Printf("minimap %d: %v", a.wv.mapID, err)
	}
}

func (a *App) updateMinimap(in *platform.Input, w, h int) bool {
	if a.minimap == nil {
		return false
	}
	v := a.mapState()
	if v == nil {
		return false
	}
	if hasChar(in, 'v') {
		v.compactHidden = !v.compactHidden
		return true
	}
	for _, ch := range in.Chars {
		if ch == 'm' || ch == 'M' {
			v.click = mapClick{}
			v.full, v.dragging = !v.full, false
			return true
		}
	}
	if v.full && in.HasKey(platform.KeyEscape) {
		v.click = mapClick{}
		v.full, v.dragging = false, false
		return true
	}
	if v.full {
		a.loadWorldMap(v)
	} else if v.compactHidden {
		return false
	}
	a.minimap.WorldAvailable = len(v.atlas.regions) > 0
	a.minimap.WorldReturn = v.atlas.overview && v.atlas.preview < 0
	action := a.minimap.Update(in, w, h, v.full)
	if action.Close {
		v.click = mapClick{}
		v.full, v.dragging = false, false
		return true
	}
	if action.World {
		v.click = mapClick{}
		if v.atlas.overview && v.atlas.preview >= 0 {
			v.atlas.preview = -1
		} else {
			v.atlas.overview = !v.atlas.overview
		}
		v.atlas.warning = false
		return true
	}
	if action.Toggle {
		v.click = mapClick{}
		v.full = !v.full
		return true
	}
	if v.full {
		worldWasOpen := v.atlas.overview
		captured := a.updateWorldMap(in, w, h)
		if captured || worldWasOpen || v.atlas.overview {
			return captured
		}
		x, y, scale := a.minimap.FullRect(w, h)
		mx, my, side := x+int(16*scale), y+int(43*scale), int(512*scale)
		over := in.MouseX >= mx && in.MouseX < mx+side && in.MouseY >= my && in.MouseY < my+side
		if v.click.update(in, over) {
			a.gmMapTeleport(a.wv.mapID, ui.MapWindow(v.u, v.v, 1/v.zoom), float32(in.MouseX-mx)/float32(side), float32(in.MouseY-my)/float32(side))
		}
		if in.Pressed && over {
			v.dragging, v.lastX, v.lastY = true, in.MouseX, in.MouseY
		}
		if v.dragging && in.MouseDown {
			v.u -= float32(in.MouseX-v.lastX) / float32(side) / v.zoom
			v.v -= float32(in.MouseY-v.lastY) / float32(side) / v.zoom
			v.lastX, v.lastY = in.MouseX, in.MouseY
		}
		if !in.MouseDown {
			v.dragging = false
		}
		if in.Wheel != 0 && over {
			old := ui.MapWindow(v.u, v.v, 1/v.zoom)
			fx, fy := float32(in.MouseX-mx)/float32(side), float32(in.MouseY-my)/float32(side)
			v.zoom = min(8, max(1, v.zoom*float32(math.Pow(1.2, in.Wheel/120))))
			v.u = old.U0 + fx*(old.U1-old.U0) + (.5-fx)/v.zoom
			v.v = old.V0 + fy*(old.V1-old.V0) + (.5-fy)/v.zoom
		}
		view := ui.MapWindow(v.u, v.v, 1/v.zoom)
		v.u, v.v = (view.U0+view.U1)/2, (view.V0+view.V1)/2
		return v.dragging || panelPointer(in, x, y, int(544*scale), int(579*scale))
	}
	x, y := a.minimap.Origin(w, h)
	if action.Zoom != 0 {
		v.click = mapClick{} // Button hit boxes can overlap the circular map edge.
	}
	if action.Zoom == 0 && a.wv.world != nil && a.wv.world.WLD != nil {
		projection := a.mapProjection(v)
		fx, fy := float32(in.MouseX-x-16)/ui.MinimapImageSize, float32(in.MouseY-y-16)/ui.MinimapImageSize
		dx, dy := fx-.5, fy-.5
		if v.click.update(in, dx*dx+dy*dy <= .25) {
			_, _, ok := projection.Position(a.wv.pos[0], a.wv.pos[2])
			if ok {
				a.gmMapTeleport(a.wv.mapID, projection.MinimapWindow(a.wv.pos[0], a.wv.pos[2], v.smallZoom), fx, fy)
			}
		}
	}
	if action.Zoom != 0 || in.Wheel != 0 && panelPointer(in, x, y, ui.MinimapWidth, ui.MinimapHeight) {
		steps := float64(action.Zoom)
		if in.Wheel != 0 {
			steps += in.Wheel / 120
		}
		v.smallZoom = min(1.5, max(.5, v.smallZoom-.1*float32(steps)))
	}
	return panelPointer(in, x, y, ui.MinimapWidth, ui.MinimapHeight)
}

func (a *App) drawMinimap(w, h int, full bool) {
	if a.minimap == nil || a.wv == nil || a.wv.world == nil || a.wv.world.WLD == nil {
		return
	}
	v := a.mapState()
	if full && !v.full || !full && v.compactHidden {
		return
	}
	a.loadMapImage(v)
	if !a.questDataLoaded {
		if err := a.loadQuests(); err != nil {
			log.Printf("minimap quest data: %v", err)
		}
	}
	projection := a.mapProjection(v)
	u, z, ok := projection.Position(a.wv.pos[0], a.wv.pos[2])
	if !ok {
		u, z = -1, -1
	}
	heading := float32(math.Atan2(float64(a.wv.facing[0]), float64(a.wv.facing[2])))
	square := max(projection.Width, projection.Height)
	markers := relicMapMarkers(a.state, square, v.markers[:0])
	markers = partyMapMarkers(a.state, a.wv.mapID, square, markers)
	markers = a.questMapMarkers(square, markers)
	markers = calibrateMapMarkers(markers, projection, square)
	v.markers = markers

	a.minimap.MapName = ui.MapName(a.msg, a.wv.mapID)
	if full {
		a.loadWorldMap(v)
		a.minimap.WorldAvailable = len(v.atlas.regions) > 0
		a.minimap.WorldReturn = v.atlas.overview && v.atlas.preview < 0
		if v.atlas.overview {
			a.drawWorldMap(w, h)
			return
		}
		a.minimap.DrawFull(a.sprites, a.font, v.image, a.white, w, h, ui.MapWindow(v.u, v.v, 1/v.zoom), u, z, heading, a.wv.mapID, markers)
		x, y, scale := a.minimap.FullRect(w, h)
		position := ui.CharacterCoordinates(a.wv.pos[0], a.wv.pos[1], a.wv.pos[2])
		a.font.DrawScaled(a.sprites, position, float32(x)+(272-float32(a.font.Width(position))/2)*scale, float32(y)+23*scale, scale, white, 0)
		fx := (float32(a.lastMouseX-x) - 16*scale) / (512 * scale)
		fy := (float32(a.lastMouseY-y) - 43*scale) / (512 * scale)
		if px, pz, inside := projection.Coordinates(ui.MapWindow(v.u, v.v, 1/v.zoom), fx, fy); inside {
			label := fmt.Sprintf("X: %.0f  Z: %.0f", px, pz)
			width := a.font.Width(label) + 8
			left, top := min(a.lastMouseX+16, w-width), min(a.lastMouseY+20, h-a.font.Height-8)
			a.sprites.Draw(a.white, float32(left), float32(top), float32(width), float32(a.font.Height+6), 0, 0, 1, 1, [4]float32{0, 0, 0, .8})
			a.font.Draw(a.sprites, label, left+4, top+3, white, 0)
		}
		return
	}
	a.updateMinimapAlarm()
	view := projection.MinimapWindow(a.wv.pos[0], a.wv.pos[2], v.smallZoom)
	// 0x4c85b8 formats the clock as "%m.%d %H:%M".
	a.minimap.Caption = time.Now().Format("01.02 15:04")
	a.minimap.Draw(a.sprites, a.font, v.image, a.white, w, h, view, u, z, heading, a.wv.mapID, a.wv.pos[0], a.wv.pos[2], markers)
}

// minimapAlarmExempt lists the maps where the alarm mechanic is disabled.
func minimapAlarmExempt(mapID uint16) bool { return mapID >= 40 && mapID <= 42 }

// updateMinimapAlarm follows 0x4c6550: the timer drops by twice the frame time
// and restarts at 2 while an enemy-faction character is around.
func (a *App) updateMinimapAlarm() {
	t := a.minimap.Alarm - 2*float32(a.frameDT)
	if t < 0 {
		t = 0
	}
	if t == 0 && a.enemyNearby() {
		t = 2
	}
	a.minimap.Alarm = t
}

func (a *App) enemyNearby() bool {
	own, ok := a.selfSlot()
	if !ok || minimapAlarmExempt(a.wv.mapID) {
		return false
	}
	for _, e := range a.state.Entities {
		if e.Kind == game.KindUser && e.ID != a.state.SelfID && e.Shape != nil && !e.Dead && own.Race/2 != e.Shape.Race/2 {
			return true
		}
	}
	return false
}
