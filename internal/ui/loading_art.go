package ui

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
)

// Loading pictures. Game.exe 0x4b7b40 rolls among the numbered pictures and
// shows loading64.jpg for map 64 only. GX generalises both rules:
//
//   - every interface file named random_loadingNN.png is part of the random
//     pool, however many there are (see RandomLoadingNames);
//   - MapScreens.INI, beside sysmsg-uni.txt, lists "MapID=file" pairs; a map
//     with a valid entry always shows that interface picture.

// MapScreensFile is the configuration file name inside the data folder.
const MapScreensFile = "MapScreens.INI"

// ParseMapScreens reads "MapID=file" lines. Section headers, blank lines and
// comments (; or #) are ignored, and so is any entry whose ID is not a map
// number or whose file name is not a plain interface file name.
func ParseMapScreens(raw []byte) map[uint16]string {
	screens := map[uint16]string{}
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == ';' || line[0] == '#' || line[0] == '[' {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimSpace(key), 10, 16)
		name := strings.TrimSpace(value)
		if err != nil || name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
			continue
		}
		screens[uint16(id)] = name
	}
	return screens
}

var randomLoadingName = regexp.MustCompile(`(?i)^random_loading\d+\.png$`)

// RandomLoadingNames keeps the names that belong to the random pool.
func RandomLoadingNames(files []string) []string {
	var names []string
	for _, name := range files {
		if randomLoadingName.MatchString(name) {
			names = append(names, name)
		}
	}
	return names
}
