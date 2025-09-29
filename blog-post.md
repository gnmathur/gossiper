---
title: "Building a Game State Gossip Protocol in Go: Eventual Consistency for Multiplayer Games"
date: 2025-01-28T10:00:00-08:00
draft: false
tags: ["golang", "distributed-systems", "networking", "gossip-protocol", "game-development"]
categories: ["tutorials"]
---

Ever wondered how multiplayer games keep player scores synchronized across servers without a central database? Let's build a gossip protocol that solves this exact problem. We'll create a distributed game server where player scores eventually converge across all nodes, perfect for scenarios where you need high availability over perfect consistency.

## What We're Building

We're building a distributed game state server where multiple nodes maintain player scores and synchronize them using a gossip protocol. Think of it as a decentralized leaderboard where each server has its own view of player scores, and they periodically sync with each other to reach consensus.

## Setting Up the Project

Let's start with our project structure. We'll follow Go best practices with a clean separation between our application entry point, business logic, and transport layer:

```bash
mkdir gossiper
cd gossiper
go mod init github.com/yourusername/gossiper

# Create our directory structure
mkdir -p cmd/server
mkdir -p internal/server
mkdir -p internal/transport
```

## The Core Data Model

First, let's define what we're synchronizing. We need to track player scores with timestamps for conflict resolution. Create `internal/server/game_server.go`:

```go
package server

import (
    "sync"
    "time"
)

// PlayerState represents a player's score with timestamp for conflict resolution
type PlayerState struct {
    Score     int       `json:"score"`
    Timestamp time.Time `json:"timestamp"`
}

// GameServer manages the distributed game state
type GameServer struct {
    nodeID      string
    peers       []string
    playerState map[string]PlayerState
    mu          sync.RWMutex
}
```

Why timestamps? In distributed systems, we need a way to resolve conflicts when the same player's score is updated on different nodes simultaneously. We're using a "Last-Write-Wins" strategy - the update with the latest timestamp wins. This is simple but effective for gaming scenarios where the most recent action should take precedence.

## Building the Game Server

Now let's implement the core server logic. Add the constructor and state management methods:

```go
func NewGameServer(nodeID string, peers []string) *GameServer {
    return &GameServer{
        nodeID:      nodeID,
        peers:       peers,
        playerState: make(map[string]PlayerState),
    }
}

// UpdatePlayerScore updates a player's score with current timestamp
func (gs *GameServer) UpdatePlayerScore(playerID string, score int) {
    gs.mu.Lock()
    defer gs.mu.Unlock()

    gs.playerState[playerID] = PlayerState{
        Score:     score,
        Timestamp: time.Now(),
    }
}

// GetState returns a copy of the current game state
func (gs *GameServer) GetState() map[string]PlayerState {
    gs.mu.RLock()
    defer gs.mu.RUnlock()

    // Deep copy to prevent data races
    stateCopy := make(map[string]PlayerState)
    for k, v := range gs.playerState {
        stateCopy[k] = v
    }
    return stateCopy
}
```

The key design decision here is using a read-write mutex (`sync.RWMutex`). This allows multiple goroutines to read the state simultaneously while ensuring exclusive access for writes. The deep copy in `GetState` prevents external code from modifying our internal state.

## Implementing the Gossip Protocol

Here's where the magic happens. Let's add the gossip mechanism that periodically syncs with peers:

```go
import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "math/rand"
    "net/http"
)

// Start begins the gossip protocol
func (gs *GameServer) Start() {
    log.Printf("Starting GameServer node: %s", gs.nodeID)
    go gs.gossipLoop()
}

func (gs *GameServer) gossipLoop() {
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        if len(gs.peers) == 0 {
            continue
        }

        // Randomly select a peer
        peer := gs.peers[rand.Intn(len(gs.peers))]
        gs.gossipWithPeer(peer)
    }
}

func (gs *GameServer) gossipWithPeer(peer string) {
    state := gs.GetState()

    data, err := json.Marshal(state)
    if err != nil {
        log.Printf("Failed to marshal state: %v", err)
        return
    }

    url := fmt.Sprintf("http://%s/gossip", peer)
    resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
    if err != nil {
        log.Printf("Failed to gossip with %s: %v", peer, err)
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusOK {
        log.Printf("Successfully gossiped with %s", peer)
    }
}
```

Why random peer selection? This ensures that information spreads evenly through the network over time. If we always picked peers in order, we might create hotspots or leave some nodes isolated. The 2-second interval is a balance between quick propagation and avoiding network congestion.

## Merging States with Conflict Resolution

Now let's implement the crucial merge logic that handles incoming state from peers:

```go
// MergeState merges incoming state with local state using Last-Write-Wins
func (gs *GameServer) MergeState(incomingState map[string]PlayerState) {
    gs.mu.Lock()
    defer gs.mu.Unlock()

    for playerID, incomingPlayer := range incomingState {
        localPlayer, exists := gs.playerState[playerID]

        // If we don't have this player, or incoming is newer, update
        if !exists || incomingPlayer.Timestamp.After(localPlayer.Timestamp) {
            gs.playerState[playerID] = incomingPlayer
            log.Printf("Updated player %s score to %d (timestamp: %v)",
                playerID, incomingPlayer.Score, incomingPlayer.Timestamp)
        }
    }
}
```

This is where our Last-Write-Wins strategy comes into play. For each player in the incoming state, we check if our timestamp is older. If it is, we accept the incoming value. This simple rule ensures that all nodes eventually converge to the same state.

## Adding the HTTP Transport Layer

Let's create the HTTP endpoints for both internal gossip and external API calls. Create `internal/transport/http.go`:

```go
package transport

import (
    "encoding/json"
    "log"
    "net/http"
)

type HTTPTransport struct {
    server *server.GameServer
}

func NewHTTPTransport(s *server.GameServer) *HTTPTransport {
    return &HTTPTransport{server: s}
}

func (h *HTTPTransport) Start(addr string) error {
    mux := http.NewServeMux()

    // Internal gossip endpoint
    mux.HandleFunc("/gossip", h.handleGossip)

    // Public API endpoints
    mux.HandleFunc("/update", h.handleUpdate)
    mux.HandleFunc("/state", h.handleState)

    log.Printf("HTTP server starting on %s", addr)
    return http.ListenAndServe(addr, h.corsMiddleware(mux))
}
```

Let's implement each handler:

```go
func (h *HTTPTransport) handleGossip(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var incomingState map[string]server.PlayerState
    if err := json.NewDecoder(r.Body).Decode(&incomingState); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    h.server.MergeState(incomingState)
    w.WriteHeader(http.StatusOK)
}

func (h *HTTPTransport) handleUpdate(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req struct {
        PlayerID string `json:"player_id"`
        Score    int    `json:"score"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    h.server.UpdatePlayerScore(req.PlayerID, req.Score)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status": "success",
        "message": "Score updated",
    })
}

func (h *HTTPTransport) handleState(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    state := h.server.GetState()

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(state)
}

func (h *HTTPTransport) corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

Why separate internal and public endpoints? The `/gossip` endpoint is for node-to-node communication and accepts full state transfers. The public `/update` and `/state` endpoints are for game clients. This separation allows us to add authentication or rate limiting to public endpoints without affecting internal communication.

## Wiring It All Together

Finally, let's create the main entry point. Create `cmd/server/main.go`:

```go
package main

import (
    "flag"
    "log"
    "strings"

    "github.com/yourusername/gossiper/internal/server"
    "github.com/yourusername/gossiper/internal/transport"
)

func main() {
    var (
        nodeID   = flag.String("node-id", "", "Unique node identifier")
        httpAddr = flag.String("http", "localhost:8080", "HTTP listen address")
        peers    = flag.String("peers", "", "Comma-separated list of peer addresses")
    )
    flag.Parse()

    if *nodeID == "" {
        log.Fatal("node-id is required")
    }

    // Parse peer list
    var peerList []string
    if *peers != "" {
        peerList = strings.Split(*peers, ",")
        log.Printf("Configured peers: %v", peerList)
    }

    // Create and start game server
    gameServer := server.NewGameServer(*nodeID, peerList)
    gameServer.Start()

    // Create and start HTTP transport
    httpTransport := transport.NewHTTPTransport(gameServer)
    if err := httpTransport.Start(*httpAddr); err != nil {
        log.Fatalf("Failed to start HTTP server: %v", err)
    }
}
```

## Building and Running

Now let's see our gossip protocol in action! First, build the project:

```bash
go build -o gossiper cmd/server/main.go
```

Start a three-node cluster in different terminals:

```bash
# Terminal 1: Node A
./gossiper -node-id=node-a -http=localhost:8080 \
  -peers="localhost:8081,localhost:8082"

# Terminal 2: Node B
./gossiper -node-id=node-b -http=localhost:8081 \
  -peers="localhost:8080,localhost:8082"

# Terminal 3: Node C
./gossiper -node-id=node-c -http=localhost:8082 \
  -peers="localhost:8080,localhost:8081"
```

## Testing the Gossip Protocol

Let's update a player's score on one node and watch it propagate:

```bash
# Update player1's score on node A
curl -X POST http://localhost:8080/update \
  -H "Content-Type: application/json" \
  -d '{"player_id": "player1", "score": 100}'

# Wait a few seconds for gossip to spread
sleep 3

# Check the state on all nodes
echo "Node A state:"
curl -s http://localhost:8080/state | jq

echo "Node B state:"
curl -s http://localhost:8081/state | jq

echo "Node C state:"
curl -s http://localhost:8082/state | jq
```

You should see player1's score on all three nodes! Try updating the same player on different nodes simultaneously:

```bash
# Simultaneous updates to test conflict resolution
curl -X POST http://localhost:8080/update \
  -d '{"player_id": "player2", "score": 200}' &

curl -X POST http://localhost:8081/update \
  -d '{"player_id": "player2", "score": 300}' &

wait
sleep 3

# Check which update won (should be the one with later timestamp)
curl -s http://localhost:8080/state | jq '.player2'
```

## Understanding the Design Tradeoffs

**Pull vs Push Gossip**: We implemented a push model where nodes actively send their state to peers. A pull model (where nodes request state from peers) would reduce unnecessary transfers but add request-response complexity.

**Full State Transfer**: We send the entire state in each gossip round. This is simple but inefficient for large datasets. A production system might use incremental updates or Merkle trees to identify differences.

**Random Peer Selection**: This provides good coverage but isn't optimal. Smart peer selection based on network topology or recent communication patterns could improve efficiency.

**Last-Write-Wins**: Simple but can lose updates if clocks aren't synchronized. Vector clocks or CRDTs would provide better conflict resolution but add complexity.

## Performance Characteristics

Let's analyze how information spreads. With our settings:
- Gossip interval: 2 seconds
- Random peer selection
- Full mesh topology (every node knows every other node)

Expected propagation time for N nodes: O(log N) gossip rounds. For 3 nodes, most updates converge within 2-4 seconds. For 10 nodes, expect 6-8 seconds.

## What's Next?

Now that we have a working gossip-based game server, here are some enhancements to try:

1. **Add Player Sessions**: Track active players and remove inactive ones after a timeout.

2. **Implement Delta Gossip**: Instead of full state, send only recent changes to reduce bandwidth.

3. **Add Failure Detection**: Use heartbeats to detect failed nodes and remove them from peer lists.

4. **Persistent Storage**: Add a database layer so state survives restarts.

5. **Metrics and Monitoring**: Track gossip latency, message rates, and convergence time.

6. **Security**: Add node authentication and encrypt gossip messages.

7. **Dynamic Membership**: Allow nodes to join and leave the cluster dynamically.

The beauty of this approach is its simplicity and resilience. There's no single point of failure, no complex leader election, and the system naturally handles network partitions. Perfect for game servers where eventual consistency is acceptable and you need high availability.

Try killing a node and restarting it - the system continues working. Add network delays to simulate real-world conditions. Scale to 10 nodes and watch how gossip patterns emerge. That's the elegance of gossip protocols - simple rules creating robust distributed behavior!