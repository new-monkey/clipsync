//go:build !windows

package publisher

import (
	"errors"
	"os/exec"
	"strings"
)

var clipboardCmd string

func init() {
	if _, err := exec.LookPath("xclip"); err == nil {
		clipboardCmd = "xclip"
	} else if _, err := exec.LookPath("xsel"); err == nil {
		clipboardCmd = "xsel"
	} else if _, err := exec.LookPath("wl-paste"); err == nil {
		clipboardCmd = "wl-paste"
	}
}

func readClipboardText() (string, error) {
	if clipboardCmd == "" {
		return "", errors.New("clipboard reader requires xclip, xsel, or wl-paste to be installed")
	}

	var cmd *exec.Cmd
	switch clipboardCmd {
	case "xclip":
		cmd = exec.Command("xclip", "-o", "-selection", "clipboard")
	case "xsel":
		cmd = exec.Command("xsel", "-o", "-b")
	case "wl-paste":
		cmd = exec.Command("wl-paste", "--no-newline")
	default:
		return "", errors.New("unsupported clipboard tool")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.New("clipboard read failed: " + string(output))
	}

	return strings.TrimSpace(string(output)), nil
}
