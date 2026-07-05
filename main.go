package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
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
		Handler:           server.New(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	log.Fatal(srv.Serve(ln))
}
