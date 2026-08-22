// Package httpapi is the HTTP adapter — DTOs, handlers, error mapping.
// Depends on service.UserService and port.TokenVerifier only.
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/PooriChaiya/backend-challenge-a1/internal/domain"
	"github.com/PooriChaiya/backend-challenge-a1/internal/service"
)

type Handlers struct {
	svc *service.UserService
}

func NewHandlers(svc *service.UserService) *Handlers { return &Handlers{svc: svc} }

// --- DTOs ---

type userResp struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func toResp(u *domain.User) userResp {
	return userResp{ID: u.ID, Name: u.Name, Email: u.Email, CreatedAt: u.CreatedAt}
}

type registerReq struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type updateReq struct {
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrDuplicateEmail):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrInvalidCredentials):
		status = http.StatusUnauthorized
	case errors.Is(err, domain.ErrInvalidInput):
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.ErrInvalidInput
	}
	return nil
}

// --- handlers ---

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	u, err := h.svc.Register(r.Context(), service.RegisterInput{Name: req.Name, Email: req.Email, Password: req.Password})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toResp(u))
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	tok, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": tok})
}

func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	us, err := h.svc.List(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	out := make([]userResp, len(us))
	for i, u := range us {
		out[i] = toResp(&u)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) Get(w http.ResponseWriter, r *http.Request) {
	u, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResp(u))
}

func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	var req updateReq
	if err := decode(r, &req); err != nil {
		writeErr(w, err)
		return
	}
	u, err := h.svc.Update(r.Context(), r.PathValue("id"), service.UpdateInput{Name: req.Name, Email: req.Email})
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toResp(u))
}

func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
