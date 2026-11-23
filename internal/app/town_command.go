package app

import (
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// townCommand is the chat command of Game.exe 0x478fcb: "/town" or the
// localized "/return" (SysMsg 5076). Alive and with no action pending, the
// client sends 0x0556 and prints SysMsg 251 in the system style (type 7); the
// server moves the character to its bind point 10 s later unless it moves,
// attacks, casts or is hit first. A dead character keeps the old chat route.
func (a *App) townCommand(text string) bool {
	command := strings.TrimSpace(text)
	if command == "" || a.state == nil || a.state.Self.Dead || a.world == nil {
		return false
	}
	return a.sendTown(command)
}

func (a *App) sendTown(command string) bool {
	alias := a.msg.Get(5076)
	if alias == "" {
		alias = "/return"
	}
	if !strings.EqualFold(command, "/town") && !strings.EqualFold(command, alias) {
		return false
	}
	if err := a.world.Send(proto.NewWriter(proto.ReturnToTown).Bytes()); err != nil {
		a.backToLoginID(12020)
		return true
	}
	a.systemMessage(7, 251, nil)
	return true
}
