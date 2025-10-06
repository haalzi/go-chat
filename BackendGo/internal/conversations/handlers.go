package conversations

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go-chat-backend/internal/auth"
	"go-chat-backend/internal/contacts"
)

type startReq struct{ PeerID string `json:"peer_id"` }

func HandleStartOrGetDirect(s *Service, cs *contacts.Service, jwt *auth.JWT, w http.ResponseWriter, r *http.Request) error {
	u := r.Context().Value("user").(*auth.Claims)
	var req startReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { return err }
	ok, err := cs.AreMutual(u.UserID, req.PeerID)
	if err != nil { return err }
	if !ok { http.Error(w, "peer is not in contacts", http.StatusForbidden); return nil }
	id, err := s.StartOrGetDirect(u.UserID, req.PeerID)
	if err != nil { return err }
	return json.NewEncoder(w).Encode(map[string]string{"id": id, "type":"direct"})
}

func HandleList(s *Service, jwt *auth.JWT, w http.ResponseWriter, r *http.Request) error {
	u := r.Context().Value("user").(*auth.Claims)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit")); if limit<=0||limit>100 { limit=50 }
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := s.ListForUser(u.UserID, limit, offset)
	if err != nil { return err }
	return json.NewEncoder(w).Encode(items)
}