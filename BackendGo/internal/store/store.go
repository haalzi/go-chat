package store

import "github.com/jmoiron/sqlx"

type Store struct{ DB *sqlx.DB }
func New(db *sqlx.DB) *Store { return &Store{DB: db} }