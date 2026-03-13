package config

import (
	"os"
	"path/filepath"
)

func ResolveDefaultConfigPath(fileName string) string {
	if fileName == "" {
		return ""
	}

	candidates := []string{}

	if exePath, err := os.Executable(); err == nil && exePath != "" {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, "configs", fileName),
			filepath.Join(exeDir, fileName),
		)
	}

	if wd, err := os.Getwd(); err == nil && wd != "" {
		candidates = append(candidates,
			filepath.Join(wd, "configs", fileName),
			filepath.Join(wd, fileName),
		)
	}

	seen := map[string]bool{}
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}

	return ""
}
