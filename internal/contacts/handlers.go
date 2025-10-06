package contacts

import (
	"encoding/json"
	"net/http"

	"go-chat-backend/internal/auth"
)

type addReq struct { ContactEmail string `json:"contact_email"`; ContactID string `json:"contact_id"` }

func HandleAdd(s *Service, jwt *auth.JWT, w http.ResponseWriter, r *http.Request) error {
	u := r.Context().Value("user").(*auth.Claims)
	var req addReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { return err }
	idOrEmail := req.ContactID
	if idOrEmail == "" { idOrEmail = req.ContactEmail }
	_, err := s.Add(u.UserID, idOrEmail)
	if err != nil { return err }
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(map[string]string{"status":"ok"})
}

func HandleList(s *Service, jwt *auth.JWT, w http.ResponseWriter, r *http.Request) error {
	u := r.Context().Value("user").(*auth.Claims)
	items, err := s.List(u.UserID)
	if err != nil { return err }
	return json.NewEncoder(w).Encode(items)
}