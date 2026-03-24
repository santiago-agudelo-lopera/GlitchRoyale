package main

import (
	"log"
	"net/http"
	"os"

	"GlitchRoyale/internal/application/usecases"
	"GlitchRoyale/internal/infrastructure/database"
	questionsvc "GlitchRoyale/internal/infrastructure/questions"
	"GlitchRoyale/internal/infrastructure/repository"
	"GlitchRoyale/internal/infrastructure/ws"
	httpHandler "GlitchRoyale/internal/interfaces/http"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := database.NewPostgresDB()
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer db.Close()

	log.Println("postgres connection established")

	questionService := questionsvc.NewService(db)

	repo := repository.NewMemoryGameRepository()
	createGame := usecases.NewCreateGame(repo)
	joinGame := usecases.NewJoinGame(repo)
	handler := httpHandler.NewHandler(createGame, joinGame)

	hub := ws.NewHub()
	go hub.Run()

	wsHandler := ws.NewWSHandler(hub, questionService)

	mux := http.NewServeMux()
	mux.HandleFunc("/create", handler.CreateGame)
	mux.HandleFunc("/join", handler.JoinGame)
	mux.HandleFunc("/ws", wsHandler.HandleConnections)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	address := ":" + port

	log.Println("Servidor corriendo en puerto:", port)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
