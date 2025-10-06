package models

import "time"

type User struct {
	ID          string    `db:"id" json:"id"`
	Email       string    `db:"email" json:"email"`
	PasswordHash string   `db:"password_hash" json:"-"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type Contact struct {
	ID        int64     `db:"id" json:"id"`
	OwnerID   string    `db:"owner_id" json:"owner_id"`
	ContactID string    `db:"contact_id" json:"contact_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type Conversation struct {
	ID        string    `db:"id" json:"id"`
	Type      string    `db:"type" json:"type"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type Message struct {
	ID             int64      `db:"id" json:"id"`
	ConversationID string     `db:"conversation_id" json:"conversation_id"`
	SenderID       string     `db:"sender_id" json:"sender_id"`
	Text           string     `db:"text" json:"text"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
	ExpiresAt      *time.Time `db:"expires_at" json:"expires_at"`
	DeletedAt      *time.Time `db:"deleted_at" json:"-"`
}