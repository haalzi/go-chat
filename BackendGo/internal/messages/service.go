package messages

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"go-chat-backend/internal/conversations"
	"go-chat-backend/internal/store"
)

type Service struct{ st *store.Store }
func NewService(st *store.Store) *Service { return &Service{st: st} }

func (s *Service) List(convID string, limit int, before *time.Time) (*sqlx.Rows, error) {
	q := `SELECT id, conversation_id, sender_id, text, created_at, expires_at, deleted_at
		FROM messages WHERE conversation_id=$1 AND (deleted_at IS NULL) AND (expires_at IS NULL OR expires_at>now())`
	args := []any{convID}
	if before != nil { q += " AND created_at < $2"; args = append(args, *before) }
	q += " ORDER BY created_at DESC LIMIT $3"
	args = append(args, limit)
	return s.st.DB.Queryx(q, args...)
}

func (s *Service) Create(convSvc *conversations.Service, convID, senderID, text string, ttl time.Duration) (int64, time.Time, *time.Time, error) {
	if len(text) == 0 || len(text) > 5*1024 { return 0, time.Time{}, nil, errors.New("invalid text size") }
	ok, err := convSvc.EnsureParticipant(convID, senderID)
	if err != nil { return 0, time.Time{}, nil, err }
	if !ok { return 0, time.Time{}, nil, errors.New("not a participant") }

	// If direct: check contacts policy
	isDirect, err := convSvc.IsDirect(convID)
	if err != nil { return 0, time.Time{}, nil, err }
	if isDirect {
		peer, err := convSvc.PeerInDirect(convID, senderID)
		if err != nil { return 0, time.Time{}, nil, err }
		var x int
		if err := s.st.DB.QueryRowx(`SELECT 1 FROM contacts WHERE owner_id=$1 AND contact_id=$2`, senderID, peer).Scan(&x); err != nil {
			if errors.Is(err, sql.ErrNoRows) { return 0, time.Time{}, nil, errors.New("peer not in contacts") }
			return 0, time.Time{}, nil, err
		}
	}

	createdAt := time.Now().UTC()
	ttl = ClampTTL(ttl)
	var expires *time.Time
	if ttl > 0 { e := createdAt.Add(ttl); expires = &e }

	var id int64
	err = s.st.DB.QueryRowx(`INSERT INTO messages(conversation_id, sender_id, text, created_at, expires_at)
		VALUES($1,$2,$3,$4,$5) RETURNING id`, convID, senderID, strings.TrimSpace(text), createdAt, expires).Scan(&id)
	if err != nil { return 0, time.Time{}, nil, err }
	return id, createdAt, expires, nil
}

func (s *Service) SoftDelete(id int64, userID string) error {
	// Only allow sender to soft-delete their message (can be expanded to moderators)
	res, err := s.st.DB.Exec(`UPDATE messages SET deleted_at=now() WHERE id=$1 AND sender_id=$2 AND deleted_at IS NULL`, id, userID)
	if err != nil { return err }
	a, _ := res.RowsAffected()
	if a == 0 { return errors.New("not found or not allowed") }
	return nil
}

func StartPurger(db *sqlx.DB, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for range t.C {
		_, _ = db.Exec(`DELETE FROM messages WHERE (expires_at IS NOT NULL AND expires_at < now()) OR (deleted_at IS NOT NULL AND deleted_at < now() - interval '24 hours')`)
	}
}