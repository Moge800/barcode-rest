package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// randomToken returns a 128-bit hex token used to gate POST /exit when the
// caller did not supply one, so an arbitrary web page cannot guess it.
func randomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("failed to generate exit token: %v", err)
	}
	return hex.EncodeToString(b)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "generate" {
		os.Exit(runGenerate(os.Args[2:]))
	}
	hideConsoleIfOwn()
	port := flag.Int("port", 8787, "listen port")
	version := flag.Bool("version", false, "print version and exit")
	exitToken := flag.String("exit-token", "", "token required as ?token= on POST /exit (random if empty)")
	flag.Parse()

	if *version {
		fmt.Printf("barcode-rest %s\n", server.Version)
		os.Exit(0)
	}

	token := *exitToken
	if token == "" {
		token = randomToken()
	}

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen failed: %v", err)
	}
	fmt.Printf("barcode-rest %s listening on %s\n", server.Version, addr)
	// Stdout only (never the request log), so the token is discoverable for a
	// standalone run but not tied to any barcode request.
	fmt.Printf("exit token: %s (POST http://%s/exit?token=%s to stop)\n", token, addr, token)
	srv := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	// gracefulStop drains in-flight requests, then makes Serve return
	// http.ErrServerClosed. It runs in its own goroutine so the /exit
	// handler can return and let Shutdown observe that connection close.
	gracefulStop := func() {
		log.Printf("shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}

	// POST /exit asks the server to stop (used by callers such as barcodekit
	// when they are done with the resident process).
	srv.Handler = server.New(func() { go gracefulStop() }, token)

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
