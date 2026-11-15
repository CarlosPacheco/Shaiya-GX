// laddercatalog refreshes small, asset-free native ladder placement facts.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
)

func main() {
	data := flag.String("data", os.Getenv("SHAIYA_DATA"), "loose data directory, read only")
	output := flag.String("output", "internal/mapgeometry/native_ladders.json", "derived coordinate catalog")
	flag.Parse()
	arc, err := archive.OpenDirectory(*data)
	if err != nil {
		log.Fatal(err)
	}
	defer arc.Close()
	catalog := map[uint16][]mapgeometry.Ladder{}
	count := 0
	arc.Walk(func(file string, _ *archive.File) {
		lower := strings.ToLower(file)
		if path.Dir(lower) != "world" || path.Ext(lower) != ".wld" {
			return
		}
		id, err := strconv.ParseUint(strings.TrimSuffix(path.Base(lower), ".wld"), 10, 16)
		if err != nil {
			return
		}
		raw, err := arc.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}
		w, err := world.ParseWLD(raw)
		if err != nil {
			log.Fatal(err)
		}
		ladders, err := mapgeometry.BuildLadders(arc.ReadFile, w)
		if err != nil {
			log.Fatal(file, err)
		}
		if len(ladders) > 0 {
			catalog[uint16(id)] = ladders
			count += len(ladders)
		}
	})
	raw, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err = os.WriteFile(*output, append(raw, '\n'), 0644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d ladders in %d maps\n", count, len(catalog))
}
