package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type playerFeedback struct {
	id    uint32
	mapID uint16
	kind  uint8
	pos   render.V3
}

func (a *App) queuePlayerFeedback(id uint32, kind uint8) {
	if id == 0 || a.state == nil || a.state.Phase != game.PhaseInWorld {
		return
	}
	var pos render.V3
	if id == a.state.SelfID {
		self := a.state.Self
		pos = render.V3{self.X, self.Y, self.Z}
	} else if e := a.state.Entities[id]; e != nil && e.Kind == game.KindUser {
		pos = render.V3{e.X, e.Y, e.Z}
	} else {
		return
	}
	if len(a.playerFeedback) >= 32 {
		return
	}
	a.playerFeedback = append(a.playerFeedback, playerFeedback{id: id, mapID: a.state.MapID, kind: kind, pos: pos})
}

func (a *App) confirmedPlayerFeedback(p []byte, previousLevel uint16) {
	switch proto.Op(p) {
	case proto.CharacterEnteredMap:
		if r, err := proto.DecodeUserEnter(p); err == nil && r.ID == a.state.SelfID && r.Marker != 0 {
			a.queuePlayerFeedback(r.ID, proto.PlayerSpawnFeedback)
		}
	case proto.LevelUpSelf, proto.LevelUpOther:
		r := proto.NewReader(p)
		id, level := r.U32(), r.U16()
		// GM point updates reuse 0508 without increasing the level.
		if id != a.state.SelfID || level > previousLevel {
			a.queuePlayerFeedback(id, proto.PlayerLevelFeedback)
		}
	case proto.LearnNewSkill:
		if a.state.LastLearn.Result == 0 {
			a.queuePlayerFeedback(a.state.SelfID, proto.PlayerSkillFeedback)
		}
	}
}

// Flush after the destination scene and actor positions are ready.
func (a *App) updatePlayerFeedback() {
	for _, cue := range a.playerFeedback {
		if cue.mapID != a.wv.mapID {
			continue
		}
		pos, ok := a.effectPosition(cue.id)
		if !ok {
			continue
		}
		var file, sound string
		sequence := 0
		switch cue.kind {
		case proto.PlayerSpawnFeedback:
			file = "Effect/Start.EFT"
		case proto.PlayerSkillFeedback:
			file, sound = "Effect/Up.EFT", "ui_skillnew001.wav"
		case proto.PlayerLevelFeedback:
			file, sound = "Effect/LevelUp.EFT", "ch_levup001.wav"
			if cue.id == a.state.SelfID {
				sequence = int(a.state.Faction.Faction)
			} else if e := a.state.Entities[cue.id]; e.Shape != nil {
				sequence = int(e.Shape.Faction)
			} else {
				continue
			}
		}
		a.playSoundWithin(sound, pos[0], pos[2], 10)
		if len(a.wv.impacts) >= 32 {
			continue
		}
		if effect := a.loadSkillEffect(file, sequence); effect != nil {
			impact := skillImpact{target: cue.id, instance: effect, authoredLifetime: true}
			if cue.kind == proto.PlayerSpawnFeedback {
				matrix := render.Identity()
				matrix[12], matrix[13], matrix[14] = cue.pos[0], cue.pos[1], cue.pos[2]
				impact.fixedWorld = &matrix
			}
			a.wv.impacts = append(a.wv.impacts, impact)
		}
	}
	a.playerFeedback = a.playerFeedback[:0]
}
