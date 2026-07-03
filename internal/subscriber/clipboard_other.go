//go:build !windows

package subscriber

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
	} else if _, err := exec.LookPath("wl-copy"); err == nil {
		clipboardCmd = "wl-copy"
	}
}

func writeClipboardText(text string) error {
	if clipboardCmd == "" {
		return errors.New("clipboard writer requires xclip, xsel, or wl-copy to be installed")
	}

	var cmd *exec.Cmd
	switch clipboardCmd {
	case "xclip":
		cmd = exec.Command("xclip", "-i", "-selection", "clipboard")
	case "xsel":
		cmd = exec.Command("xsel", "-i", "-b")
	case "wl-copy":
		cmd = exec.Command("wl-copy")
	default:
		return errors.New("unsupported clipboard tool")
	}

	cmd.Stdin = strings.NewReader(text)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New("clipboard write failed: " + string(output))
	}

	return nil
}