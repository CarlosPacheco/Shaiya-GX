package studio

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const preferencesFile = "gx-studio.ini"

type Preferences struct {
	DataDir       string
	Database      string
	ServerGameDir string
	Theme         string
}

func LoadPreferences(root string) (Preferences, error) {
	var paths Preferences
	file, err := os.Open(filepath.Join(root, preferencesFile))
	if errors.Is(err, os.ErrNotExist) {
		return paths, nil
	}
	if err != nil {
		return paths, err
	}
	defer file.Close()

	section := ""
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		line := strings.TrimSpace(lines.Text())
		if strings.HasPrefix(line, "[") {
			section = line
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		var target *string
		switch section + strings.TrimSpace(key) {
		case "[Installation]ClientData":
			target = &paths.DataDir
		case "[Installation]ServerGame":
			target = &paths.ServerGameDir
		case "[Installation]Database":
			target = &paths.Database
		case "[Appearance]Theme":
			target = &paths.Theme
		default:
			continue
		}
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, `"`) {
			if value, err = strconv.Unquote(value); err != nil {
				return paths, fmt.Errorf("%s: invalid %s value: %w", preferencesFile, key, err)
			}
		}
		*target = value
	}
	return paths, lines.Err()
}

func SavePreferences(root string, paths Preferences) error {
	data := fmt.Appendf(nil, "[Installation]\nClientData=%q\nDatabase=%q\n", paths.DataDir, paths.Database)
	if paths.ServerGameDir != "" {
		data = fmt.Appendf(data, "ServerGame=%q\n", paths.ServerGameDir)
	}
	if paths.Theme != "" {
		data = fmt.Appendf(data, "\n[Appearance]\nTheme=%s\n", paths.Theme)
	}
	return writeAtomic(filepath.Join(root, preferencesFile), data, 0600)
}
