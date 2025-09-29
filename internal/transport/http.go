package transport

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gmathur.dev/gossiper/internal/server"
)

// Server dependencies for use by the HTTP server
type Server struct {
	gs *server.GameServer
}

func NewServer(gs *server.GameServer, _ any) *Server {
	return &Server{gs: gs}
}

func (s *Server) RegisterHandlers() {
	// API handlers
	http.HandleFunc("/gossip", s.HandleGossip)
	http.HandleFunc("/update", s.HandleUpdate)
	http.HandleFunc("/state", s.HandleGetState)
}

func (s *Server) HandleGossip(w http.ResponseWriter, r *http.Request) {
	var incomingMap map[string]server.PlayerState
	err := json.NewDecoder(r.Body).Decode(&incomingMap)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Merge incoming state with local state
	s.gs.MergeState(incomingMap)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	playerId := r.URL.Query().Get("playerId")
	scoreStr := r.URL.Query().Get("score")
	if playerId == "" || scoreStr == "" {
		http.Error(w, "missing playerId or score", http.StatusBadRequest)
		return
	}

	var score int
	if _, err := fmt.Sscanf(scoreStr, "%d", &score); err != nil {
		http.Error(w, "invalid playerId or score", http.StatusBadRequest)
		return
	}

	// Update player score
	s.gs.UpdatePlayerScore(playerId, int64(score))

	w.WriteHeader(http.StatusOK)
}

func (s *Server) HandleGetState(w http.ResponseWriter, r *http.Request) {
	state := s.gs.GetPlayerState()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if err := json.NewEncoder(w).Encode(state); err != nil {
		http.Error(w, "failed to encode state", http.StatusInternalServerError)
		return
	}
}
