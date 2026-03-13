//go:build !windows

package servernotify

func HandleProtocolArg(_ string) (bool, error) {
	return false, nil
}

func DiagnoseEnvironment(_ bool) error {
	return nil
}

func EnsureProtocolRegistered(_ string, _ bool) error {
	return nil
}

func NotifyClipReceived(_, _, _, _ string, _ bool) error {
	return nil
}

func SaveClipText(text string) (string, error) {
	return "", nil
}

func ExecutablePath() (string, error) {
	return "", nil
}

func NormalizePathFromArgs(p string) string {
	return p
}

func IsLikelyWSLPath(_ string) bool {
	return false
}
