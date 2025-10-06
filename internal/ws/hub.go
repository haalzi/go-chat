package ws

import (
	"encoding/json"
	"sync"
	"time"

	"go-chat-backend/internal/conversations"
)

type Hub struct {
	Conversations *conversations.Service
	rooms map[string]map[*Client]bool
	mu    sync.RWMutex
}

func NewHub(msgSvc any, convSvc *conversations.Service) *Hub {
	return &Hub{Conversations: convSvc, rooms: make(map[string]map[*Client]bool)}
}

func (h *Hub) Run() { /* placeholder for future housekeeping */ }

func (h *Hub) Join(convID string, c *Client) {
	h.mu.Lock(); defer h.mu.Unlock()
	if h.rooms[convID] == nil { h.rooms[convID] = make(map[*Client]bool) }
	h.rooms[convID][c] = true
}

func (h *Hub) Leave(convID string, c *Client) {
	h.mu.Lock(); defer h.mu.Unlock()
	if m := h.rooms[convID]; m != nil { delete(m, c); if len(m)==0 { delete(h.rooms, convID) } }
}

func (h *Hub) Broadcast(convID string, payload any) {
	b, _ := json.Marshal(payload)
	h.mu.RLock(); conns := h.rooms[convID]; h.mu.RUnlock()
	for c := range conns { select { case c.send <- b: default: go c.Close() } }
}

const (
	writeWait = 10 * time.Second
	pongWait  = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)