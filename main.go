package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"barcode-rest/internal/server"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "generate" {
		os.Exit(runGenerate(os.Args[2:]))
	}
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
	srv := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	// gracefulStop drains in-flight requests, then makes Serve return
	// http.ErrServerClosed. It runs in its own goroutine so the /shutdown
	// handler can return and let Shutdown observe that connection close.
	gracefulStop := func() {
		log.Printf("shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}

	// POST /shutdown asks the server to stop (used by callers such as barcodekit
	// when they are done with the resident process).
	srv.Handler = server.New(func() { go gracefulStop() })

	// Ctrl-C / SIGTERM also stop cleanly.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		gracefulStop()
	}()

	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
