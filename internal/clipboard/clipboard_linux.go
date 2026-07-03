//go:build linux

package clipboard

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
		clipboardCmd = "wl"
	}
}

func ReadText() (string, error) {
	if clipboardCmd == "" {
		return "", errors.New("clipboard requires xclip, xsel, or wl-paste to be installed")
	}

	var cmd *exec.Cmd
	switch clipboardCmd {
	case "xclip":
		cmd = exec.Command("xclip", "-o", "-selection", "clipboard")
	case "xsel":
		cmd = exec.Command("xsel", "-o", "-b")
	case "wl":
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

func WriteText(text string) error {
	if clipboardCmd == "" {
		return errors.New("clipboard requires xclip, xsel, or wl-copy to be installed")
	}

	var cmd *exec.Cmd
	switch clipboardCmd {
	case "xclip":
		cmd = exec.Command("xclip", "-i", "-selection", "clipboard")
	case "xsel":
		cmd = exec.Command("xsel", "-i", "-b")
	case "wl":
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

func IsSupported() bool {
	return clipboardCmd != ""
}