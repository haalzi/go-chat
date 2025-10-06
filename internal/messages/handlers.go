package messages

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-chat-backend/internal/auth"
	"go-chat-backend/internal/ws"
)

type createReq struct{ ConversationID, Text string; TTLSeconds *int64 `json:"ttl_seconds"` }

func HandleList(s *Service, jwt *auth.JWT, w http.ResponseWriter, r *http.Request) error {
	convID := r.URL.Query().Get("conversation_id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit")); if limit<=0||limit>200 { limit=50 }
	var before *time.Time
	if v := r.URL.Query().Get("before"); v != "" { if ts, err := time.Parse(time.RFC3339Nano, v); err==nil { before=&ts } }
	rows, err := s.List(convID, limit, before)
	if err != nil { return err }
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var m struct{
			ID int64 `db:"id"`
			ConversationID, SenderID, Text string
			CreatedAt time.Time
			ExpiresAt, DeletedAt *time.Time
		}
		if err := rows.StructScan(&m); err != nil { return err }
		out = append(out, map[string]any{
			"id": m.ID, "conversation_id": m.ConversationID, "sender_id": m.SenderID, "text": m.Text, "created_at": m.CreatedAt, "expires_at": m.ExpiresAt,
		})
	}
	return json.NewEncoder(w).Encode(out)
}

func HandleCreate(s *Service, hub *ws.Hub, jwt *auth.JWT, w http.ResponseWriter, r *http.Request) error {
	u := r.Context().Value("user").(*auth.Claims)
	var req createReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { return err }
	var ttl time.Duration
	if req.TTLSeconds != nil { ttl = time.Duration(*req.TTLSeconds) * time.Second }
	id, created, expires, err := s.Create(hub.Conversations, req.ConversationID, u.UserID, req.Text, ttl)
	if err != nil { return err }
	payload := map[string]any{"type":"message","id":id,"text":req.Text,"sender_id":u.UserID,"created_at":created,"expires_at":expires}
	hub.Broadcast(req.ConversationID, payload)
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(payload)
}

func HandleDelete(s *Service, jwt *auth.JWT, w http.ResponseWriter, r *http.Request) error {
	u := r.Context().Value("user").(*auth.Claims)
	idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if err := s.SoftDelete(id, u.UserID); err != nil { return err }
	w.WriteHeader(http.StatusNoContent)
	return nil
}