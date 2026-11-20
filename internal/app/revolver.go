package app

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type revolverSlot struct {
	Skill  uint16 `json:",omitempty"`
	Action uint8  `json:",omitempty"` // Basic Action index plus one.
}

type revolverConfig struct {
	Slots    [ui.RevolverPages][ui.RevolverSlots]revolverSlot
	Counts   [ui.RevolverPages]int
	Page     int
	Expanded bool
}

type revolverCast struct {
	skill      sdata.SkillKey
	page, slot int
	deadline   float64
}

const (
	revolverBackground = 9
	revolverActiveCell = 10
	revolverEditorCell = 11
)

type revolverView struct {
	revolverConfig
	Position             ui.WindowPosition
	profile              string
	selected             int
	held, startX, startY int
	moving, dragging     bool
	cast                 revolverCast
}

func (v *revolverView) count() int {
	return min(ui.RevolverSlots, max(3, v.Counts[v.Page]))
}

func (v *revolverView) step(delta int) {
	count := v.count()
	v.selected = (v.selected + count + delta) % count
	v.cast = revolverCast{}
}

func readRevolvers(path string) (map[string]revolverConfig, error) {
	configs := make(map[string]revolverConfig)
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return configs, nil
	}
	if err == nil {
		err = json.Unmarshal(b, &configs)
	}
	if configs == nil {
		configs = make(map[string]revolverConfig)
	}
	return configs, err
}

func revolverPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "shaiya-go", "revolver.json")
}

func (a *App) revolverState() *revolverView {
	v := &a.revolver
	if a.state == nil || a.state.SelfID == 0 {
		return v
	}
	key := a.localActionsKey()
	if v.profile == key {
		return v
	}
	*v = revolverView{Position: v.Position, profile: key}
	if path := revolverPath(); path != "" {
		m, err := readRevolvers(path)
		if err != nil {
			log.Printf("load revolver: %v", err)
		} else {
			v.revolverConfig = m[key]
			v.Page = min(ui.RevolverPages-1, max(0, v.Page))
		}
	}
	return v
}

func (a *App) saveRevolver() {
	v := &a.revolver
	path := revolverPath()
	if path == "" || v.profile == "" {
		return
	}
	m, err := readRevolvers(path)
	if err == nil {
		m[v.profile] = v.revolverConfig
		var data []byte
		data, err = json.Marshal(m)
		if err == nil {
			err = writeRevolverConfig(path, data)
		}
	}
	if err != nil {
		log.Printf("save revolver: %v", err)
	}
}

func writeRevolverConfig(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".revolver-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}

func (a *App) revolverLayout(w, h int) ui.RevolverLayout {
	v := a.revolverState()
	return ui.NewRevolverLayout(v.Position, w, h, v.count(), v.Expanded)
}

func (a *App) revolverDrop(in *platform.Input, w, h int, slot revolverSlot) bool {
	if a.barCovered(in, w, h) || a.actionBar.Contains(in.MouseX, in.MouseY, w, h) {
		return false
	}
	v := a.revolverState()
	layout := a.revolverLayout(w, h)
	index := -1
	if v.Expanded {
		for i := range v.count() {
			if layout.Cell(i, true).Contains(in.MouseX, in.MouseY) {
				index = i
				break
			}
		}
	}
	if index < 0 {
		for i := range 3 {
			if layout.Cell(i, false).Contains(in.MouseX, in.MouseY) {
				index = ui.RevolverCompactSlot(i, v.selected, v.count())
				break
			}
		}
	}
	if index < 0 {
		return false
	}
	if slot.Skill != 0 || slot.Action != 0 {
		v.Slots[v.Page][index] = slot
		if index == v.selected {
			v.cast = revolverCast{}
		}
		a.saveRevolver()
		a.playUI()
	}
	return true
}

func (a *App) useRevolver() {
	if a.world == nil || a.state.Self.Dead {
		return
	}
	v := a.revolverState()
	slot := v.Slots[v.Page][v.selected]
	if slot.Action > 0 && int(slot.Action) <= len(subActionNames) {
		v.step(1)
		a.activateSubAction(int(slot.Action) - 1)
		return
	}
	if a.state.Self.Mounted {
		return
	}
	k, ok := a.learnedSkill(slot.Skill)
	if !ok {
		return
	}
	if a.skills.cooldowns[k.ID] > a.clock {
		a.learnedSkillPacket(k) // Preserve the existing reuse feedback.
		v.step(1)
		return
	}
	if p := a.learnedSkillPacket(k); p != nil && a.dispatchShortcut(p) {
		v.cast = revolverCast{skill: sdata.SkillKey{ID: k.ID, Level: k.Level}, page: v.Page, slot: v.selected, deadline: a.clock + 15}
	}
}

// EP5.4 advances a skill only when the local caster's matching result arrives.
func (a *App) confirmRevolver(r proto.SkillResult) {
	v := &a.revolver
	cast := v.cast
	if cast.skill.ID == 0 || r.Source != a.state.SelfID || cast.skill != (sdata.SkillKey{ID: r.ID, Level: r.Level}) {
		return
	}
	v.cast = revolverCast{}
	if a.clock <= cast.deadline && r.Result <= 4 && !(r.Result == 3 && r.Target == 0) &&
		v.Page == cast.page && v.selected == cast.slot && v.Slots[v.Page][v.selected].Skill == r.ID {
		v.step(1)
	}
}

func (a *App) updateRevolver(in *platform.Input, w, h int) bool {
	v := a.revolverState()
	for _, key := range in.Keys {
		switch {
		case key == platform.KeyRevolver:
			a.useRevolver()
			return true
		case key == platform.KeyRevolverReset:
			v.selected, v.cast = 0, revolverCast{}
			return true
		case key >= platform.KeyRevolverPage1 && key <= platform.KeyRevolverPage4:
			v.Page, v.selected = int(key-platform.KeyRevolverPage1), 0
			v.cast = revolverCast{}
			a.saveRevolver()
			return true
		}
	}
	layout := a.revolverLayout(w, h)
	inside := layout.Main.Contains(in.MouseX, in.MouseY) || layout.Editor.Contains(in.MouseX, in.MouseY)
	hit := revolverBackground
	for i, box := range layout.Controls() {
		if box.Contains(in.MouseX, in.MouseY) {
			hit = i + 1
			break
		}
	}
	if hit == revolverBackground && layout.Cell(1, false).Contains(in.MouseX, in.MouseY) {
		hit = revolverActiveCell
	}
	if hit == revolverBackground && v.Expanded {
		for i := range v.count() {
			if layout.Cell(i, true).Contains(in.MouseX, in.MouseY) {
				hit = revolverEditorCell + i
				break
			}
		}
	}
	covered := a.barCovered(in, w, h) || a.actionBar.Contains(in.MouseX, in.MouseY, w, h)
	if in.Pressed && inside && !covered {
		v.held, v.startX, v.startY = hit, in.MouseX, in.MouseY
		v.moving, v.dragging = false, false
	}
	if v.held == 0 {
		return inside && !covered && (in.RightPressed || in.RightDown || in.Wheel != 0)
	}
	if in.HasKey(platform.KeyEscape) || !in.Clicked && !in.MouseDown {
		v.held, v.moving, v.dragging = 0, false, false
		return true
	}
	if absInt(in.MouseX-v.startX) > 4 || absInt(in.MouseY-v.startY) > 4 {
		if v.held == revolverBackground {
			v.moving = true
		} else if v.held >= revolverEditorCell {
			v.dragging = true
		}
	}
	if v.moving {
		v.Position.Move(layout.Panel.X+in.MouseX-v.startX, layout.Panel.Y+in.MouseY-v.startY, w, h, layout.Panel.W, layout.Panel.H)
		v.startX, v.startY = in.MouseX, in.MouseY
	}
	if !in.Clicked {
		return true
	}
	held, moving, dragging := v.held, v.moving, v.dragging
	v.held, v.moving, v.dragging = 0, false, false
	switch {
	case moving:
		a.saveUILayout()
	case dragging:
		if !covered {
			from := held - revolverEditorCell
			if hit >= revolverEditorCell {
				to := hit - revolverEditorCell
				v.Slots[v.Page][from], v.Slots[v.Page][to] = v.Slots[v.Page][to], v.Slots[v.Page][from]
			} else if !inside {
				v.Slots[v.Page][from] = revolverSlot{}
			}
			a.saveRevolver()
		}
	case !inside || covered || hit != held:
	case hit == 1:
		v.Position.Vertical = !v.Position.Vertical
		l := a.revolverLayout(w, h)
		v.Position.Move(layout.Panel.X, layout.Panel.Y, w, h, l.Panel.W, l.Panel.H)
		a.saveUILayout()
	case hit == 2:
		v.selected, v.cast = 0, revolverCast{}
	case hit == 8:
		v.Page, v.selected = (v.Page+1)%ui.RevolverPages, 0
		v.cast = revolverCast{}
		a.saveRevolver()
	case hit == 3:
		v.Expanded = !v.Expanded
		a.saveRevolver()
	case hit == 4:
		v.step(1)
	case hit == 5:
		v.step(-1)
	case hit == 6 || hit == 7:
		delta := 1
		if hit == 7 {
			delta = -1
		}
		v.Counts[v.Page] = min(ui.RevolverSlots, max(3, v.count()+delta))
		v.selected = min(v.selected, v.count()-1)
		a.saveRevolver()
	case hit == revolverActiveCell:
		a.useRevolver()
	case hit >= revolverEditorCell:
		v.selected = hit - revolverEditorCell
		v.cast = revolverCast{}
	}
	return true
}

func (a *App) revolverIcon(slot revolverSlot) (out ui.ActionSlot) {
	if slot.Action > 0 && int(slot.Action) <= len(subActionNames) {
		i := slot.Action - 1
		u, v := float32(i%4)/4, float32(i/4)/4
		return ui.ActionSlot{Icon: a.assets.Texture("interface/icon/icon_sub.png"), UV: ui.UV{U0: u, V0: v, U1: u + .25, V1: v + .25}}
	}
	if k, ok := a.learnedSkill(slot.Skill); ok {
		d := a.skillDefs[sdata.SkillKey{ID: k.ID, Level: k.Level}]
		if file, uv, ok := ui.SkillIcon(d.Icon); ok {
			out.Icon, out.UV = a.assets.Texture("interface/"+file), uv
		}
		out.Cooldown = max(0, a.skills.cooldowns[k.ID]-a.clock)
		out.CooldownDuration = a.skillCooldownTotal(k.ID, d)
	}
	return out
}

func (a *App) drawRevolver(w, h int) {
	if a.arc == nil {
		return
	}
	v := a.revolverState()
	a.loadSkills()
	var art [len(ui.RevolverArt)]*render.Texture
	for i, name := range ui.RevolverArt {
		if _, ok := a.arc.Lookup("interface/" + name); ok {
			art[i] = a.assets.Texture("interface/" + name)
		}
	}
	var slots [ui.RevolverSlots]ui.ActionSlot
	for i := range v.count() {
		slots[i] = a.revolverIcon(v.Slots[v.Page][i])
	}
	layout := a.revolverLayout(w, h)
	layout.Draw(a.sprites, a.font, art, a.white, slots, v.Page, v.selected, v.count())
	in := &platform.Input{MouseX: a.lastMouseX, MouseY: a.lastMouseY}
	if a.barCovered(in, w, h) || a.actionBar.Contains(in.MouseX, in.MouseY, w, h) {
		return
	}
	for i := range v.count() {
		if v.Expanded && layout.Cell(i, true).Contains(in.MouseX, in.MouseY) {
			if k, ok := a.learnedSkill(v.Slots[v.Page][i].Skill); ok {
				a.queueSkillTooltip(sdata.SkillKey{ID: k.ID, Level: k.Level})
			}
			return
		}
	}
	for i := range 3 {
		if layout.Cell(i, false).Contains(in.MouseX, in.MouseY) {
			index := ui.RevolverCompactSlot(i, v.selected, v.count())
			if k, ok := a.learnedSkill(v.Slots[v.Page][index].Skill); ok {
				a.queueSkillTooltip(sdata.SkillKey{ID: k.ID, Level: k.Level})
			}
			return
		}
	}
}

func (a *App) drawRevolverDrag() {
	v := &a.revolver
	if !v.dragging || v.held < revolverEditorCell || v.held-revolverEditorCell >= v.count() {
		return
	}
	slot := a.revolverIcon(v.Slots[v.Page][v.held-revolverEditorCell])
	if slot.Icon != nil {
		u := slot.UV
		a.sprites.Draw(slot.Icon, float32(a.lastMouseX-16), float32(a.lastMouseY-16), 32, 32, u.U0, u.V0, u.U1, u.V1, white)
	}
}

// dispatchShortcut is shared by the regular bar and the revolver.
func (a *App) dispatchShortcut(p []byte) bool {
	if proto.Op(p) == proto.UseVehicle {
		a.requestVehicle()
		return true
	}
	if a.approachSkillPacket(p) {
		return a.wv.approachSkill != nil
	}
	if !a.prepareSkillMovement(p) {
		return false
	}
	if err := a.world.Send(p); err != nil {
		a.backToLogin("Unable to use action.")
		return false
	}
	a.requestedSkillPresentation(p)
	if proto.Op(p) == proto.UseItem || proto.Op(p) == proto.ItemBagToBag {
		a.inventory.pending = true
		a.inventory.deadline = a.clock + 10
	}
	return true
}
