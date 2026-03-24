package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	questionsvc "GlitchRoyale/internal/infrastructure/questions"

	"github.com/gorilla/websocket"
)

const questionFetchTimeout = 5 * time.Second

type QuestionService interface {
	GetRandomQuestion(ctx context.Context) (questionsvc.Question, error)
}

type WSHandler struct {
	hub             *Hub
	questionService QuestionService
}

type Message struct {
	Type     string `json:"type"`
	Code     string `json:"code,omitempty"`
	Name     string `json:"name,omitempty"`
	AnswerID string `json:"answerId,omitempty"`
	TargetID string `json:"targetId,omitempty"`
}

type RoomCreatedResponse struct {
	Type string `json:"type"`
	Code string `json:"code"`
}

type AnswerResultResponse struct {
	Type     string `json:"type"`
	RoomCode string `json:"roomCode"`
	PlayerID string `json:"playerId"`
	Correct  bool   `json:"correct"`
	Tokens   int    `json:"tokens"`
	HP       int    `json:"hp"`
}

type ErrorResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var playerSequence atomic.Uint64

func NewWSHandler(hub *Hub, questionService QuestionService) *WSHandler {
	return &WSHandler{hub: hub, questionService: questionService}
}

func (h *WSHandler) HandleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("error upgrading websocket:", err)
		return
	}

	playerID := nextPlayerID()

	playerName := normalizePlayerName(r.URL.Query().Get("name"))

	client := &Client{
		ID:     playerID,
		Name:   playerName,
		HP:     initialPlayerHP,
		Tokens: initialPlayerTokens,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	h.hub.Register <- client
	log.Println("client connected:", client.ID)

	go h.readPump(client)
	go h.writePump(client)
}

func (h *WSHandler) readPump(client *Client) {
	defer func() {
		h.hub.Unregister <- client
		client.Conn.Close()
		log.Println("client disconnected:", client.ID)
	}()

	for {
		_, payload, err := client.Conn.ReadMessage()
		if err != nil {
			log.Println("error reading websocket:", err)
			return
		}

		var message Message
		if err := json.Unmarshal(payload, &message); err != nil {
			log.Println("error parsing websocket message:", err)
			h.sendError(client, "invalid_message")
			continue
		}

		switch message.Type {
		case "create_room":
			h.handleCreateRoom(client, message.Name)

		case "join_room":
			h.handleJoinRoom(client, message.Code, message.Name)

		case "start_game":
			h.handleStartGame(client)

		case "answer":
			h.handleAnswer(client, message.AnswerID)

		case "attack":
			h.handleAttack(client, message.TargetID)

		default:
			log.Println("unknown message type:", message.Type)
			h.sendError(client, "unknown_message_type")
		}
	}
}

func (h *WSHandler) writePump(client *Client) {
	defer client.Conn.Close()

	for message := range client.Send {
		if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Println("error writing websocket:", err)
			return
		}
	}
}

func (h *WSHandler) handleCreateRoom(client *Client, name string) {
	roomCode, err := h.hub.CreateRoomForClient(client, name)
	if err != nil {
		log.Println("error creating room:", err)
		h.sendError(client, "room_creation_failed")
		return
	}

	h.sendJSON(client, RoomCreatedResponse{Type: "room_created", Code: roomCode})
}

func (h *WSHandler) handleJoinRoom(client *Client, code string, name string) {
	roomCode := strings.ToUpper(strings.TrimSpace(code))
	if roomCode == "" {
		h.sendError(client, "room_code_required")
		return
	}

	if err := h.hub.JoinRoomForClient(client, roomCode, name); err != nil {
		if err == ErrRoomNotFound {
			h.sendError(client, "room_not_found")
			return
		}

		log.Println("error joining room:", err)
		h.sendError(client, "room_join_failed")
	}
}

func (h *WSHandler) handleStartGame(client *Client) {
	if strings.TrimSpace(client.RoomCode) == "" {
		h.sendError(client, "client_not_in_room")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), questionFetchTimeout)
	defer cancel()

	question, err := h.questionService.GetRandomQuestion(ctx)
	if err != nil {
		log.Println("error fetching question:", err)
		h.sendError(client, "question_fetch_failed")
		return
	}

	roomCode := client.RoomCode
	if err := h.hub.BeginCountdownForClient(client); err != nil {
		h.sendError(client, h.mapHubError(err))
		return
	}
	defer h.hub.EndCountdownForRoom(roomCode)

	h.hub.BroadcastStartGame(roomCode)

	for countdown := 3; countdown >= 1; countdown-- {
		h.hub.BroadcastCountdown(roomCode, countdown)
		time.Sleep(time.Second)
	}

	if err := h.hub.StartGameForClient(client, question); err != nil {
		log.Println("error starting game:", err)
		h.sendError(client, h.mapHubError(err))
	}
}

func (h *WSHandler) handleAnswer(client *Client, answerID string) {
	normalizedAnswerID := strings.TrimSpace(answerID)
	if normalizedAnswerID == "" {
		h.sendError(client, "answer_id_required")
		return
	}

	result := h.hub.SubmitAnswerForClient(client, normalizedAnswerID)
	if result.Err != nil {
		h.sendError(client, h.mapHubError(result.Err))
		return
	}

	if !result.Accepted {
		return
	}
}

func (h *WSHandler) handleAttack(client *Client, targetID string) {
	normalizedTargetID := strings.TrimSpace(targetID)
	if normalizedTargetID == "" {
		h.sendError(client, "target_id_required")
		return
	}

	if err := h.hub.AttackPlayerForClient(client, normalizedTargetID); err != nil {
		h.sendError(client, h.mapHubError(err))
	}
}

func (h *WSHandler) mapHubError(err error) string {
	switch err {
	case ErrRoomNotFound:
		return "room_not_found"
	case ErrClientNotInRoom:
		return "client_not_in_room"
	case ErrNoActiveQuestion:
		return "no_active_question"
	case ErrNotEnoughTokens:
		return "not_enough_tokens"
	case ErrTargetNotFound:
		return "target_not_found"
	case ErrCannotAttackYourself:
		return "cannot_attack_yourself"
	case ErrPlayerAlreadyDefeated:
		return "player_already_defeated"
	case ErrGameAlreadyFinished:
		return "game_over"
	case ErrCountdownAlreadyRunning:
		return "countdown_in_progress"
	case ErrTurnContinuationNotAllowed:
		return "turn_continuation_not_allowed"
	default:
		return "internal_error"
	}
}

func (h *WSHandler) sendError(client *Client, message string) {
	h.sendJSON(client, ErrorResponse{Type: "error", Message: message})
}

func (h *WSHandler) sendJSON(client *Client, payload interface{}) {
	message, err := json.Marshal(payload)
	if err != nil {
		log.Println("error marshalling websocket payload:", err)
		return
	}

	client.Send <- message
}

func nextPlayerID() string {
	value := playerSequence.Add(1)
	return fmt.Sprintf("player-%d", value)
}
