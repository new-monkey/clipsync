//go:build !windows

package winclip

import "errors"

func SetText(_ string) error {
	return errors.New("clipboard write is only supported on windows")
}
