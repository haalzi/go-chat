package httputil

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"context"

	"go-chat-backend/internal/auth"
)

// JSONHandler wraps handlers that return error

type JSONHandler func(http.ResponseWriter, *http.Request) error

func (h JSONHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := h(w, r); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
	}
}

// Chain middlewares

type Middleware func(http.Handler) http.Handler

func Chain(mws ...Middleware) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- { h = mws[i](h) }
		return h
	}
}

// JWTAuth protects /api/*

func JWTAuth(jwt *auth.JWT) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") { http.Error(w, "missing bearer", http.StatusUnauthorized); return }
			tok := strings.TrimPrefix(h, "Bearer ")
			claims, err := jwt.Parse(tok)
			if err != nil { http.Error(w, "invalid token", http.StatusUnauthorized); return }
			r = r.WithContext(context.WithValue(r.Context(), "user", claims))
			next.ServeHTTP(w, r)
		})
	}
}

// CORS

func CORS(allowed string) Middleware {
	allowAll := allowed == "*"
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowAll || strings.Contains(allowed, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
			}
			if r.Method == http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
			next.ServeHTTP(w, r)
		})
	}
}

// Simple IP rate limit

type tokenBucket struct{ mu sync.Mutex; tokens int; last time.Time }

func RateLimit(n int, per time.Duration) Middleware {
	buckets := make(map[string]*tokenBucket)
	var mu sync.Mutex
	refill := func(b *tokenBucket) {
		elapsed := time.Since(b.last)
		add := int(elapsed / per)
		if add > 0 { b.tokens = min(n, b.tokens+add); b.last = b.last.Add(time.Duration(add) * per) }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			mu.Lock()
			b := buckets[ip]
			if b == nil { b = &tokenBucket{tokens: n, last: time.Now()}; buckets[ip] = b }
			b.mu.Lock(); mu.Unlock()
			refill(b)
			if b.tokens <= 0 { b.mu.Unlock(); http.Error(w, "rate limit", http.StatusTooManyRequests); return }
			b.tokens--; b.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

func min(a, b int) int {
    if a < b { return a }
    return b
}