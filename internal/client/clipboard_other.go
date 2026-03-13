//go:build !windows

package client

import "errors"

func readClipboardText() (string, error) {
	return "", errors.New("clipboard listener is only supported on windows")
}
