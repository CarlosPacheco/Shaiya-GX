// Command skillpatch applies the GX cleanup to a native Character/Skill.SData:
// dead levels, chain wording, the Mana/Stamina Shot restriction text,
// Aggravation's time damage and detail 25. Input and output must differ.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
)

func main() {
	input := flag.String("input", "", "source Skill.SData")
	output := flag.String("output", "", "new output file")
	retiredOnly := flag.Bool("retired-only", false, "only remove retired player skills")
	soulBurnOnly := flag.Bool("soul-burn-only", false, "only enable Soul Burn")
	debuffTextOnly := flag.Bool("combat-debuff-text-only", false, "only update Slump Shot and Confusion tooltips")
	npcSpecialOnly := flag.Bool("npc-special-only", false, "repair Porcine Party, Ice Stun, and Wide Death Sentence in NpcSkill.SData")
	flag.Parse()
	if *input == "" || *output == "" || *input == *output {
		log.Fatal("distinct input and output paths required")
	}
	var patch func([]byte) ([]byte, sdata.SkillCatalogReport, error)
	for _, option := range []struct {
		selected bool
		apply    func([]byte) ([]byte, sdata.SkillCatalogReport, error)
	}{
		{*retiredOnly, sdata.RemoveRetiredPlayerSkills},
		{*soulBurnOnly, sdata.EnableSoulBurn},
		{*debuffTextOnly, sdata.UpdateCombatDebuffTooltips},
		{*npcSpecialOnly, sdata.RepairNPCSkillCatalog},
	} {
		if option.selected {
			if patch != nil {
				log.Fatal("choose one single-purpose patch")
			}
			patch = option.apply
		}
	}
	if patch == nil {
		patch = sdata.NormalizeSkillCatalog
	}
	raw, err := os.ReadFile(*input)
	if err != nil {
		log.Fatal(err)
	}
	out, report, err := patch(raw)
	if err != nil {
		log.Fatal(err)
	}
	if err = os.WriteFile(*output, out, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Println(report)
}
