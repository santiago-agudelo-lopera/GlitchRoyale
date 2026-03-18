package http

import (
	"encoding/json"
	"net/http"

	"GlitchRoyale/internal/application/usecases"
)

type Handler struct {
	createGame *usecases.CreateGame
	joinGame   *usecases.JoinGame
}

func NewHandler(cg *usecases.CreateGame, jg *usecases.JoinGame) *Handler {
	return &Handler{cg, jg}
}

func (h *Handler) CreateGame(w http.ResponseWriter, r *http.Request) {
	game, _ := h.createGame.Execute("game-1")

	json.NewEncoder(w).Encode(game)
}

func (h *Handler) JoinGame(w http.ResponseWriter, r *http.Request) {
	game, _ := h.joinGame.Execute("game-1", "player-1")

	json.NewEncoder(w).Encode(game)
}
