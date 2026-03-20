package ws

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"

	questionsvc "GlitchRoyale/internal/infrastructure/questions"
)

const (
	defaultPlayerName   = "Player"
	maxPlayerNameLength = 12
	initialPlayerHP     = 100
	initialPlayerTokens = 0
	answerRewardTokens  = 10
	attackCostTokens    = 5
	attackDamageHP      = 20
)

var (
	ErrRoomNotFound          = errors.New("room not found")
	ErrClientNotInRoom       = errors.New("client not in room")
	ErrNoActiveQuestion      = errors.New("no active question")
	ErrAnswerAlreadySent     = errors.New("answer already sent")
	ErrInvalidAnswerOption   = errors.New("invalid answer option")
	ErrNotEnoughTokens       = errors.New("not enough tokens")
	ErrTargetNotFound        = errors.New("target not found")
	ErrCannotAttackYourself  = errors.New("cannot attack yourself")
	ErrPlayerAlreadyDefeated = errors.New("player already defeated")
	ErrGameAlreadyFinished   = errors.New("game already finished")
)

type Player struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	HP       int    `json:"hp"`
	Tokens   int    `json:"tokens"`
	RoomCode string `json:"roomCode"`
}

type RoomJoinedMessage struct {
	Type    string   `json:"type"`
	Code    string   `json:"code"`
	Players []Player `json:"players"`
	YourID  string   `json:"yourId"`
}

type GameOverMessage struct {
	Type     string `json:"type"`
	RoomCode string `json:"roomCode"`
	WinnerID string `json:"winnerId"`
	LoserID  string `json:"loserId"`
}

type QuestionOption struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type RoomQuestion struct {
	ID              string
	Question        string
	Category        string
	Difficulty      string
	Options         []QuestionOption
	AnswerCorrectBy map[string]bool
	AnsweredPlayers map[string]bool
}

type RoomGameState struct {
	CurrentQuestion *RoomQuestion
	GameOver        bool
	WinnerID        string
	LoserID         string
}

type CreateRoomRequest struct {
	Client   *Client
	Name     string
	Response chan CreateRoomResult
}

type CreateRoomResult struct {
	Code string
	Err  error
}

type JoinRoomRequest struct {
	Client   *Client
	Code     string
	Name     string
	Response chan error
}

type StartGameRequest struct {
	Client   *Client
	Question questionsvc.Question
	Response chan error
}

type SubmitAnswerRequest struct {
	Client   *Client
	AnswerID string
	Response chan AnswerResult
}

type AnswerResult struct {
	Correct bool
	Tokens  int
	Err     error
}

type AttackPlayerRequest struct {
	Client   *Client
	TargetID string
	Response chan error
}

type RoomBroadcast struct {
	Code    string
	Payload []byte
}

type Hub struct {
	Clients       map[*Client]bool
	Rooms         map[string][]*Client
	RoomStates    map[string]*RoomGameState
	Register      chan *Client
	Unregister    chan *Client
	CreateRoom    chan CreateRoomRequest
	JoinRoom      chan JoinRoomRequest
	StartGame     chan StartGameRequest
	SubmitAnswer  chan SubmitAnswerRequest
	AttackPlayer  chan AttackPlayerRequest
	BroadcastRoom chan RoomBroadcast
}

func NewHub() *Hub {
	return &Hub{
		Clients:       make(map[*Client]bool),
		Rooms:         make(map[string][]*Client),
		RoomStates:    make(map[string]*RoomGameState),
		Register:      make(chan *Client),
		Unregister:    make(chan *Client),
		CreateRoom:    make(chan CreateRoomRequest),
		JoinRoom:      make(chan JoinRoomRequest),
		StartGame:     make(chan StartGameRequest),
		SubmitAnswer:  make(chan SubmitAnswerRequest),
		AttackPlayer:  make(chan AttackPlayerRequest),
		BroadcastRoom: make(chan RoomBroadcast),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			log.Println("client registered:", client.ID, "total:", len(h.Clients))

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case request := <-h.CreateRoom:
			code, err := generateRoomCode(h.Rooms)
			if err != nil {
				request.Response <- CreateRoomResult{Err: err}
				continue
			}

			h.resetPlayerState(request.Client)
			request.Client.Name = h.buildUniquePlayerName(code, request.Name, request.Client)
			h.moveClientToRoom(request.Client, code)
			request.Response <- CreateRoomResult{Code: code}
			h.broadcastRoomState(code)

		case request := <-h.JoinRoom:
			if _, ok := h.Rooms[request.Code]; !ok {
				request.Response <- ErrRoomNotFound
				continue
			}

			h.resetPlayerState(request.Client)
			request.Client.Name = h.buildUniquePlayerName(request.Code, request.Name, request.Client)
			h.moveClientToRoom(request.Client, request.Code)
			request.Response <- nil
			h.broadcastRoomState(request.Code)

		case request := <-h.StartGame:
			request.Response <- h.startGame(request.Client, request.Question)

		case request := <-h.SubmitAnswer:
			request.Response <- h.submitAnswer(request.Client, request.AnswerID)

		case request := <-h.AttackPlayer:
			request.Response <- h.attackPlayer(request.Client, request.TargetID)

		case message := <-h.BroadcastRoom:
			h.broadcastRawToRoom(message.Code, message.Payload)
		}
	}
}

func (h *Hub) CreateRoomForClient(client *Client, name string) (string, error) {
	response := make(chan CreateRoomResult)

	h.CreateRoom <- CreateRoomRequest{
		Client:   client,
		Name:     name,
		Response: response,
	}

	result := <-response
	return result.Code, result.Err
}

func (h *Hub) JoinRoomForClient(client *Client, code string, name string) error {
	response := make(chan error)

	h.JoinRoom <- JoinRoomRequest{
		Client:   client,
		Code:     code,
		Name:     name,
		Response: response,
	}

	return <-response
}

func (h *Hub) StartGameForClient(client *Client, question questionsvc.Question) error {
	response := make(chan error)

	h.StartGame <- StartGameRequest{
		Client:   client,
		Question: question,
		Response: response,
	}

	return <-response
}

func (h *Hub) SubmitAnswerForClient(client *Client, answerID string) AnswerResult {
	response := make(chan AnswerResult)

	h.SubmitAnswer <- SubmitAnswerRequest{
		Client:   client,
		AnswerID: answerID,
		Response: response,
	}

	return <-response
}

func (h *Hub) AttackPlayerForClient(client *Client, targetID string) error {
	response := make(chan error)

	h.AttackPlayer <- AttackPlayerRequest{
		Client:   client,
		TargetID: targetID,
		Response: response,
	}

	return <-response
}

func (h *Hub) BroadcastToRoom(code string, payload []byte) {
	if code == "" {
		return
	}

	h.BroadcastRoom <- RoomBroadcast{Code: code, Payload: payload}
}

func (h *Hub) unregisterClient(client *Client) {
	if _, ok := h.Clients[client]; !ok {
		return
	}

	delete(h.Clients, client)
	close(client.Send)

	roomCode, roomHasClients := h.removeClientFromRoom(client)
	log.Println("client removed:", client.ID, "total:", len(h.Clients))

	if roomHasClients {
		h.broadcastRoomState(roomCode)
		h.broadcastGameState(roomCode)
	}
}

func (h *Hub) startGame(client *Client, question questionsvc.Question) error {
	roomCode := client.RoomCode
	if roomCode == "" {
		return ErrClientNotInRoom
	}

	roomState := h.ensureRoomState(roomCode)
	if roomState.GameOver {
		h.resetFinishedMatch(roomCode)
		roomState = h.ensureRoomState(roomCode)
	}

	options, answerCorrectBy := buildQuestionOptions(question.Answers)
	roomState.GameOver = false
	roomState.WinnerID = ""
	roomState.LoserID = ""
	roomState.CurrentQuestion = &RoomQuestion{
		ID:              question.ID,
		Question:        question.Question,
		Category:        question.Category,
		Difficulty:      question.Difficulty,
		Options:         options,
		AnswerCorrectBy: answerCorrectBy,
		AnsweredPlayers: make(map[string]bool),
	}

	h.broadcastQuestion(roomCode, roomState.CurrentQuestion)
	h.broadcastGameState(roomCode)
	return nil
}

func (h *Hub) submitAnswer(client *Client, answerID string) AnswerResult {
	roomCode := client.RoomCode
	if roomCode == "" {
		return AnswerResult{Err: ErrClientNotInRoom}
	}

	roomState := h.ensureRoomState(roomCode)
	if roomState.GameOver {
		return AnswerResult{Err: ErrGameAlreadyFinished}
	}

	if roomState.CurrentQuestion == nil {
		return AnswerResult{Err: ErrNoActiveQuestion}
	}

	if roomState.CurrentQuestion.AnsweredPlayers[client.ID] {
		return AnswerResult{Err: ErrAnswerAlreadySent}
	}

	correct, ok := roomState.CurrentQuestion.AnswerCorrectBy[answerID]
	if !ok {
		return AnswerResult{Err: ErrInvalidAnswerOption}
	}

	roomState.CurrentQuestion.AnsweredPlayers[client.ID] = true
	if correct {
		client.Tokens += answerRewardTokens
	}

	h.broadcastGameState(roomCode)
	return AnswerResult{Correct: correct, Tokens: client.Tokens}
}

func (h *Hub) attackPlayer(client *Client, targetID string) error {
	roomCode := client.RoomCode
	if roomCode == "" {
		return ErrClientNotInRoom
	}

	roomState := h.ensureRoomState(roomCode)
	if roomState.GameOver {
		return ErrGameAlreadyFinished
	}

	if client.ID == targetID {
		return ErrCannotAttackYourself
	}

	if client.HP <= 0 {
		return ErrPlayerAlreadyDefeated
	}

	if client.Tokens < attackCostTokens {
		return ErrNotEnoughTokens
	}

	var target *Client
	for _, roomClient := range h.Rooms[roomCode] {
		if roomClient.ID == targetID {
			target = roomClient
			break
		}
	}

	if target == nil {
		return ErrTargetNotFound
	}

	if target.HP <= 0 {
		return ErrPlayerAlreadyDefeated
	}

	client.Tokens -= attackCostTokens
	target.HP -= attackDamageHP
	if target.HP < 0 {
		target.HP = 0
	}

	payload, err := json.Marshal(struct {
		Type         string `json:"type"`
		Attacker     string `json:"attacker"`
		AttackerName string `json:"attackerName"`
		Target       string `json:"target"`
		TargetName   string `json:"targetName"`
		Damage       int    `json:"damage"`
		RemainingHP  int    `json:"remainingHP"`
	}{
		Type:         "player_attacked",
		Attacker:     client.ID,
		AttackerName: client.Name,
		Target:       target.ID,
		TargetName:   target.Name,
		Damage:       attackDamageHP,
		RemainingHP:  target.HP,
	})
	if err != nil {
		return err
	}

	h.broadcastRawToRoom(roomCode, payload)
	h.broadcastGameState(roomCode)

	if target.HP <= 0 {
		h.finishGame(roomCode, target.ID)
	}

	return nil
}

func (h *Hub) moveClientToRoom(client *Client, roomCode string) {
	previousRoomCode, previousRoomHasClients := h.removeClientFromRoom(client)
	if previousRoomHasClients && previousRoomCode != roomCode {
		h.broadcastRoomState(previousRoomCode)
		h.broadcastGameState(previousRoomCode)
	}

	client.RoomCode = roomCode
	h.Rooms[roomCode] = append(h.Rooms[roomCode], client)
	h.ensureRoomState(roomCode)
}

func (h *Hub) removeClientFromRoom(client *Client) (string, bool) {
	if client.RoomCode == "" {
		return "", false
	}

	roomCode := client.RoomCode
	roomClients := h.Rooms[roomCode]
	updatedClients := make([]*Client, 0, len(roomClients))

	for _, roomClient := range roomClients {
		if roomClient != client {
			updatedClients = append(updatedClients, roomClient)
		}
	}

	client.RoomCode = ""

	if len(updatedClients) == 0 {
		delete(h.Rooms, roomCode)
		delete(h.RoomStates, roomCode)
		log.Println("room deleted:", roomCode)
		return roomCode, false
	}

	h.Rooms[roomCode] = updatedClients
	if roomState, ok := h.RoomStates[roomCode]; ok && roomState.CurrentQuestion != nil {
		delete(roomState.CurrentQuestion.AnsweredPlayers, client.ID)
	}
	return roomCode, true
}

func (h *Hub) ensureRoomState(roomCode string) *RoomGameState {
	if roomState, ok := h.RoomStates[roomCode]; ok {
		return roomState
	}

	roomState := &RoomGameState{}
	h.RoomStates[roomCode] = roomState
	return roomState
}

func (h *Hub) broadcastRoomState(roomCode string) {
	players := h.roomPlayers(roomCode)
	if len(players) == 0 {
		return
	}

	for _, client := range h.Rooms[roomCode] {
		payload, err := json.Marshal(RoomJoinedMessage{
			Type:    "room_joined",
			Code:    roomCode,
			Players: players,
			YourID:  client.ID,
		})
		if err != nil {
			log.Println("error marshalling room state:", err)
			return
		}

		client.Send <- payload
	}
}

func (h *Hub) broadcastGameState(roomCode string) {
	players := h.roomPlayers(roomCode)
	if len(players) == 0 {
		return
	}

	payload, err := json.Marshal(struct {
		Type     string   `json:"type"`
		RoomCode string   `json:"roomCode"`
		Players  []Player `json:"players"`
	}{
		Type:     "game_state",
		RoomCode: roomCode,
		Players:  players,
	})
	if err != nil {
		log.Println("error marshalling game state:", err)
		return
	}

	h.broadcastRawToRoom(roomCode, payload)
}

func (h *Hub) broadcastQuestion(roomCode string, question *RoomQuestion) {
	if question == nil {
		return
	}

	payload, err := json.Marshal(struct {
		Type       string           `json:"type"`
		RoomCode   string           `json:"roomCode"`
		QuestionID string           `json:"questionId"`
		Question   string           `json:"question"`
		Category   string           `json:"category"`
		Difficulty string           `json:"difficulty"`
		Answers    []QuestionOption `json:"answers"`
	}{
		Type:       "question",
		RoomCode:   roomCode,
		QuestionID: question.ID,
		Question:   question.Question,
		Category:   question.Category,
		Difficulty: question.Difficulty,
		Answers:    question.Options,
	})
	if err != nil {
		log.Println("error marshalling question:", err)
		return
	}

	h.broadcastRawToRoom(roomCode, payload)
}

func (h *Hub) broadcastGameOver(roomCode string, winnerID string, loserID string) {
	if roomCode == "" || winnerID == "" || loserID == "" {
		return
	}

	payload, err := json.Marshal(GameOverMessage{
		Type:     "game_over",
		RoomCode: roomCode,
		WinnerID: winnerID,
		LoserID:  loserID,
	})
	if err != nil {
		log.Println("error marshalling game over:", err)
		return
	}

	h.broadcastRawToRoom(roomCode, payload)
}

func (h *Hub) broadcastRawToRoom(roomCode string, payload []byte) {
	roomClients := h.Rooms[roomCode]
	log.Println("broadcast to room:", roomCode, "clients:", len(roomClients))

	for _, client := range roomClients {
		client.Send <- payload
	}
}

func (h *Hub) roomPlayers(roomCode string) []Player {
	roomClients := h.Rooms[roomCode]
	players := make([]Player, 0, len(roomClients))

	for _, client := range roomClients {
		players = append(players, Player{
			ID:       client.ID,
			Name:     client.Name,
			HP:       client.HP,
			Tokens:   client.Tokens,
			RoomCode: client.RoomCode,
		})
	}

	return players
}

func (h *Hub) alivePlayers(roomCode string) []*Client {
	aliveClients := make([]*Client, 0, len(h.Rooms[roomCode]))

	for _, client := range h.Rooms[roomCode] {
		if client.HP > 0 {
			aliveClients = append(aliveClients, client)
		}
	}

	return aliveClients
}

func buildQuestionOptions(answers []questionsvc.Answer) ([]QuestionOption, map[string]bool) {
	options := make([]QuestionOption, 0, len(answers))
	answerCorrectBy := make(map[string]bool, len(answers))
	seenAnswerIDs := make(map[string]struct{}, len(answers))

	for _, answer := range answers {
		if answer.ID == "" {
			continue
		}

		if _, exists := seenAnswerIDs[answer.ID]; exists {
			continue
		}

		seenAnswerIDs[answer.ID] = struct{}{}
		options = append(options, QuestionOption{
			ID:   answer.ID,
			Text: answer.Text,
		})
		answerCorrectBy[answer.ID] = answer.IsCorrect
	}

	return options, answerCorrectBy
}

func (h *Hub) finishGame(roomCode string, loserID string) {
	roomState := h.ensureRoomState(roomCode)
	if roomState.GameOver {
		return
	}

	aliveClients := h.alivePlayers(roomCode)
	if len(aliveClients) != 1 {
		return
	}

	winnerID := aliveClients[0].ID
	roomState.GameOver = true
	roomState.WinnerID = winnerID
	roomState.LoserID = loserID
	roomState.CurrentQuestion = nil

	h.broadcastGameOver(roomCode, winnerID, loserID)
}

func (h *Hub) resetFinishedMatch(roomCode string) {
	roomState := h.ensureRoomState(roomCode)
	roomState.CurrentQuestion = nil
	roomState.GameOver = false
	roomState.WinnerID = ""
	roomState.LoserID = ""

	for _, client := range h.Rooms[roomCode] {
		h.resetPlayerState(client)
	}
}

func (h *Hub) buildUniquePlayerName(roomCode string, requestedName string, client *Client) string {
	baseName := normalizePlayerName(requestedName)
	if !h.isPlayerNameTaken(roomCode, baseName, client) {
		return baseName
	}

	for suffix := 2; suffix < 1000; suffix++ {
		suffixText := fmt.Sprintf("%d", suffix)
		trimmedBase := []rune(baseName)
		maxBaseLength := maxPlayerNameLength - len([]rune(suffixText))
		if maxBaseLength < 1 {
			maxBaseLength = 1
		}
		if len(trimmedBase) > maxBaseLength {
			trimmedBase = trimmedBase[:maxBaseLength]
		}

		candidate := string(trimmedBase) + suffixText
		if !h.isPlayerNameTaken(roomCode, candidate, client) {
			return candidate
		}
	}

	return baseName
}

func (h *Hub) isPlayerNameTaken(roomCode string, requestedName string, excludedClient *Client) bool {
	for _, client := range h.Rooms[roomCode] {
		if client == excludedClient {
			continue
		}

		if strings.EqualFold(client.Name, requestedName) {
			return true
		}
	}

	return false
}

func (h *Hub) resetPlayerState(client *Client) {
	client.HP = initialPlayerHP
	client.Tokens = initialPlayerTokens
}

func normalizePlayerName(name string) string {
	collapsedName := strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	if collapsedName == "" {
		return defaultPlayerName
	}

	runes := []rune(collapsedName)
	if len(runes) > maxPlayerNameLength {
		return string(runes[:maxPlayerNameLength])
	}

	return collapsedName
}

func generateRoomCode(rooms map[string][]*Client) (string, error) {
	for attempts := 0; attempts < 1000; attempts++ {
		value, err := rand.Int(rand.Reader, big.NewInt(900))
		if err != nil {
			return "", err
		}

		code := fmt.Sprintf("GR-%03d", value.Int64()+100)
		if _, exists := rooms[code]; !exists {
			return code, nil
		}
	}

	return "", errors.New("could not generate unique room code")
}
