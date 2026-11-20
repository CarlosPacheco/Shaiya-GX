package app

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func (a *App) confirmedFirecracker(p []byte) {
	if a.state == nil || a.wv == nil {
		return
	}
	event, err := proto.DecodeFirecracker(p)
	if err != nil {
		return
	}
	// The original localized message is SysMsg 835. The explosion supplies
	// its own audio, so use the on-screen notice style without a news chime.
	a.systemMessage(27, 835, map[string]string{"p": event.Name})
	sound := "mob_explosion001.ogg"
	if event.Variant == 2 {
		sound = "mob_explosion002.ogg"
	}
	a.playSound(sound, event.X, event.Z)
	if !a.clientSettings().SkillEffects || len(a.wv.impacts) >= 32 {
		return
	}
	if instance := a.loadSkillEffect(fmt.Sprintf("Effect/%dflarebomb.EFT", event.Variant), 0); instance != nil {
		matrix := render.Basis(render.V3{event.X, event.Y, event.Z}, render.V3{0, 0, 1}, render.V3{0, 1, 0})
		instance.SetTransform(matrix)
		a.wv.impacts = append(a.wv.impacts, skillImpact{instance: instance, authoredLifetime: true, fixedWorld: &matrix})
	}
}
