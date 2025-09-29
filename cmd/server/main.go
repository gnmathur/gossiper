package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"gmathur.dev/gossiper/internal/server"
	"gmathur.dev/gossiper/internal/transport"
)

func main() {
	id := flag.String("id", "node1", "Node ID")
	httpAddr := flag.String("addr", "localhost:8081", "HTTP address")
	peersStr := flag.String("peers", "", "Comma-separated list of peer addresses")
	flag.Parse()

	peers := []string{}
	if *peersStr != "" {
		peers = strings.Split(*peersStr, ",")
	}

	// 1. Create the core node
	gs := server.NewGameServer(*id, *httpAddr, peers)

	// 2. Create the HTTP server and register handlers
	t := transport.NewServer(gs, nil)
	t.RegisterHandlers()

	// 3. Start the node's background processes
	gs.Start()

	// 4. Start the HTTP server
	log.Printf("[%s] HTTP server listening on %s", *id, *httpAddr)
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}
