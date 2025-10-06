package ws

import (
	"net/http"

	"github.com/gorilla/websocket"
	"go-chat-backend/internal/auth"
	"go-chat-backend/internal/conversations"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, restrict to configured origins.
		return true
	},
}

func Handle(h *Hub, jwt *auth.JWT, convSvc *conversations.Service, w http.ResponseWriter, r *http.Request) {
	convID := r.URL.Query().Get("conversation_id")
	tok := r.URL.Query().Get("token")
	claims, err := jwt.Parse(tok)
	if err != nil { http.Error(w, "invalid token", http.StatusUnauthorized); return }
	ok, err := convSvc.EnsureParticipant(convID, claims.UserID)
	if err != nil || !ok { http.Error(w, "not in conversation", http.StatusForbidden); return }

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil { return }
	c := newClient(h, convID, conn)
	h.Join(convID, c)
	go c.writePump()
	go c.readPump()
}