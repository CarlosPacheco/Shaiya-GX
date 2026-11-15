// Package vehicle reads the external race-specific MON seat bindings.
package vehicle

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

type Bones struct{ Rider, Passenger int }

var RaceNames = [4]string{"Human", "Elf", "Vail", "DeathEater"}

// ParseBones reads race sections containing MON-index=rider,passenger pairs.
// An omitted passenger (or -1) means a single seat.
func ParseBones(raw []byte) (map[uint8]map[uint8]Bones, error) {
	out := map[uint8]map[uint8]Bones{}
	current := -1
	scan := bufio.NewScanner(strings.NewReader(string(raw)))
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := line[1 : len(line)-1]
			current = -1
			for i, race := range RaceNames {
				if name == race {
					current = i
					break
				}
			}
			if current < 0 {
				return nil, fmt.Errorf("unknown vehicle race %q", name)
			}
			if out[uint8(current)] != nil {
				return nil, fmt.Errorf("duplicate vehicle race %q", name)
			}
			out[uint8(current)] = map[uint8]Bones{}
			continue
		}
		pair := strings.SplitN(line, "=", 2)
		if current < 0 || len(pair) != 2 {
			return nil, fmt.Errorf("invalid vehicle binding %q", line)
		}
		index, err := strconv.Atoi(strings.TrimSpace(pair[0]))
		if err != nil || index < 0 || index > 255 {
			return nil, fmt.Errorf("invalid vehicle index %q", pair[0])
		}
		rows := out[uint8(current)]
		if _, ok := rows[uint8(index)]; ok {
			return nil, fmt.Errorf("duplicate vehicle %s/%d", RaceNames[current], index)
		}
		values := strings.Split(pair[1], ",")
		if len(values) < 1 || len(values) > 2 {
			return nil, fmt.Errorf("invalid vehicle seats %q", line)
		}
		rider, err := strconv.Atoi(strings.TrimSpace(values[0]))
		if err != nil || rider < 0 || rider > 1023 {
			return nil, fmt.Errorf("invalid rider bone %q", line)
		}
		passenger := -1
		if len(values) == 2 {
			passenger, err = strconv.Atoi(strings.TrimSpace(values[1]))
			if err != nil || passenger < -1 || passenger > 1023 {
				return nil, fmt.Errorf("invalid passenger bone %q", line)
			}
		}
		if rider == passenger {
			return nil, fmt.Errorf("duplicate seat bone %q", line)
		}
		rows[uint8(index)] = Bones{rider, passenger}
	}
	if err := scan.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
