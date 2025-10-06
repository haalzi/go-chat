package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	"go-chat-backend/internal/auth"
	"go-chat-backend/internal/contacts"
	"go-chat-backend/internal/conversations"
	"go-chat-backend/internal/httputil"
	"go-chat-backend/internal/messages"
	"go-chat-backend/internal/store"
	"go-chat-backend/internal/ws"
)

//go:embed web/*
var webFS embed.FS
func main() {
	// ENV
	_ = godotenv.Load() 
	addr := getEnv("ADDR", ":8080")
	jwtSecret := mustEnv("JWT_SECRET")
	allowedOrigins := getEnv("ALLOWED_ORIGINS", "*") // e.g. "https://yourapp.com"
	purgeEvery := getEnvInt("PURGE_INTERVAL_SECONDS", 300)

	// DB
	dsn := mustEnv("DATABASE_URL") // e.g. postgres://user:pass@host:5432/dbname?sslmode=disable
	db, err := sqlx.Open("pgx", dsn)
	if err != nil { log.Fatalf("db open: %v", err) }
	if err := db.Ping(); err != nil { log.Fatalf("db ping: %v", err) }

	// Stores & services
	st := store.New(db)
	jwt := auth.NewJWT(jwtSecret)
	msgSvc := messages.NewService(st)
	convSvc := conversations.NewService(st)
	contactSvc := contacts.NewService(st)

	// WS hub per conversation
	roomHub := ws.NewHub(msgSvc, convSvc)
	go roomHub.Run()

	// Background purger
	go messages.StartPurger(db, time.Duration(purgeEvery)*time.Second)

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Auth
	mux.Handle("/api/auth/login", httputil.JSONHandler(func(w http.ResponseWriter, r *http.Request) error {
		return auth.HandleLogin(db, jwt, w, r)
	}))

	// Protected routes
	protected := httputil.Chain(
		httputil.JWTAuth(jwt),
		httputil.RateLimit(100, time.Minute), // naive leaky bucket per IP
	)

	mux.Handle("/api/users/me", protected(httputil.JSONHandler(func(w http.ResponseWriter, r *http.Request) error {
		return auth.HandleMe(db, jwt, w, r)
	})))

	mux.Handle("/api/contacts", protected(httputil.JSONHandler(func(w http.ResponseWriter, r *http.Request) error {
		switch r.Method {
		case http.MethodGet:
			return contacts.HandleList(contactSvc, jwt, w, r)
		case http.MethodPost:
			return contacts.HandleAdd(contactSvc, jwt, w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return nil
		}
	})))

	mux.Handle("/api/conversations/direct", protected(httputil.JSONHandler(func(w http.ResponseWriter, r *http.Request) error {
		if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return nil }
		return conversations.HandleStartOrGetDirect(convSvc, contactSvc, jwt, w, r)
	})))

	mux.Handle("/api/conversations", protected(httputil.JSONHandler(func(w http.ResponseWriter, r *http.Request) error {
		if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return nil }
		return conversations.HandleList(convSvc, jwt, w, r)
	})))

	mux.Handle("/api/messages", protected(httputil.JSONHandler(func(w http.ResponseWriter, r *http.Request) error {
		switch r.Method {
		case http.MethodGet:
			return messages.HandleList(msgSvc, jwt, w, r)
		case http.MethodPost:
			return messages.HandleCreate(msgSvc, roomHub, jwt, w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return nil
		}
	})))

	mux.Handle("/api/messages/", protected(httputil.JSONHandler(func(w http.ResponseWriter, r *http.Request) error {
		if r.Method != http.MethodDelete { w.WriteHeader(http.StatusMethodNotAllowed); return nil }
		return messages.HandleDelete(msgSvc, jwt, w, r)
	})))

	// WS endpoint with JWT & participant check inside handler
	mux.Handle("/ws", httputil.CORS(allowedOrigins)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws.Handle(roomHub, jwt, convSvc, w, r)
	})))

	// --- Static (embedded) tanpa loop ---
	sub, err := fs.Sub(webFS, "web")
	if err != nil { log.Fatalf("embed sub: %v", err) }

	// layani /web/... dari folder embed "web"
	mux.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.FS(sub))))

	// root diarahkan sekali ke /web/
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/web/", http.StatusFound)
	})

	server := &http.Server{
		Addr:    addr,
		Handler: httputil.CORS(allowedOrigins)(mux),
	}

	log.Printf("listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" { log.Fatalf("missing env %s", k) }
	return v
}
func getEnv(k, def string) string { if v := os.Getenv(k); v != "" { return v }; return def }
func getEnvInt(k string, def int) int { if v := os.Getenv(k); v != "" { if i, err := strconv.Atoi(v); err == nil { return i } }; return def }
