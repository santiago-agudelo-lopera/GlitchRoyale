package main

import (
	"log"
	"net/http"

	"GlitchRoyale/internal/application/usecases"
	"GlitchRoyale/internal/infrastructure/database"
	questionsvc "GlitchRoyale/internal/infrastructure/questions"
	"GlitchRoyale/internal/infrastructure/repository"
	"GlitchRoyale/internal/infrastructure/ws"
	httpHandler "GlitchRoyale/internal/interfaces/http"
)

func main() {
	db, err := database.NewPostgresDB()
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	questionService := questionsvc.NewService(db)

	repo := repository.NewMemoryGameRepository()
	createGame := usecases.NewCreateGame(repo)
	joinGame := usecases.NewJoinGame(repo)
	handler := httpHandler.NewHandler(createGame, joinGame)

	hub := ws.NewHub()
	go hub.Run()

	wsHandler := ws.NewWSHandler(hub, questionService)

	http.HandleFunc("/create", handler.CreateGame)
	http.HandleFunc("/join", handler.JoinGame)
	http.HandleFunc("/ws", wsHandler.HandleConnections)

	log.Println("Server running on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
