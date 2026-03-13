package serverpanel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"clipsync/internal/winclip"
)

type Entry struct {
	ID         int64     `json:"id"`
	ReceivedAt time.Time `json:"received_at"`
	Machine    string    `json:"machine"`
	Bytes      int       `json:"bytes"`
	SHA256     string    `json:"sha256"`
	Text       string    `json:"text"`
}

type Panel struct {
	mu         sync.RWMutex
	maxHistory int
	nextID     int64
	entries    []Entry
}

func New(maxHistory int) *Panel {
	if maxHistory <= 0 {
		maxHistory = 200
	}
	return &Panel{maxHistory: maxHistory}
}

func (p *Panel) Add(machine string, bytes int, sha256, text string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.nextID++
	e := Entry{
		ID:         p.nextID,
		ReceivedAt: time.Now(),
		Machine:    machine,
		Bytes:      bytes,
		SHA256:     sha256,
		Text:       text,
	}
	p.entries = append(p.entries, e)
	if len(p.entries) > p.maxHistory {
		p.entries = p.entries[len(p.entries)-p.maxHistory:]
	}
}

func (p *Panel) listDesc() []Entry {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]Entry, len(p.entries))
	copy(out, p.entries)
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out
}

func (p *Panel) copyLatest() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.entries) == 0 {
		return fmt.Errorf("no entries")
	}
	return winclip.SetText(p.entries[len(p.entries)-1].Text)
}

func (p *Panel) latestEntry() (Entry, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.entries) == 0 {
		return Entry{}, fmt.Errorf("no entries")
	}
	return p.entries[len(p.entries)-1], nil
}

func (p *Panel) copyByID(id int64) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for i := len(p.entries) - 1; i >= 0; i-- {
		if p.entries[i].ID == id {
			return winclip.SetText(p.entries[i].Text)
		}
	}
	return fmt.Errorf("entry not found")
}

func (p *Panel) entryByID(id int64) (Entry, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for i := len(p.entries) - 1; i >= 0; i-- {
		if p.entries[i].ID == id {
			return p.entries[i], nil
		}
	}
	return Entry{}, fmt.Errorf("entry not found")
}

func RegisterHandlers(mux *http.ServeMux, panel *Panel) {
	mux.HandleFunc("/panel", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(panelHTML))
	})

	mux.HandleFunc("/panel/api/history", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"items": panel.listDesc()})
	})

	mux.HandleFunc("/panel/api/copy-latest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		entry, err := panel.latestEntry()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := panel.copyLatest(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":          true,
			"id":          entry.ID,
			"bytes":       entry.Bytes,
			"received_at": entry.ReceivedAt,
		})
	})

	mux.HandleFunc("/panel/api/copy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		idStr := r.URL.Query().Get("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		entry, err := panel.entryByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := panel.copyByID(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":          true,
			"id":          entry.ID,
			"bytes":       entry.Bytes,
			"received_at": entry.ReceivedAt,
		})
	})
}

const panelHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>ClipSync 面板</title>
  <style>
    :root {
      --bg: #f2f5f7;
      --card: #ffffff;
      --ink: #142028;
      --sub: #51606b;
      --line: #d9e2e7;
      --brand: #0f7b6c;
      --brand2: #e2f4f1;
    }
    body { margin: 0; font-family: "Segoe UI", "Microsoft YaHei", sans-serif; background: linear-gradient(140deg, #edf4f6, #f7f9fa); color: var(--ink); }
    .wrap { max-width: 900px; margin: 20px auto; padding: 0 12px; }
    .bar { display:flex; gap:10px; align-items:center; margin-bottom:12px; }
    button { border:1px solid var(--line); background: var(--card); padding: 8px 12px; border-radius: 10px; cursor: pointer; }
    button.primary { background: var(--brand); color:#fff; border-color: var(--brand); }
    .meta { color: var(--sub); font-size: 13px; }
    .meta.ok { color: #0b6b44; }
    .meta.err { color: #b42318; }
    .toast { position: fixed; top: 12px; right: 12px; max-width: 320px; background: #112f2a; color: #fff; padding: 10px 12px; border-radius: 10px; box-shadow: 0 8px 22px rgba(0,0,0,.16); opacity: 0; transform: translateY(-6px); transition: all .18s ease; pointer-events: none; }
    .toast.show { opacity: 1; transform: translateY(0); }
    .toast.err { background: #7f1d1d; }
    .list { display:flex; flex-direction:column; gap:10px; }
    .item { background: var(--card); border:1px solid var(--line); border-radius: 12px; overflow:hidden; }
    .head { display:flex; justify-content:space-between; align-items:center; padding:10px 12px; background: var(--brand2); }
    .title { font-weight: 600; }
    .acts { display:flex; gap:8px; }
    .detail { display:none; padding:10px 12px; white-space: pre-wrap; word-break: break-word; max-height: 280px; overflow:auto; border-top:1px solid var(--line); }
    .detail.show { display:block; }
    .empty { padding:18px; color:var(--sub); background: var(--card); border:1px dashed var(--line); border-radius: 12px; }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="bar">
      <button class="primary" id="copyLatest">复制最新内容</button>
      <button id="refresh">刷新</button>
      <span class="meta" id="status">准备就绪</span>
    </div>
    <div class="list" id="list"></div>
  </div>
  <div class="toast" id="toast"></div>
  <script>
    const manualState = new Map(); // id -> true(open) | false(closed), only set by user actions
    let autoExpandedLatestId = null;

    async function getHistory() {
      const resp = await fetch('/panel/api/history');
      if (!resp.ok) throw new Error('history fetch failed');
      return await resp.json();
    }

    async function copyLatest() {
      const btn = document.getElementById('copyLatest');
      const oldText = btn.textContent;
      btn.disabled = true;
      btn.textContent = '复制中...';
      const resp = await fetch('/panel/api/copy-latest', { method: 'POST' });
      btn.disabled = false;
      btn.textContent = oldText;
      if (!resp.ok) throw new Error(await resp.text());
      const data = await resp.json();
      const ts = new Date(data.received_at).toLocaleTimeString();
      setStatus('复制成功: ID=' + data.id + ' | ' + data.bytes + 'B | ' + ts, false);
      showToast('已复制最新内容到剪贴板', false);
    }

    async function copyById(id, btn) {
      const oldText = btn.textContent;
      btn.disabled = true;
      btn.textContent = '复制中...';
      const resp = await fetch('/panel/api/copy?id=' + encodeURIComponent(id), { method: 'POST' });
      btn.disabled = false;
      btn.textContent = oldText;
      if (!resp.ok) throw new Error(await resp.text());
      const data = await resp.json();
      const ts = new Date(data.received_at).toLocaleTimeString();
      setStatus('复制成功: ID=' + data.id + ' | ' + data.bytes + 'B | ' + ts, false);
      showToast('复制成功，ID=' + data.id, false);
    }

    function setStatus(msg, isErr) {
      const el = document.getElementById('status');
      el.textContent = msg;
      el.className = 'meta ' + (isErr ? 'err' : 'ok');
    }

    let toastTimer = null;
    function showToast(msg, isErr) {
      const t = document.getElementById('toast');
      t.textContent = msg;
      t.className = 'toast show' + (isErr ? ' err' : '');
      if (toastTimer) clearTimeout(toastTimer);
      toastTimer = setTimeout(() => {
        t.className = 'toast' + (isErr ? ' err' : '');
      }, 1800);
    }

    function render(items) {
      const list = document.getElementById('list');
      list.innerHTML = '';
      if (!items || items.length === 0) {
        autoExpandedLatestId = null;
        manualState.clear();
        const d = document.createElement('div');
        d.className = 'empty';
        d.textContent = '暂无历史记录，等待客户端推送...';
        list.appendChild(d);
        return;
      }

      // Keep only manual states that still exist in current history.
      const alive = new Set(items.map(it => it.id));
      for (const id of manualState.keys()) {
        if (!alive.has(id)) {
          manualState.delete(id);
        }
      }

      // Auto-expand only the current latest item.
      autoExpandedLatestId = items[0].id;

      for (let i = 0; i < items.length; i++) {
        const it = items[i];
        const node = document.createElement('div');
        node.className = 'item';

        const head = document.createElement('div');
        head.className = 'head';

        const title = document.createElement('div');
        title.className = 'title';
        const t = new Date(it.received_at);
        title.textContent = '[' + t.toLocaleTimeString() + '] ' + it.machine + ' | ' + it.bytes + 'B';

        const acts = document.createElement('div');
        acts.className = 'acts';

        const btnCopy = document.createElement('button');
        btnCopy.textContent = '复制';
        btnCopy.onclick = async () => {
          try { await copyById(it.id, btnCopy); } catch (e) { setStatus('复制失败: ' + e.message, true); showToast('复制失败: ' + e.message, true); }
        };

        const btnToggle = document.createElement('button');
        const shouldOpen = manualState.has(it.id) ? manualState.get(it.id) : (it.id === autoExpandedLatestId);
        btnToggle.textContent = shouldOpen ? '收起' : '展开';

        acts.appendChild(btnCopy);
        acts.appendChild(btnToggle);

        head.appendChild(title);
        head.appendChild(acts);

        const detail = document.createElement('div');
        detail.className = shouldOpen ? 'detail show' : 'detail';
        detail.textContent = '时间: ' + it.received_at + '\n机器: ' + it.machine + '\n字节: ' + it.bytes + '\nSHA256: ' + it.sha256 + '\n\n' + it.text;

        btnToggle.onclick = () => {
          const show = detail.classList.toggle('show');
          // Manual operation always wins and is preserved across refresh.
          manualState.set(it.id, show);
          btnToggle.textContent = show ? '收起' : '展开';
        };

        node.appendChild(head);
        node.appendChild(detail);
        list.appendChild(node);
      }
    }

    async function refresh() {
      try {
        const data = await getHistory();
        render(data.items || []);
      } catch (e) {
        setStatus('刷新失败: ' + e.message, true);
      }
    }

    document.getElementById('copyLatest').onclick = async () => {
      try { await copyLatest(); } catch (e) { setStatus('复制失败: ' + e.message, true); showToast('复制失败: ' + e.message, true); }
    };
    document.getElementById('refresh').onclick = refresh;

    refresh();
    setInterval(refresh, 1200);
  </script>
</body>
</html>`
