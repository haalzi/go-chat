package conversations

import (
	"database/sql"
	"errors"
	"sort"
	"time"

	"go-chat-backend/internal/store"
)

type Service struct{ st *store.Store }
func NewService(st *store.Store) *Service { return &Service{st: st} }

func (s *Service) StartOrGetDirect(a, b string) (string, error) {
	ids := []string{a, b}; sort.Strings(ids)
	var convID string
	err := s.st.DB.QueryRowx(`SELECT c.id FROM conversations c
		JOIN conversation_participants p1 ON p1.conversation_id=c.id AND p1.user_id=$1
		JOIN conversation_participants p2 ON p2.conversation_id=c.id AND p2.user_id=$2
		WHERE c.type='direct' LIMIT 1`, ids[0], ids[1]).Scan(&convID)
	if err == nil { return convID, nil }
	if !errors.Is(err, sql.ErrNoRows) { return "", err }

	// create
	err = s.st.DB.QueryRowx(`INSERT INTO conversations(id, type, created_at) VALUES(gen_random_uuid(),'direct',$1) RETURNING id`, time.Now().UTC()).Scan(&convID)
	if err != nil { return "", err }
	// participants
	if _, err := s.st.DB.Exec(`INSERT INTO conversation_participants(conversation_id,user_id) VALUES($1,$2),($1,$3) ON CONFLICT DO NOTHING`, convID, ids[0], ids[1]); err != nil { return "", err }
	return convID, nil
}

func (s *Service) EnsureParticipant(convID, userID string) (bool, error) {
	var x int
	err := s.st.DB.QueryRowx(`SELECT 1 FROM conversation_participants WHERE conversation_id=$1 AND user_id=$2`, convID, userID).Scan(&x)
	if err != nil { if errors.Is(err, sql.ErrNoRows) { return false, nil } ; return false, err }
	return true, nil
}

func (s *Service) ListForUser(userID string, limit, offset int) ([]struct{ ID, Type string; CreatedAt time.Time }, error) {
	rows, err := s.st.DB.Queryx(`SELECT c.id,c.type,c.created_at FROM conversations c
		JOIN conversation_participants p ON p.conversation_id=c.id
		WHERE p.user_id=$1 ORDER BY c.created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []struct{ ID, Type string; CreatedAt time.Time }
	for rows.Next() { var id, t string; var ct time.Time; if err := rows.Scan(&id, &t, &ct); err != nil { return nil, err }; out = append(out, struct{ ID, Type string; CreatedAt time.Time }{id,t,ct}) }
	return out, rows.Err()
}

func (s *Service) IsDirect(convID string) (bool, error) {
	var t string
	err := s.st.DB.QueryRowx(`SELECT type FROM conversations WHERE id=$1`, convID).Scan(&t)
	if err != nil { return false, err }
	return t == "direct", nil
}

func (s *Service) PeerInDirect(convID, self string) (string, error) {
	var peer string
	err := s.st.DB.QueryRowx(`SELECT p2.user_id FROM conversation_participants p1 JOIN conversation_participants p2 ON p1.conversation_id=p2.conversation_id AND p1.user_id<>p2.user_id WHERE p1.conversation_id=$1 AND p1.user_id=$2 LIMIT 1`, convID, self).Scan(&peer)
	return peer, err
}