package main

import (
	"syscall"
	"unsafe"
)

// hideConsoleIfOwn hides the console window when this process is its sole
// owner (double-click / shell:startup launch). Launched from an existing
// shell, the console is shared and stays visible. Built as a plain console
// app instead of -H windowsgui to avoid antivirus false positives.
func hideConsoleIfOwn() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	var pids [2]uint32
	n, _, _ := kernel32.NewProc("GetConsoleProcessList").Call(uintptr(unsafe.Pointer(&pids[0])), uintptr(len(pids)))
	if n != 1 {
		return
	}
	hwnd, _, _ := kernel32.NewProc("GetConsoleWindow").Call()
	if hwnd != 0 {
		const swHide = 0
		syscall.NewLazyDLL("user32.dll").NewProc("ShowWindow").Call(hwnd, swHide)
	}
}
