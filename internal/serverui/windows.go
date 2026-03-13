//go:build windows

package serverui

import (
	"fmt"
	"sync"
	"time"

	"clipsync/internal/winclip"

	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

type manager struct {
	mu sync.Mutex

	entries       []ClipEntry
	maxHistory    int
	detailVisible bool

	mw          *walk.MainWindow
	historyList *walk.ListBox
	detailEdit  *walk.TextEdit
	statusLabel *walk.Label
	toggleBtn   *walk.PushButton
}

func Start(opts Options) (Manager, error) {
	if opts.MaxHistory <= 0 {
		opts.MaxHistory = 200
	}
	if opts.Title == "" {
		opts.Title = "ClipSync Server"
	}

	m := &manager{
		maxHistory:    opts.MaxHistory,
		detailVisible: true,
	}

	errCh := make(chan error, 1)
	m.run(opts, errCh)
	if err := <-errCh; err != nil {
		return nil, err
	}

	return m, nil
}

func (m *manager) run(opts Options, errCh chan<- error) {
	var mw *walk.MainWindow
	var historyList *walk.ListBox
	var detailEdit *walk.TextEdit
	var statusLabel *walk.Label
	var toggleBtn *walk.PushButton

	createErr := (declarative.MainWindow{
		AssignTo: &mw,
		Title:    opts.Title,
		MinSize:  declarative.Size{Width: 420, Height: 280},
		Size:     declarative.Size{Width: 560, Height: 420},
		Layout:   declarative.VBox{},
		Children: []declarative.Widget{
			declarative.Composite{
				Layout: declarative.HBox{},
				Children: []declarative.Widget{
					declarative.PushButton{
						Text: "复制最新内容",
						OnClicked: func() {
							m.copyLatestToClipboard()
						},
					},
					declarative.PushButton{
						AssignTo: &toggleBtn,
						Text:     "折叠详情",
						OnClicked: func() {
							m.toggleDetailVisibility()
						},
					},
					declarative.PushButton{
						Text: "清空历史",
						OnClicked: func() {
							m.clearHistory()
						},
					},
				},
			},
			declarative.ListBox{
				AssignTo: &historyList,
				Model:    []string{},
				OnCurrentIndexChanged: func() {
					m.updateDetailFromSelection()
				},
			},
			declarative.TextEdit{
				AssignTo: &detailEdit,
				ReadOnly: true,
				VScroll:  true,
			},
			declarative.Label{
				AssignTo: &statusLabel,
				Text:     "等待接收剪贴板内容...",
			},
		},
	}).Create()
	if createErr != nil {
		errCh <- createErr
		return
	}

	m.mu.Lock()
	m.mw = mw
	m.historyList = historyList
	m.detailEdit = detailEdit
	m.statusLabel = statusLabel
	m.toggleBtn = toggleBtn
	m.mu.Unlock()

	if opts.AlwaysOnTop {
		setTopMost(mw, true)
	}

	errCh <- nil
	mw.Run()
}

func (m *manager) Publish(entry ClipEntry) {
	m.mu.Lock()
	m.entries = append(m.entries, entry)
	if len(m.entries) > m.maxHistory {
		m.entries = m.entries[len(m.entries)-m.maxHistory:]
	}
	mw := m.mw
	m.mu.Unlock()

	if mw == nil {
		return
	}

	mw.Synchronize(func() {
		m.refreshListUI(true)
	})
}

func (m *manager) Close() {
	m.mu.Lock()
	mw := m.mw
	m.mu.Unlock()
	if mw == nil {
		return
	}
	mw.Synchronize(func() {
		mw.Close()
	})
}

func (m *manager) copyLatestToClipboard() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.entries) == 0 {
		m.statusLabel.SetText("暂无可复制内容")
		return
	}

	idx := m.historyList.CurrentIndex()
	if idx < 0 || idx >= len(m.entries) {
		idx = len(m.entries) - 1
	}

	if err := winclip.SetText(m.entries[idx].Text); err != nil {
		m.statusLabel.SetText("复制失败: " + err.Error())
		return
	}
	m.statusLabel.SetText("已复制到剪贴板")
}

func (m *manager) toggleDetailVisibility() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.detailVisible = !m.detailVisible
	m.detailEdit.SetVisible(m.detailVisible)
	if m.detailVisible {
		m.toggleBtn.SetText("折叠详情")
	} else {
		m.toggleBtn.SetText("展开详情")
	}
}

func (m *manager) clearHistory() {
	m.mu.Lock()
	m.entries = nil
	m.mu.Unlock()
	m.refreshListUI(false)
}

func (m *manager) refreshListUI(selectLast bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	titles := make([]string, len(m.entries))
	for i, e := range m.entries {
		titles[i] = fmt.Sprintf("[%s] %s | %dB", e.ReceivedAt.Format("15:04:05"), e.Machine, e.Bytes)
	}

	m.historyList.SetModel(titles)
	if len(titles) == 0 {
		m.detailEdit.SetText("")
		m.statusLabel.SetText("历史已清空")
		return
	}
	if selectLast {
		m.historyList.SetCurrentIndex(len(titles) - 1)
	}
	m.updateDetailFromSelectionLocked()
	m.statusLabel.SetText(fmt.Sprintf("历史记录: %d 条", len(titles)))
}

func (m *manager) updateDetailFromSelection() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateDetailFromSelectionLocked()
}

func (m *manager) updateDetailFromSelectionLocked() {
	if len(m.entries) == 0 {
		m.detailEdit.SetText("")
		return
	}
	idx := m.historyList.CurrentIndex()
	if idx < 0 || idx >= len(m.entries) {
		idx = len(m.entries) - 1
	}
	e := m.entries[idx]
	content := fmt.Sprintf(
		"时间: %s\r\n来源机器: %s\r\n字节数: %d\r\nSHA256: %s\r\n\r\n%s",
		e.ReceivedAt.Format(time.RFC3339),
		e.Machine,
		e.Bytes,
		e.SHA256,
		e.Text,
	)
	m.detailEdit.SetText(content)
}

func setTopMost(mw *walk.MainWindow, topMost bool) {
	if mw == nil {
		return
	}
	insertAfter := win.HWND_NOTOPMOST
	if topMost {
		insertAfter = win.HWND_TOPMOST
	}
	win.SetWindowPos(
		win.HWND(mw.Handle()),
		insertAfter,
		0,
		0,
		0,
		0,
		win.SWP_NOMOVE|win.SWP_NOSIZE|win.SWP_NOACTIVATE,
	)
}
