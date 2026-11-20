package catalogfiles

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/gameini"
	"github.com/Spelunkern/Shaiya-GX/internal/maprules"
	"github.com/Spelunkern/Shaiya-GX/internal/teleportrunes"
)

var raceStarts = [...]string{"HUGameStartMapPos", "ELGameStartMapPos", "BAGameStartMapPos", "DEGameStartMapPos"}
var positionKeys = [...]string{"RebirthMapPos1", "RebirthMapPos2"}

func ParseMap(raw []byte) (maprules.Catalog, error) {
	ini, err := gameini.Parse(raw)
	if err != nil {
		return maprules.Catalog{}, err
	}
	zones, err := ini.Entries("SET_ZONE_", 16)
	if err != nil {
		return maprules.Catalog{}, err
	}
	var c maprules.Catalog
	for _, zone := range zones {
		row, err := parseZone(zone, ini["DEFAULT_SET_ZONE"])
		if err != nil {
			return maprules.Catalog{}, fmt.Errorf("map %d: %w", zone.Index, err)
		}
		c.Maps = append(c.Maps, row)
	}
	for i, key := range raceStarts {
		p, err := parsePoint(ini["BASE"][strings.ToUpper(key)])
		if err != nil {
			return maprules.Catalog{}, fmt.Errorf("[Base] %s: %w", key, err)
		}
		found := false
		for _, row := range c.Maps {
			if row[0] == int64(p.MapID) && row[3] == "" && row[4] == int64(0) && row[5] == int64(0) {
				found = true
				break
			}
		}
		if !found {
			return maprules.Catalog{}, fmt.Errorf("birth map %d must be shared and allow all levels", p.MapID)
		}
		c.Starts = append(c.Starts, maprules.Row{int64(i), int64(p.MapID), p.X, p.Y, p.Z})
	}
	return c, nil
}

func parseZone(zone gameini.Entry, defaults map[string]string) (maprules.Row, error) {
	fields := maps.Clone(defaults)
	if fields == nil {
		fields = make(map[string]string)
	}
	maps.Copy(fields, zone.Fields)
	r := gameini.Record{Fields: fields}
	get := func(key string, bits int, fallback int64) int64 {
		if r.Text(key) == "" {
			return fallback
		}
		return int64(r.Uint(key, bits))
	}
	weather := get("WeatherType", 16, 0)
	if r.Text("WeatherType") == "" {
		weather = get("WeatherState", 16, 0)
	}
	frequency := get("WeatherFrequency", 16, 0)
	if weather != 0 && r.Text("WeatherFrequency") == "" {
		frequency = get("WeatherDelay", 16, 0) + get("WeatherNoneDelay", 16, 0)
	}
	minLevel, maxLevel := get("MinLevel", 16, 0), get("MaxLevel", 16, 0)
	row := make(maprules.Row, len(maprules.MapFields))
	copy(row, maprules.Row{int64(zone.Index), r.Text("MapType"), r.Text("WarType"), r.Text("CreateType"),
		minLevel, maxLevel, get("CreateMinUser", 16, 0), get("EnterMaxUser", 16, 30), get("ExpireTime", 16, 0), weather, frequency})
	for i := 11; i < 17; i++ {
		bits := 16
		if i >= 14 {
			bits = 31
		}
		row[i] = get(maprules.MapFields[i].Name, bits, 0)
	}
	if r.Err != nil {
		return nil, r.Err
	}
	if weather > 2 || frequency > 65535 || maxLevel != 0 && minLevel > maxLevel {
		return nil, fmt.Errorf("invalid weather or level range")
	}
	for i, key := range positionKeys {
		value, ok := zone.Fields[strings.ToUpper(key)]
		if !ok {
			value, ok = zone.Fields["REBIRTHMAPPOS"]
		}
		if !ok {
			value = r.Text(key)
			if value == "" {
				value = r.Text("RebirthMapPos")
			}
		}
		if value == "" {
			continue
		}
		parts := gameini.Fields(value)
		if len(parts) == 4 && parts[0] == "-1" {
			continue
		}
		p, err := parsePoint(value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		copy(row[17+4*i:], maprules.Row{int64(p.MapID), p.X, p.Y, p.Z})
	}
	times := gameini.Fields(r.Text("CreateTime"))
	if len(times)%3 != 0 || len(times) > 21 {
		return nil, fmt.Errorf("CreateTime requires up to seven (day, hour, duration) tuples")
	}
	for i, v := range times {
		n, err := strconv.ParseUint(v, 10, 16)
		if err != nil || i%3 == 0 && n > 7 || i%3 == 1 && n > 23 {
			return nil, fmt.Errorf("invalid CreateTime component %q", v)
		}
		row[25+i] = int64(n)
	}
	return row, nil
}

// EncodeMap retains fields outside the editor and updates the native keys.
func EncodeMap(c maprules.Catalog, original []byte) ([]byte, error) {
	ini, err := gameini.Parse(original)
	if err != nil {
		return nil, err
	}
	if ini["BASE"] == nil {
		ini["BASE"] = map[string]string{}
	}
	base := ini["BASE"]
	base["MAPCOUNT"] = strconv.Itoa(len(c.Maps))
	point := func(row maprules.Row) (string, error) {
		m, mok := row[0].(int64)
		x, xok := row[1].(float64)
		y, yok := row[2].(float64)
		z, zok := row[3].(float64)
		if !mok || !xok || !yok || !zok || m < 0 || m > 65535 {
			return "", fmt.Errorf("expected a complete map position")
		}
		return formatPoint(teleportrunes.Destination{MapID: uint16(m), X: x, Y: y, Z: z}), nil
	}
	for _, row := range c.Starts {
		if len(row) != len(maprules.StartFields) {
			return nil, fmt.Errorf("invalid birth position")
		}
		race, ok := row[0].(int64)
		if !ok || race < 0 || race >= int64(len(raceStarts)) {
			return nil, fmt.Errorf("invalid race in birth position")
		}
		pos, err := point(row[1:])
		if err != nil {
			return nil, err
		}
		base[strings.ToUpper(raceStarts[race])] = pos
	}
	var order []string
	for _, row := range c.Maps {
		if len(row) != len(maprules.MapFields) {
			return nil, fmt.Errorf("invalid map row")
		}
		name := fmt.Sprintf("SET_ZONE_%d", row[0])
		order = append(order, name)
		m := ini[name]
		if m == nil {
			m = map[string]string{}
			ini[name] = m
		}
		keys := [...]string{"MapType", "WarType", "CreateType", "MinLevel", "MaxLevel", "CreateMinUser", "EnterMaxUser", "ExpireTime", "WeatherState", "WeatherFrequency"}
		for i, key := range keys {
			m[strings.ToUpper(key)] = fmt.Sprint(row[i+1])
		}
		delete(m, "WEATHERTYPE")
		for i := 11; i < 17; i++ {
			m[strings.ToUpper(maprules.MapFields[i].Name)] = fmt.Sprint(row[i])
		}
		for i, key := range positionKeys {
			start := 17 + i*4
			var pos string
			if row[start] == nil {
				if row[start+1] != nil || row[start+2] != nil || row[start+3] != nil {
					return nil, fmt.Errorf("%s %s requires a complete position", name, key)
				}
				pos = "-1, 0, 0, 0"
			} else {
				pos, err = point(row[start:])
				if err != nil {
					return nil, fmt.Errorf("%s %s: %w", name, key, err)
				}
			}
			m[strings.ToUpper(key)] = pos
		}
		var times []string
		for i := 25; i < len(row); i += 3 {
			if row[i] != nil {
				times = append(times, fmt.Sprintf("(%v, %v, %v)", row[i], row[i+1], row[i+2]))
			}
		}
		m["CREATETIME"] = strings.Join(times, " ")
	}
	var b strings.Builder
	b.WriteString("; Map rules. Coordinates use (map, x, y, z); -1 disables a rebirth.\n; WeatherFrequency and ExpireTime are in minutes. Level 0 means unrestricted.\n")
	write := func(section string) {
		m, ok := ini[section]
		if !ok {
			return
		}
		fmt.Fprintf(&b, "\n[%s]\n", section)
		for _, key := range slices.Sorted(maps.Keys(m)) {
			value := m[key]
			if strings.ContainsAny(value, ";\r\n") || strings.Contains(value, "//") || strings.HasPrefix(value, "\"") {
				value = strconv.Quote(value)
			}
			fmt.Fprintf(&b, "%s=%s\n", key, value)
		}
		delete(ini, section)
	}
	write("BASE")
	write("DEFAULT_SET_ZONE")
	for _, section := range order {
		write(section)
	}
	for _, section := range slices.Sorted(maps.Keys(ini)) {
		if !strings.HasPrefix(section, "SET_ZONE_") {
			write(section)
		}
	}
	out := []byte(b.String())
	_, err = ParseMap(out)
	return out, err
}
