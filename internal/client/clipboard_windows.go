//go:build windows

package client

import (
	"errors"
	"syscall"
	"time"
	"unsafe"
)

const cfUnicodeText = 13

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procOpenClipboard  = user32.NewProc("OpenClipboard")
	procCloseClipboard = user32.NewProc("CloseClipboard")
	procGetClipboard   = user32.NewProc("GetClipboardData")
	procGlobalLock     = kernel32.NewProc("GlobalLock")
	procGlobalUnlock   = kernel32.NewProc("GlobalUnlock")
	procGlobalSize     = kernel32.NewProc("GlobalSize")
)

func readClipboardText() (string, error) {
	if err := openClipboardWithRetry(); err != nil {
		return "", err
	}
	defer procCloseClipboard.Call()

	h, _, _ := procGetClipboard.Call(uintptr(cfUnicodeText))
	if h == 0 {
		return "", errors.New("clipboard has no unicode text")
	}

	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return "", errors.New("GlobalLock failed")
	}
	defer procGlobalUnlock.Call(h)

	sizeBytes, _, _ := procGlobalSize.Call(h)
	if sizeBytes == 0 {
		return "", nil
	}

	u16Len := int(sizeBytes / 2)
	raw := unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), u16Len)

	end := 0
	for end < len(raw) && raw[end] != 0 {
		end++
	}

	return syscall.UTF16ToString(raw[:end]), nil
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
