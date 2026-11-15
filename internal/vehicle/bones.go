// Package vehicle reads the external race-specific MON seat bindings.
package vehicle

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

type Bones struct{ Rider, Passenger int }

var RaceFolders = [4]string{"Human", "Elf", "Vail", "DeathEater"}

// ParseBones accepts [MON index] sections with rider and passenger indices.
// An omitted passenger (or -1) means a single seat. Rider is mandatory.
func ParseBones(raw []byte) (map[uint8]Bones, error) {
	out := map[uint8]Bones{}
	current := -1
	seen := map[string]bool{}
	scan := bufio.NewScanner(strings.NewReader(string(raw)))
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			n, e := strconv.Atoi(line[1 : len(line)-1])
			if e != nil || n < 0 || n > 255 {
				return nil, fmt.Errorf("invalid vehicle section %q", line)
			}
			if _, exists := out[uint8(n)]; exists {
				return nil, fmt.Errorf("duplicate vehicle %d", n)
			}
			current = n
			out[uint8(n)] = Bones{-1, -1}
			continue
		}
		pair := strings.SplitN(line, "=", 2)
		if current < 0 || len(pair) != 2 {
			return nil, fmt.Errorf("invalid vehicle binding %q", line)
		}
		key := strings.TrimSpace(pair[0])
		n, e := strconv.Atoi(strings.TrimSpace(pair[1]))
		tag := fmt.Sprintf("%d/%s", current, key)
		if e != nil || n < -1 || n > 1023 || seen[tag] {
			return nil, fmt.Errorf("invalid vehicle bone %q", line)
		}
		seen[tag] = true
		b := out[uint8(current)]
		switch key {
		case "rider":
			b.Rider = n
		case "passenger":
			b.Passenger = n
		default:
			return nil, fmt.Errorf("unknown vehicle key %q", key)
		}
		out[uint8(current)] = b
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}
	for i, b := range out {
		if b.Rider < 0 || b.Rider == b.Passenger {
			return nil, fmt.Errorf("invalid seats for vehicle %d", i)
		}
	}
	return out, nil
}
