package httpapi

import (
	"net/http"

	"github.com/PooriChaiya/backend-challenge-a1/internal/port"
)

func NewRouter(h *Handlers, v port.TokenVerifier) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.Health)
	mux.HandleFunc("POST /register", h.Register)
	mux.HandleFunc("POST /login", h.Login)

	auth := Auth(v)
	mux.Handle("GET /users", auth(http.HandlerFunc(h.List)))
	mux.Handle("GET /users/{id}", auth(http.HandlerFunc(h.Get)))
	mux.Handle("PUT /users/{id}", auth(http.HandlerFunc(h.Update)))
	mux.Handle("DELETE /users/{id}", auth(http.HandlerFunc(h.Delete)))

	return Logging(mux)
}
