package world

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Obelisk.ini uses repeated [Mob] keys and nested tuples, not an INI key map.
// Country/mob triplets here stay in wire/config order: neutral, Light, Fury.
type eventSpawn struct {
	pos    [3]float32
	radius float32
	mobs   []uint16
	count  int
	actors []*eventActor
	next   time.Time
}
type relic struct {
	mapID                uint16
	key, id              uint32 // persistent map<<16|number versus sequential client identity
	pos                  [3]float32
	mobs                 [3]uint16
	initial, owner, last uint8
	changeTime           uint32
	principal            *eventActor
	guards               []*eventSpawn
	standing             bool
	next                 time.Time
	pending              *relicCapture
}
type linkedBoss struct {
	mapID                  uint16
	key                    uint32
	pos                    [3]float32
	mobs                   []uint16
	guards                 []*eventSpawn
	portalMap              uint16
	portalID               int32
	delay                  time.Duration
	random                 bool
	randomMin, randomRange time.Duration
	randomPositions        [3][3]float32
	stage                  int
	portalOpen             bool
	principal              *eventActor
	next                   time.Time
}
type relicPortal struct {
	key, id  uint32
	mapID    uint16
	portalID int32
	refs     [2][]uint32
	open     [2]bool
	country  int32
	pos      [3]float32
}
type relicConfig struct {
	relics  []*relic
	bosses  []*linkedBoss
	portals []*relicPortal
}

var eventNumber = regexp.MustCompile(`[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?`)

func eventNumbers(s string) ([]float64, error) {
	var out []float64
	for _, n := range eventNumber.FindAllString(s, -1) {
		v, e := strconv.ParseFloat(n, 32)
		if e != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("invalid event number %q", n)
		}
		out = append(out, v)
	}
	return out, nil
}
func eventInteger(v float64, max uint64) bool {
	return v >= 0 && v <= float64(max) && math.Trunc(v) == v
}
func eventPosition(n []float64) [3]float32 {
	return [3]float32{float32(n[0]), float32(n[1]), float32(n[2])}
}
func eventMobIDs(n []float64) ([]uint16, error) {
	out := make([]uint16, len(n))
	for i, v := range n {
		if !eventInteger(v, 65535) {
			return nil, fmt.Errorf("invalid event mob ID")
		}
		out[i] = uint16(v)
	}
	return out, nil
}

func parseRelicConfig(raw []byte) (relicConfig, error) {
	var out relicConfig
	zone := -1
	var active *relic
	var boss *linkedBoss
	var portal *relicPortal
	seen := map[uint32]bool{}
	scan := bufio.NewScanner(bytes.NewReader(raw))
	scan.Buffer(make([]byte, 8192), 1024*1024)
	line := 0
	fail := func() (relicConfig, error) {
		return relicConfig{}, fmt.Errorf("Obelisk.ini line %d: malformed event record", line)
	}
	for scan.Scan() {
		line++
		text := strings.TrimSpace(strings.SplitN(scan.Text(), "//", 2)[0])
		if !strings.HasPrefix(text, "[") {
			continue
		}
		lower := strings.ToLower(text)
		if !strings.HasPrefix(lower, "[zone_") && !strings.HasPrefix(lower, "[obelisk_") && !strings.HasPrefix(lower, "[bothmob_") && !strings.HasPrefix(lower, "[portal_") && !strings.HasPrefix(lower, "[mob]") && !strings.HasPrefix(lower, "[randomrespawn]") && !strings.HasPrefix(lower, "[light]") && !strings.HasPrefix(lower, "[fury]") {
			continue
		}
		n, err := eventNumbers(text)
		if err != nil {
			return fail()
		}
		switch {
		case strings.HasPrefix(lower, "[zone_"):
			if len(n) != 1 || !eventInteger(n[0], 65535) {
				return fail()
			}
			zone = int(n[0])
			active, boss, portal = nil, nil, nil
		case strings.HasPrefix(lower, "[obelisk_"):
			if zone < 0 || len(n) != 8 || !eventInteger(n[0], 65535) || n[0] < 1 || !eventInteger(n[7], 2) {
				return fail()
			}
			ids, err := eventMobIDs(n[4:7])
			if err != nil {
				return fail()
			}
			key := uint32(zone)<<16 | uint32(n[0])
			if seen[key] {
				return fail()
			}
			seen[key] = true
			active = &relic{mapID: uint16(zone), key: key, pos: eventPosition(n[1:4]), initial: uint8(n[7]), owner: uint8(n[7]), last: uint8(n[7])}
			copy(active.mobs[:], ids)
			out.relics = append(out.relics, active)
			boss, portal = nil, nil
		case strings.HasPrefix(lower, "[bothmob_"):
			if len(n) < 10 || !eventInteger(n[0], 65535) || !eventInteger(n[1], 65535) || !eventInteger(n[2], 15) || n[2] < 1 || !eventInteger(n[3], math.MaxUint32) || !eventInteger(n[4], 65535) || !eventInteger(n[5], math.MaxInt32) || len(n) != 9+int(n[2]) {
				return fail()
			}
			ids, err := eventMobIDs(n[9:])
			if err != nil {
				return fail()
			}
			boss = &linkedBoss{key: uint32(n[0]), mapID: uint16(n[1]), mobs: ids, delay: time.Duration(n[3]) * time.Second, portalMap: uint16(n[4]), portalID: int32(n[5]), pos: eventPosition(n[6:9])}
			out.bosses = append(out.bosses, boss)
			active, portal = nil, nil
			zone = -1
		case strings.HasPrefix(lower, "[portal_"):
			// Legacy files can contain a human-readable portal template without
			// dependency rows. It is documentation, not a configured live portal.
			active, boss, portal = nil, nil, nil
			zone = -1
			if len(n) == 0 {
				continue
			}
			if len(n) != 3 || !eventInteger(n[0], 0xfffffff) || !eventInteger(n[1], 65535) || !eventInteger(n[2], math.MaxInt32) {
				return fail()
			}
			portal = &relicPortal{key: 0x10000000 | uint32(n[0]), mapID: uint16(n[1]), portalID: int32(n[2]), country: 3}
			out.portals = append(out.portals, portal)
		case strings.HasPrefix(lower, "[mob]"):
			if active == nil && boss == nil {
				continue
			}
			variants := 3
			if boss != nil {
				variants = len(boss.mobs) + 1
			}
			if len(n) != 5+variants || n[3] < 1 || !eventInteger(n[len(n)-1], 65535) {
				return fail()
			}
			ids, err := eventMobIDs(n[4 : 4+variants])
			if err != nil {
				return fail()
			}
			spawn := &eventSpawn{pos: eventPosition(n[:3]), radius: float32(n[3]), mobs: ids, count: int(n[len(n)-1])}
			if active != nil {
				if len(active.guards) >= 64 {
					return fail()
				}
				active.guards = append(active.guards, spawn)
			} else {
				spawn.count = min(spawn.count, 16)
				if len(boss.guards) >= 32 {
					return fail()
				}
				boss.guards = append(boss.guards, spawn)
			}
		case strings.HasPrefix(lower, "[randomrespawn]"):
			if boss == nil || len(n) != 11 || !eventInteger(n[0], math.MaxUint32) || !eventInteger(n[1], math.MaxUint32) {
				return fail()
			}
			boss.random = true
			boss.randomMin = time.Duration(n[0]) * time.Minute
			boss.randomRange = time.Duration(n[1]) * time.Minute
			for i := range 3 {
				boss.randomPositions[i] = eventPosition(n[2+i*3:])
			}
		case strings.HasPrefix(lower, "[light]"), strings.HasPrefix(lower, "[fury]"):
			if portal == nil {
				continue
			}
			if len(n)%2 != 0 || len(n) > 8 {
				return fail()
			}
			side := 0
			if strings.HasPrefix(lower, "[fury]") {
				side = 1
			}
			for i := 0; i < len(n); i += 2 {
				if !eventInteger(n[i], 65535) || !eventInteger(n[i+1], 65535) || n[i+1] < 1 {
					return fail()
				}
				portal.refs[side] = append(portal.refs[side], uint32(n[i])<<16|uint32(n[i+1]))
			}
		}
	}
	return out, scan.Err()
}
