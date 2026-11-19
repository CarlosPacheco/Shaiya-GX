package ui

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
)

func TestMapNameUsesSysMsgRange(t *testing.T) {
	messages := text.SysMsg{5069: "Legacy", 11001: "Borderland", 11002: "Erina", 11100: "Map 99", 11200: "Custom", 11201: "Unrelated"}
	for id, want := range map[uint16]string{
		0: "Borderland", 1: "Erina", 2: "Map 2", 99: "Map 99", 199: "Custom", 200: "Map 200",
	} {
		if got := MapName(messages, id); got != want {
			t.Errorf("map %d = %q, want %q", id, got, want)
		}
	}
}
