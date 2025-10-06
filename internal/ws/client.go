package ws

import (
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
	hub  *Hub
	convID string
	send chan []byte
}

func newClient(h *Hub, convID string, conn *websocket.Conn) *Client {
	return &Client{conn: conn, hub: h, convID: convID, send: make(chan []byte, 256)}
}

func (c *Client) readPump() {
	defer c.Close()
	c.conn.SetReadLimit(8 * 1024)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil { return }
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func(){ ticker.Stop(); c.conn.Close() }()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok { c.conn.WriteMessage(websocket.CloseMessage, []byte{}); return }
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil { return }
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil { return }
		}
	}
}

func (c *Client) Close() {
	c.hub.Leave(c.convID, c)
	close(c.send)
	_ = c.conn.Close()
}