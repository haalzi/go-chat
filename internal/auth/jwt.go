package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
)

type JWT struct{ secret []byte }
func NewJWT(secret string) *JWT { return &JWT{secret: []byte(secret)} }

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func (j *JWT) Sign(userID, email string, ttl time.Duration) (string, error) {
	claims := Claims{UserID: userID, Email: email, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl))}}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(j.secret)
}

func (j *JWT) Parse(token string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) { return j.secret, nil })
	if err != nil { return nil, err }
	if c, ok := parsed.Claims.(*Claims); ok && parsed.Valid { return c, nil }
	return nil, errors.New("invalid token")
}

// Handlers

type loginReq struct { Email, Password string }

func HandleLogin(db *sqlx.DB, jwt *JWT, w http.ResponseWriter, r *http.Request) error {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { return err }
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	var (
		id string
		ph string
	)
	err := db.QueryRowx(`SELECT id, password_hash FROM users WHERE email=$1`, req.Email).Scan(&id, &ph)
	if err != nil { if errors.Is(err, sql.ErrNoRows) { http.Error(w, "invalid credentials", http.StatusUnauthorized); return nil } ; return err }

	if !checkPassword(req.Password, ph) { http.Error(w, "invalid credentials", http.StatusUnauthorized); return nil }
	tok, err := jwt.Sign(id, req.Email, 24*time.Hour)
	if err != nil { return err }

	resp := map[string]any{"token": tok, "user": map[string]any{"id": id, "email": req.Email}}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

func HandleMe(db *sqlx.DB, jwt *JWT, w http.ResponseWriter, r *http.Request) error {
	u := r.Context().Value("user").(*Claims)
	var createdAt time.Time
	err := db.QueryRowx(`SELECT created_at FROM users WHERE id=$1`, u.UserID).Scan(&createdAt)
	if err != nil { return err }
	resp := map[string]any{"id": u.UserID, "email": u.Email, "created_at": createdAt}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

// Password helpers (replace with argon2/bcrypt in prod)
func checkPassword(plain, storedHash string) bool {
	// For demo: storedHash is bcrypt. In production ensure bcrypt.CompareHashAndPassword.
	return bcryptCompare(storedHash, plain) == nil
}

func bcryptCompare(hash, plain string) error { return nil } // replace in real build
