//go:build !windows

package subscriber

import "errors"

func writeClipboardText(_ string) error {
	return errors.New("clipboard writer is only supported on windows")
}