package app

import (
	"fmt"
	"strings"
)

// gmTargetText aims /mera, /nera and /kill at the selected target: written
// alone (or with the old "t" argument) they become "<command> #<id>", which the
// world server resolves. "/mera <range>" is left as it is. It reports false,
// with a notice, when there is nothing valid to aim at.
func (a *App) gmTargetText(text string) (string, bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 || len(fields) > 2 || len(fields) == 2 && !strings.EqualFold(fields[1], "t") {
		return text, true
	}
	switch strings.ToLower(fields[0]) {
	case "/mera", "/nera", "/kill":
	default:
		return text, true
	}
	if a.state == nil || a.state.Target == 0 || a.state.Target == a.state.SelfID {
		a.systemMessage(17, 12374, nil)
		return "", false
	}
	return fmt.Sprintf("%s #%d", fields[0], a.state.Target), true
}
