package main

import (
	"syscall"
	"unsafe"

	"github.com/JamesHovious/w32"
	"pipelined.dev/audio/vst2"
)

func newNativeWindow(name string, width, height int) vst2.Ptr {
	hInstance := w32.GetModuleHandle("")
	lpszClassName := syscall.StringToUTF16Ptr("WNDclass")

	var wcex w32.WNDCLASSEX
	wcex.Size = uint32(unsafe.Sizeof(wcex))
	wcex.Style = w32.CS_HREDRAW | w32.CS_VREDRAW
	wcex.WndProc = syscall.NewCallback(wndProc)
	wcex.ClsExtra = 0
	wcex.WndExtra = 0
	wcex.Instance = hInstance
	wcex.Icon = w32.LoadIcon(hInstance, makeIntResource(w32.IDI_APPLICATION))
	wcex.Cursor = w32.LoadCursor(0, makeIntResource(w32.IDC_ARROW))
	wcex.Background = w32.COLOR_WINDOW + 11
	wcex.MenuName = nil
	wcex.ClassName = lpszClassName
	wcex.IconSm = w32.LoadIcon(hInstance, makeIntResource(w32.IDI_APPLICATION))
	w32.RegisterClassEx(&wcex)

	return vst2.Ptr(w32.CreateWindowEx(
		0, lpszClassName, syscall.StringToUTF16Ptr(name),
		w32.WS_OVERLAPPEDWINDOW|w32.WS_VISIBLE,
		w32.CW_USEDEFAULT, w32.CW_USEDEFAULT, width, height, 0, 0, hInstance, nil))
}

func wndProc(hWnd w32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case w32.WM_DESTROY:
		w32.PostQuitMessage(0)
	default:
		return w32.DefWindowProc(hWnd, msg, wParam, lParam)
	}
	return 0
}

func makeIntResource(id uint16) *uint16 {
	return (*uint16)(unsafe.Pointer(uintptr(id)))
}
