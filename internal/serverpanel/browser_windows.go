//go:build windows

package serverpanel

import (
	"fmt"
	"os/exec"
)

func OpenBrowser(url string) error {
	cmd := exec.Command("cmd", "/c", "start", "", url)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser failed: %w", err)
	}
	return nil
}
