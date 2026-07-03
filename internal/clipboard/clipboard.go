//go:build !linux && !windows

package clipboard

import "errors"

var ErrNotSupported = errors.New("clipboard not supported on this platform")

func ReadText() (string, error) {
	return "", ErrNotSupported
}

func WriteText(text string) error {
	return ErrNotSupported
}

func IsSupported() bool {
	return false
}
