package app

import (
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type chatBubble struct {
	speaker uint32
	text    string
	until   float64
}

// Game.exe 0x40fbc0 replaces one actor's text and restarts its five-second timer.
// Only authoritative ordinary-chat echoes create speech bubbles.
func (a *App) confirmedChatBubble(op proto.Opcode) {
	if op != proto.ChatNormal || a.wv == nil || len(a.state.Chat) == 0 {
		return
	}
	line := a.state.Chat[len(a.state.Chat)-1]
	if line.Op != op || line.SenderID == 0 || line.ChatProtected || strings.TrimSpace(line.Text) == "" {
		return
	}
	if a.speakerConcealed(line.SenderID) {
		return
	}
	for i := range a.chatBubbles {
		if a.chatBubbles[i].speaker == line.SenderID {
			a.chatBubbles[i] = chatBubble{line.SenderID, line.Text, a.clock + 5}
			return
		}
	}

	a.chatBubbles = append(a.chatBubbles, chatBubble{line.SenderID, line.Text, a.clock + 5})
}

// speakerConcealed reports invisible characters, GMs included: their balloons
// would otherwise reveal where they stand.
func (a *App) speakerConcealed(id uint32) bool {
	if id == a.state.SelfID {
		return a.state.Self.Concealed
	}
	e := a.state.Entities[id]
	return e != nil && e.Concealed
}

func (a *App) chatBubbleOrigin(id uint32) (render.V3, bool) {
	var pos render.V3
	var vehicle proto.VehicleState
	if id == a.state.SelfID {
		pos, vehicle = a.wv.visualPosition(), a.state.Self.Vehicle
	} else if e := a.state.Entities[id]; e != nil {
		pos, vehicle = a.visualEntityPosition(e), e.Vehicle
	} else {
		return render.V3{}, false
	}
	if actor := a.playerActor(id); actor != nil {
		pos = render.V3{actor.World[12], actor.World[13], actor.World[14]}
		if actor.Mounted {
			driver := id
			passenger := vehicle.State == proto.VehiclePassenger
			if passenger {
				driver = vehicle.Driver
			}
			if mount := a.driverMount(driver); mount != nil {
				pos = mount.RestRiderSeat(passenger)
			}
		}
	}
	return pos, true
}

func (a *App) drawChatBubbles(w, h int) {
	if a.wv == nil || a.scene3d == nil || !a.clientSettings().WordBalloon {
		return // Word Balloon off (0x411e10): no balloon is updated or drawn
	}
	f := a.chatFont()
	vp := a.scene3d.View.Mul(a.scene3d.Proj)
	active := a.chatBubbles[:0]
	filter := a.chatFilter != nil && a.clientSettings().BannedWords
	for _, b := range a.chatBubbles {
		if a.clock >= b.until {
			continue
		}
		pos, exists := a.chatBubbleOrigin(b.speaker)
		if !exists || a.speakerConcealed(b.speaker) {
			continue
		}
		active = append(active, b)
		if pos.Sub(a.wv.pos).Len() > 50 {
			continue
		}
		// 0x411e57 passes the actor origin to 0x41c830, which adds 1.5.
		pos[1] += nativeCharacterOriginHeight + 1.5
		x, y, _, ok := projectLabel(pos, vp, w, h)
		if !ok {
			continue
		}
		body := b.text
		if filter {
			body = a.chatFilter.Censor(body)
		}
		width := f.Width(body)
		left, top := x-width/2-10, y-10
		a.sprites.Draw(a.white, float32(left), float32(top), float32(width+20), 30, 0, 0, 1, 1, [4]float32{72.0 / 255, 72.0 / 255, 72.0 / 255, 128.0 / 255})
		f.Draw(a.sprites, body, left+10, top+(30-f.Height)/2, white, 0)
	}
	a.chatBubbles = active
}
