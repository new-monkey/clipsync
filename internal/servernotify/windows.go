//go:build windows

package servernotify

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"

	"clipsync/internal/winclip"
)

const protocolScheme = "clipsync-copy"

func HandleProtocolArg(arg string) (bool, error) {
	a := strings.TrimSpace(arg)
	if !strings.HasPrefix(strings.ToLower(a), protocolScheme+":") {
		return false, nil
	}
	u, err := url.Parse(a)
	if err != nil {
		return true, err
	}
	filePath := u.Query().Get("file")
	if filePath == "" {
		return true, fmt.Errorf("missing file query")
	}
	b, err := os.ReadFile(filePath)
	if err != nil {
		return true, err
	}
	if err := winclip.SetText(string(b)); err != nil {
		return true, err
	}
	return true, nil
}

func DiagnoseEnvironment(debug bool) error {
	if !debug {
		return nil
	}
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "$PSVersionTable.PSVersion.ToString() | Out-String").CombinedOutput()
	if err != nil {
		return fmt.Errorf("powershell unavailable: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	log.Printf("Notify debug: PowerShell version: %s", strings.TrimSpace(string(out)))

	checkScript := "[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType=WindowsRuntime] > $null;"
	checkScript += "[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType=WindowsRuntime] > $null;"
	checkScript += "Write-Output 'WINRT_OK'"
	outText, err := runPowerShellEncoded(checkScript)
	if err != nil {
		return err
	}
	log.Printf("Notify debug: WinRT probe output: %s", strings.TrimSpace(outText))
	return nil
}

func EnsureProtocolRegistered(exePath string, debug bool) error {
	commandValue := fmt.Sprintf("\"%s\" \"%%1\"", exePath)
	commands := [][]string{
		{"add", `HKCU\Software\Classes\` + protocolScheme, "/ve", "/d", "URL:ClipSync Copy Protocol", "/f"},
		{"add", `HKCU\Software\Classes\` + protocolScheme, "/v", "URL Protocol", "/d", "", "/f"},
		{"add", `HKCU\Software\Classes\` + protocolScheme + `\shell\open\command`, "/ve", "/d", commandValue, "/f"},
	}
	for _, args := range commands {
		cmd := exec.Command("reg", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("register protocol failed: %v (%s)", err, strings.TrimSpace(string(out)))
		}
		if debug {
			log.Printf("Notify debug: reg %s => %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func NotifyClipReceived(appID, title, text, filePath string, debug bool) error {
	if appID == "" {
		appID = "PowerShell"
	}
	preview := normalizePreview(text)
	copyURI := protocolScheme + ":?file=" + url.QueryEscape(filePath)

	xmlPayload := buildToastXML(title, preview, copyURI)
	xmlB64 := base64.StdEncoding.EncodeToString([]byte(xmlPayload))

	appIDs := []string{appID, "PowerShell", "Windows.PowerShell"}
	tried := map[string]bool{}
	var lastErr error
	for _, candidate := range appIDs {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || tried[candidate] {
			continue
		}
		tried[candidate] = true
		if err := notifyWithAppID(candidate, xmlB64, debug); err != nil {
			if debug {
				log.Printf("Notify debug: appID=%s failed: %v", candidate, err)
			}
			lastErr = err
			continue
		}
		if debug {
			log.Printf("Notify debug: appID=%s delivered", candidate)
		}
		return nil
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("toast failed: no app id available")
}

func notifyWithAppID(appID, xmlB64 string, debug bool) error {

	script := "$appId='" + escapePS(appID) + "';"
	script += "$xmlB64='" + xmlB64 + "';"
	script += "[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType=WindowsRuntime] > $null;"
	script += "[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType=WindowsRuntime] > $null;"
	script += "$xmlString=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($xmlB64));"
	script += "$xml=New-Object Windows.Data.Xml.Dom.XmlDocument;"
	script += "$xml.LoadXml($xmlString);"
	script += "$toast=[Windows.UI.Notifications.ToastNotification]::new($xml);"
	script += "[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($appId).Show($toast);"

	out, err := runPowerShellEncoded(script)
	if err != nil {
		return fmt.Errorf("toast failed for appId=%s: %v (%s)", appID, err, strings.TrimSpace(string(out)))
	}
	if debug {
		log.Printf("Notify debug: appID=%s powershell output=%s", appID, strings.TrimSpace(out))
	}
	return nil
}

func runPowerShellEncoded(script string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-EncodedCommand", encodePowerShell(script))
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func SaveClipText(text string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "ClipSync", "received")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, "clip-*.txt")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.WriteString(text); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func buildToastXML(title, body, copyURI string) string {
	return fmt.Sprintf("<toast><visual><binding template='ToastGeneric'><text>%s</text><text>%s</text></binding></visual><actions><action content='复制到剪贴板' activationType='protocol' arguments='%s'/></actions></toast>",
		xmlEscape(title),
		xmlEscape(body),
		xmlEscape(copyURI),
	)
}

func normalizePreview(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if s == "" {
		return "(空文本，点击按钮也会复制空字符串)"
	}
	r := []rune(s)
	if len(r) > 120 {
		return string(r[:120]) + "..."
	}
	return s
}

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func escapePS(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func encodePowerShell(script string) string {
	u := utf16.Encode([]rune(script))
	b := make([]byte, len(u)*2)
	for i, v := range u {
		b[i*2] = byte(v)
		b[i*2+1] = byte(v >> 8)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func ExecutablePath() (string, error) {
	return os.Executable()
}

func NormalizePathFromArgs(p string) string {
	p = strings.Trim(p, "\"")
	if strings.HasPrefix(p, `\\?\`) {
		p = strings.TrimPrefix(p, `\\?\`)
	}
	return filepath.Clean(p)
}

func IsLikelyWSLPath(p string) bool {
	pl := strings.ToLower(strings.TrimSpace(p))
	if strings.HasPrefix(pl, `\\wsl$\`) {
		return true
	}
	if strings.HasPrefix(pl, `\\wsl.localhost\`) {
		return true
	}
	return false
}

func IsClipboardUnavailable(err error) bool {
	if err == nil {
		return false
	}
	if errno, ok := err.(syscall.Errno); ok {
		return errno != 0
	}
	return false
}
