package main

import (
	"log"
	"net/http"

	"GlitchRoyale/internal/application/usecases"
	"GlitchRoyale/internal/infrastructure/repository"
	"GlitchRoyale/internal/infrastructure/ws"
	httpHandler "GlitchRoyale/internal/interfaces/http"
)

func main() {

	// 🔹 Repository (infraestructura)
	repo := repository.NewMemoryGameRepository()

	// 🔹 Casos de uso (application)
	createGame := usecases.NewCreateGame(repo)
	joinGame := usecases.NewJoinGame(repo)

	// 🔹 HTTP handlers (interfaces)
	handler := httpHandler.NewHandler(createGame, joinGame)

	// 🔹 WebSocket Hub (infraestructura)
	hub := ws.NewHub()
	go hub.Run()

	wsHandler := ws.NewWSHandler(hub)

	// 🔹 Rutas HTTP
	http.HandleFunc("/create", handler.CreateGame)
	http.HandleFunc("/join", handler.JoinGame)

	// 🔹 Ruta WebSocket
	http.HandleFunc("/ws", wsHandler.HandleConnections)

	log.Println("🚀 Server running on :8080")
	http.ListenAndServe(":8080", nil)
}
