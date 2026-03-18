package ws

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type WSHandler struct {
	hub *Hub
}

// 🔹 Estado global (simple para demo)
var players = make(map[string]int)   // tokens
var playersHP = make(map[string]int) // HP

func NewWSHandler(hub *Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// 🔹 Mensajes
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type Answer struct {
	Answer string `json:"answer"`
}

type Attack struct {
	Target string `json:"target"`
}

// 🔹 Conexión
func (h *WSHandler) HandleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	playerID := r.URL.Query().Get("playerId")

	client := &Client{
		ID:   playerID,
		Conn: conn,
		Send: make(chan []byte),
	}

	// inicializar estado
	players[playerID] = 0
	playersHP[playerID] = 100

	h.hub.Register <- client

	go h.readPump(client)
	go h.writePump(client)
}

// 🔹 Leer mensajes
func (h *WSHandler) readPump(c *Client) {
	defer func() {
		h.hub.Unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		json.Unmarshal(message, &msg)

		switch msg.Type {

		case "START":
			h.sendQuestion()

		case "ANSWER":
			h.handleAnswer(c, msg.Data)

		case "ATTACK":
			h.handleAttack(c, msg.Data)
		}
	}
}

// 🔹 Escribir mensajes
func (h *WSHandler) writePump(c *Client) {
	for msg := range c.Send {
		err := c.Conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			break
		}
	}
}

// 🔹 Enviar pregunta
func (h *WSHandler) sendQuestion() {
	resp := map[string]interface{}{
		"type": "QUESTION",
		"data": map[string]interface{}{
			"question": "2+2",
			"options":  []string{"3", "4", "5"},
		},
	}

	b, _ := json.Marshal(resp)
	h.hub.Broadcast <- b
}

// 🔹 Manejar respuesta
func (h *WSHandler) handleAnswer(c *Client, data []byte) {
	var ans Answer
	json.Unmarshal(data, &ans)

	correct := ans.Answer == "4"

	if correct {
		players[c.ID]++
	}

	resp := map[string]interface{}{
		"type": "RESULT",
		"data": map[string]interface{}{
			"correct": correct,
			"tokens":  players[c.ID],
		},
	}

	b, _ := json.Marshal(resp)
	c.Send <- b
}

// 🔥 Manejar ataque
func (h *WSHandler) handleAttack(c *Client, data []byte) {
	var atk Attack
	json.Unmarshal(data, &atk)

	damage := 10

	// validar target
	hp, ok := playersHP[atk.Target]
	if !ok {
		return
	}

	hp -= damage
	if hp < 0 {
		hp = 0
	}

	playersHP[atk.Target] = hp

	resp := map[string]interface{}{
		"type": "ATTACK_RESULT",
		"data": map[string]interface{}{
			"from":   c.ID,
			"to":     atk.Target,
			"damage": damage,
			"hp":     hp,
		},
	}

	b, _ := json.Marshal(resp)
	h.hub.Broadcast <- b
}
