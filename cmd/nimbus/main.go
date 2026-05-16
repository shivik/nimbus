// Nimbus - a multi-cloud local emulator
// Entry point: starts an HTTP server that routes incoming cloud API
// requests to the correct provider (AWS, GCP, Azure...) based on the
// request shape (host header, path, auth header).
package main

import (
	"flag"
	"log"

	"github.com/yourname/nimbus/internal/server"
)

func main() {
	port := flag.String("port", "4566", "port to listen on")
	dataDir := flag.String("data", "./data", "directory for persistent state")
	flag.Parse()

	srv := server.New(server.Config{
		Port:    *port,
		DataDir: *dataDir,
	})

	log.Printf("Nimbus listening on :%s (data=%s)", *port, *dataDir)
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}
