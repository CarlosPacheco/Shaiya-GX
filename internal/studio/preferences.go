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

type InstallationPaths struct {
	DataDir  string
	Database string
}

func LoadInstallationPaths(root string) (InstallationPaths, error) {
	var paths InstallationPaths
	file, err := os.Open(filepath.Join(root, preferencesFile))
	if errors.Is(err, os.ErrNotExist) {
		return paths, nil
	}
	if err != nil {
		return paths, err
	}
	defer file.Close()

	inInstallation := false
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		line := strings.TrimSpace(lines.Text())
		if strings.HasPrefix(line, "[") {
			inInstallation = line == "[Installation]"
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !inInstallation || !found {
			continue
		}
		var target *string
		switch strings.TrimSpace(key) {
		case "ClientData":
			target = &paths.DataDir
		case "Database":
			target = &paths.Database
		default:
			continue
		}
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, `"`) {
			if value, err = strconv.Unquote(value); err != nil {
				return paths, fmt.Errorf("%s: invalid %s path: %w", preferencesFile, key, err)
			}
		}
		*target = value
	}
	return paths, lines.Err()
}

func SaveInstallationPaths(root string, paths InstallationPaths) error {
	data := fmt.Appendf(nil, "[Installation]\nClientData=%q\nDatabase=%q\n", paths.DataDir, paths.Database)
	return writeAtomic(filepath.Join(root, preferencesFile), data, 0600)
}
