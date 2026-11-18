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
	flag.Parse()
	if *input == "" || *output == "" || *input == *output {
		log.Fatal("distinct input and output paths required")
	}
	raw, err := os.ReadFile(*input)
	if err != nil {
		log.Fatal(err)
	}
	out, report, err := sdata.NormalizeSkillCatalog(raw)
	if err != nil {
		log.Fatal(err)
	}
	if err = os.WriteFile(*output, out, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Println(report)
}
