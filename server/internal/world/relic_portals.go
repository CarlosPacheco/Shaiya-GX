package world

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
)

func portalMarker(country int32) uint8 {
	switch country {
	case 0:
		return 103
	case 1:
		return 101
	case 2:
		return 102
	}
	return 100
}
func bindRelicPortals(inputs []*relicPortal, saved map[uint32]db.Keep, id *uint32, defaults *[]db.Keep) error {
	known := map[uint32]bool{}
	for _, list := range relics.byMap {
		for _, r := range list {
			known[r.key] = true
		}
	}
	for _, p := range inputs {
		found := false
		for _, target := range EntitiesOnMap(p.mapID).Portals {
			if target.PortalID == p.portalID {
				p.pos = [3]float32{target.PosX, target.PosY, target.PosZ}
				found = true
				break
			}
		}
		if !found {
			continue
		}
		for side := range 2 {
			valid := p.refs[side][:0]
			for _, key := range p.refs[side] {
				if known[key] {
					valid = append(valid, key)
				}
			}
			p.refs[side] = valid
		}
		if len(p.refs[0])+len(p.refs[1]) == 0 {
			continue
		}
		if len(relics.portals) >= 12 {
			return fmt.Errorf("too many relic portal links")
		}
		p.country = 3
		p.id = *id
		*id++
		if k, ok := saved[p.key]; ok {
			if k.Country > 3 {
				return fmt.Errorf("invalid portal keep country")
			}
			p.country = int32(k.Country)
		} else {
			*defaults = append(*defaults, db.Keep{ID: p.key, Country: 3})
		}
		p.open = [2]bool{p.country == 0 || p.country == 1, p.country == 0 || p.country == 2}
		relics.portals = append(relics.portals, p)
		setEventPortal(p.mapID, p.portalID, p.country)
	}
	return nil
}
func setEventPortal(mapID uint16, id, country int32) {
	if !hasGrid(mapID) {
		return
	}
	s := shard(mapID)
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.entities.Portals {
		p := &s.entities.Portals[i]
		if p.PortalID == id {
			p.FactionOrPortalID = country
			return
		}
	}
}
func changeRelicPortal(p *relicPortal, open [2]bool, out *[]eventPublication, log logPrinter) bool {
	country := int32(3)
	switch {
	case open[0] && open[1]:
		country = 0
	case open[0]:
		country = 1
	case open[1]:
		country = 2
	}
	if country == p.country {
		p.open = open
		return false
	}
	if err := db.SaveKeeps([]db.Keep{{ID: p.key, Country: uint8(country)}}); err != nil {
		if log != nil {
			log.Printf("GX Server World portal keep save failed: %v", err)
		}
		return false
	}
	old := p.country
	p.country, p.open = country, open
	setEventPortal(p.mapID, p.portalID, country)
	*out = append(*out, eventPublication{mapID: p.mapID, frame: buildRelicOwner(p.id, portalMarker(country))})
	w := packet.New(0x2104)
	w.U16(uint16(p.portalID))
	w.U8(portalMarker(old))
	w.U8(portalMarker(country))
	*out = append(*out, eventPublication{global: true, frame: w.MustFinish()})
	return true
}

// Native all/none hysteresis uses the last nonneutral conqueror, not proximity
// or the current neutral appearance. Adjacent faction entries exclude each other.
func updateRelicPortalsLocked(out *[]eventPublication, log logPrinter) {
	owners := map[uint32]uint8{}
	for _, list := range relics.byMap {
		for _, r := range list {
			owners[r.key] = r.last
		}
	}
	closeNeighbor := map[int]bool{}
	for i, p := range relics.portals {
		side := 0
		if len(p.refs[0]) == 0 {
			side = 1
		}
		refs := p.refs[side]
		if len(refs) == 0 {
			continue
		}
		matches := 0
		for _, key := range refs {
			if owners[key] == uint8(side+1) {
				matches++
			}
		}
		open := p.open
		if matches == len(refs) {
			if !open[side] {
				open[side] = true
				if changeRelicPortal(p, open, out, log) {
					neighbor := i + 1
					if side == 1 {
						neighbor = i - 1
					}
					closeNeighbor[neighbor] = true
				}
			}
		} else if matches == 0 {
			open[side] = false
			changeRelicPortal(p, open, out, log)
		}
	}
	for i, p := range relics.portals {
		if closeNeighbor[i] {
			side := 0
			if len(p.refs[0]) == 0 {
				side = 1
			}
			open := p.open
			open[side] = false
			changeRelicPortal(p, open, out, log)
		}
	}
}
