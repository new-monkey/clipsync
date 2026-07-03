//go:build !windows

package publisher

import "errors"

func readClipboardText() (string, error) {
	return "", errors.New("clipboard reader is only supported on windows")
}