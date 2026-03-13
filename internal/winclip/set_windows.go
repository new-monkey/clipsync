//go:build windows

package winclip

import (
	"errors"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

const (
	cfUnicodeText = 13
	gmemMoveable = 0x0002
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procOpenClipboard   = user32.NewProc("OpenClipboard")
	procCloseClipboard  = user32.NewProc("CloseClipboard")
	procEmptyClipboard  = user32.NewProc("EmptyClipboard")
	procSetClipboard    = user32.NewProc("SetClipboardData")
	procGlobalAlloc     = kernel32.NewProc("GlobalAlloc")
	procGlobalLock      = kernel32.NewProc("GlobalLock")
	procGlobalUnlock    = kernel32.NewProc("GlobalUnlock")
)

func SetText(text string) error {
	if err := openClipboardWithRetry(); err != nil {
		return err
	}
	defer procCloseClipboard.Call()

	r, _, _ := procEmptyClipboard.Call()
	if r == 0 {
		return errors.New("EmptyClipboard failed")
	}

	u16 := utf16.Encode([]rune(text + "\x00"))
	sz := uintptr(len(u16) * 2)
	hMem, _, _ := procGlobalAlloc.Call(gmemMoveable, sz)
	if hMem == 0 {
		return errors.New("GlobalAlloc failed")
	}

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return errors.New("GlobalLock failed")
	}
	buf := unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), len(u16))
	copy(buf, u16)
	procGlobalUnlock.Call(hMem)

	r, _, _ = procSetClipboard.Call(cfUnicodeText, hMem)
	if r == 0 {
		return errors.New("SetClipboardData failed")
	}

	return nil
}

func openClipboardWithRetry() error {
	for i := 0; i < 10; i++ {
		r, _, _ := procOpenClipboard.Call(0)
		if r != 0 {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return errors.New("OpenClipboard failed")
}
