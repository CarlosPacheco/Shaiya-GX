// gx_mapbuild generates server-only collision and navigation data from loose assets.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/mapgeometry"
	"github.com/Spelunkern/Shaiya-GX/internal/mapnav"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	data := flag.String("data", os.Getenv("SHAIYA_DATA"), "loose client data (read only)")
	output := flag.String("output", "server/data/geometry", "server geometry output directory")
	serverMaps := flag.String("server-maps", "server/data/maps", "server svmap directory for authoritative walk masks")
	physicsOnly := flag.Bool("physics-only", false, "add player collision resources to an unchanged existing navigation generation")
	selected := flag.String("maps", "3", "comma-separated map IDs or all")
	flag.Parse()
	arc, err := archive.OpenDirectory(*data)
	if err != nil {
		log.Fatal(err)
	}
	defer arc.Close()
	var ids []int
	if *selected == "all" {
		arc.Walk(func(file string, _ *archive.File) {
			lower := strings.ToLower(file)
			if path.Dir(lower) == "world" && path.Ext(lower) == ".wld" {
				if id, err := strconv.ParseUint(strings.TrimSuffix(path.Base(lower), ".wld"), 10, 16); err == nil {
					ids = append(ids, int(id))
				}
			}
		})
	} else {
		for _, word := range strings.Split(*selected, ",") {
			id, err := strconv.ParseUint(word, 10, 16)
			if err != nil {
				log.Fatal(err)
			}
			ids = append(ids, int(id))
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		started := time.Now()
		file := fmt.Sprintf("world/%d.wld", id)
		raw, err := arc.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}
		layout, err := world.ParseWLD(raw)
		if err != nil {
			log.Fatal(err)
		}
		geometry, err := mapgeometry.Load(arc.ReadFile, layout)
		if err != nil {
			log.Fatal(err)
		}
		geometry.Sources[file] = fmt.Sprintf("%x", sha256.Sum256(raw))
		var masks []mapnav.WalkMask
		maskPath := filepath.Join(*serverMaps, fmt.Sprintf("%d.svmap", id))
		if raw, e := os.ReadFile(maskPath); e == nil {
			if len(raw) < 4 {
				log.Fatal("short svmap")
			}
			size := binary.LittleEndian.Uint32(raw)
			length := uint64(size) * uint64(size) / 8
			if size == 0 || size > 16384 || length > uint64(len(raw)-4) {
				log.Fatal("invalid svmap mask")
			}
			masks = append(masks, mapnav.WalkMask{Size: size, Bits: raw[4 : 4+length]})
			geometry.Sources[fmt.Sprintf("server/maps/%d.svmap", id)] = fmt.Sprintf("%x", sha256.Sum256(raw))
		} else if !os.IsNotExist(e) {
			log.Fatal(e)
		}
		if *physicsOnly {
			err = mapnav.WritePlayerPhysics(geometry, uint16(id), *output)
		} else {
			err = mapnav.Bake(geometry, uint16(id), *output, masks...)
		}
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("map %d: navigation and collision generation ready in %s", id, time.Since(started))
	}
	// Merge successful selections into the catalog, including a single new map.
	maps := []uint16{}
	seen := map[uint16]bool{}
	if catalog, e := mapnav.ReadCatalog(*output); e == nil {
		for _, id := range catalog.Maps {
			seen[id] = true
		}
	} else if !os.IsNotExist(e) {
		log.Fatal(e)
	}
	for _, id := range ids {
		seen[uint16(id)] = true
	}
	for id := range seen {
		maps = append(maps, id)
	}
	sort.Slice(maps, func(i, j int) bool { return maps[i] < maps[j] })
	if err = mapnav.WriteCatalog(*output, maps); err != nil {
		log.Fatal(err)
	}
}
