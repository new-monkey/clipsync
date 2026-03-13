//go:build !windows

package serverui

type noopManager struct{}

func Start(_ Options) (Manager, error) {
	return noopManager{}, nil
}

func (noopManager) Publish(_ ClipEntry) {}

func (noopManager) Close() {}
