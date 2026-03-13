//go:build !windows

package serverpanel

func OpenBrowser(_ string) error {
	return nil
}
