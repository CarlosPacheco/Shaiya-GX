package app

import (
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
)

// A cost of 1 spends the whole pool: the tooltip and the bar label say so with
// the client's SysMsg 5322 instead of printing 1.
func TestSkillCostOfOneReadsSpendAll(t *testing.T) {
	a := inputApp()
	a.msg = text.SysMsg{5322: "spend all "}
	key := sdata.SkillKey{ID: 106, Level: 2}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{key: {Name: "Sacrifice", Effect: 2, SP: 1, MP: 1, ResetTime: 600}}
	a.queueSkillTooltip(key)
	joined := ""
	for _, row := range a.tooltip {
		joined += row.text + "|"
	}
	if !strings.Contains(joined, "SP: spend all|") || !strings.Contains(joined, "MP: spend all|") || strings.Contains(joined, "SP: 1") || strings.Contains(joined, "MP: 1") {
		t.Fatalf("tooltip rows %q", joined)
	}
	a.skillDefs[key] = sdata.Skill{Name: "Heal", Effect: 2, SP: 12, MP: 30}
	a.queueSkillTooltip(key)
	joined = ""
	for _, row := range a.tooltip {
		joined += row.text + "|"
	}
	if !strings.Contains(joined, "SP: 12|") || !strings.Contains(joined, "MP: 30|") {
		t.Fatalf("ordinary costs changed: %q", joined)
	}
	if got := a.skillCostValue(1); got != "spend all" {
		t.Fatalf("cost text %q", got)
	}
	a.msg = text.SysMsg{}
	if got := a.skillCostValue(1); got != "1" {
		t.Fatalf("without the SysMsg the number stays: %q", got)
	}
}
