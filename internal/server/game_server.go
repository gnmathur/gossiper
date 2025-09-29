package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// PlayerState represents the state of a player in the game. This is the data that we will sync across game servers
// via gossip
type PlayerState struct {
	Score     int64 `json: "score"`
	Timestamp int64 `json: "timestamp"`
}

type GameServer struct {
	ID        string                 // unique ID of the game server
	Address   string                 // address of the game server. host:port format
	Peers     []string               // list of peer game server IDs
	PlayerMap map[string]PlayerState // map of player ID to player state
	mu        sync.RWMutex
}

func NewGameServer(id, addr string, peers []string) *GameServer {
	return &GameServer{
		ID:        id,
		Address:   addr,
		Peers:     peers,
		PlayerMap: make(map[string]PlayerState),
	}
}

func (gs *GameServer) Start() {
	go gs.gossipLoop()
}

func (gs *GameServer) gossipLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if len(gs.Peers) == 0 {
			continue
		}
		peerAddr := gs.Peers[rand.Intn(len(gs.Peers))]
		gs.gossipWithPeer(peerAddr)
	}
}

func (gs *GameServer) gossipWithPeer(peerAddr string) {
	gs.mu.RLock()
	payload, err := json.Marshal(gs.PlayerMap)
	gs.mu.RUnlock()

	if err != nil {
		log.Println("failed to gossip with peer:", err)
	}

	url := fmt.Sprintf("http://%s/gossip", peerAddr)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		// If we add logs here, we'll spam the logs a lot in case a peer is down
		// Todo: We'll add failure metrics here later
		return
	}

	defer resp.Body.Close()
}

func (gs *GameServer) MergeState(incomingMap map[string]PlayerState) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	for playerId, incomingState := range incomingMap {
		localState, exists := gs.PlayerMap[playerId]
		if !exists || incomingState.Timestamp > localState.Timestamp {
			gs.PlayerMap[playerId] = PlayerState{
				Score:     incomingState.Score,
				Timestamp: incomingState.Timestamp,
			}
		}
	}
}

func (gs *GameServer) UpdatePlayerScore(playerId string, score int64) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gs.PlayerMap[playerId] = PlayerState{
		Score:     score,
		Timestamp: time.Now().UnixNano(),
	}
	log.Printf("updated score for player %s. New score %d", playerId, score)
}

func (gs *GameServer) GetPlayerState() map[string]PlayerState {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	result := make(map[string]PlayerState, len(gs.PlayerMap))
	for playerId, state := range gs.PlayerMap {
		result[playerId] = state
	}

	return result
}
