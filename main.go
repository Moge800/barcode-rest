package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"syscall"
	"unsafe"

	"barcode-rest/internal/server"
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

func main() {
	hideConsoleIfOwn()
	port := flag.Int("port", 8787, "listen port")
	version := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *version {
		fmt.Printf("barcode-rest %s\n", server.Version)
		os.Exit(0)
	}

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}
	fmt.Printf("barcode-rest %s listening on %s\n", server.Version, addr)
	log.Fatal(http.Serve(ln, server.New()))
}
