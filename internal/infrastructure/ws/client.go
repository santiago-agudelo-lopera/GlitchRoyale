package ws

import "github.com/gorilla/websocket"

type Client struct {
	ID       string
	Name     string
	HP       int
	Tokens   int
	Conn     *websocket.Conn
	Send     chan []byte
	RoomCode string
}
