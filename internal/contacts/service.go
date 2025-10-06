package contacts

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"go-chat-backend/internal/store"
)

type Service struct{ st *store.Store }
func NewService(st *store.Store) *Service { return &Service{st: st} }

type AddInput struct{ OwnerID, ContactID, ContactEmail string }

func (s *Service) Add(ownerID, contactIDOrEmail string) (int64, error) {
	// Resolve email -> user id if needed
	cid := contactIDOrEmail
	if strings.Contains(contactIDOrEmail, "@") {
		if err := s.st.DB.QueryRowx(`SELECT id FROM users WHERE email=$1`, strings.ToLower(contactIDOrEmail)).Scan(&cid); err != nil { return 0, err }
	}
	var id int64
	err := s.st.DB.QueryRowx(`INSERT INTO contacts(owner_id, contact_id, created_at)
		VALUES($1,$2,$3)
		ON CONFLICT(owner_id, contact_id) DO UPDATE SET created_at=contacts.created_at
		RETURNING id`, ownerID, cid, time.Now().UTC()).Scan(&id)
	if err != nil { return 0, err }
	return id, nil
}

func (s *Service) List(ownerID string) ([]struct{ ContactID, Email string }, error) {
	rows, err := s.st.DB.Queryx(`SELECT u.id as contact_id, u.email FROM contacts c JOIN users u ON u.id=c.contact_id WHERE c.owner_id=$1 ORDER BY u.email`, ownerID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []struct{ ContactID, Email string }
	for rows.Next() {
		var a, b string
		if err := rows.Scan(&a, &b); err != nil { return nil, err }
		out = append(out, struct{ ContactID, Email string }{a, b})
	}
	return out, rows.Err()
}

func (s *Service) AreMutual(a, b string) (bool, error) {
	var x int
	err := s.st.DB.QueryRowx(`SELECT 1 FROM contacts WHERE owner_id=$1 AND contact_id=$2`, a, b).Scan(&x)
	if err != nil { if errors.Is(err, sql.ErrNoRows) { return false, nil } ; return false, err }
	return true, nil
}