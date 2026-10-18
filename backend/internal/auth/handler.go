package auth

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"samacode/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Routes publiques de /api/v1/auth.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/otp/request", httpx.Handle(h.requestOTP))
	r.Post("/otp/verify", httpx.Handle(h.verifyOTP))
	r.Post("/register", httpx.Handle(h.register))
	r.Post("/login", httpx.Handle(h.login))
	r.Post("/pin/reset", httpx.Handle(h.resetPIN))
	r.Post("/refresh", httpx.Handle(h.refresh))
	r.Post("/logout", httpx.Handle(h.logout))
}

type otpRequest struct {
	PhoneNumber string  `json:"phone_number"`
	Purpose     Purpose `json:"purpose"`
}

func (h *Handler) requestOTP(w http.ResponseWriter, r *http.Request) error {
	var req otpRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	if err := h.svc.RequestOTP(r.Context(), req.PhoneNumber, req.Purpose); err != nil {
		return err
	}
	resp := map[string]any{
		"status":             "sent",
		"expires_in_seconds": int(h.svc.cfg.OTPTTL.Seconds()),
	}
	if code, ok := h.svc.takeDevCode(req.PhoneNumber); ok {
		resp["dev_code"] = code
	}
	httpx.WriteJSON(w, http.StatusAccepted, resp)
	return nil
}

type verifyRequest struct {
	PhoneNumber string  `json:"phone_number"`
	Purpose     Purpose `json:"purpose"`
	Code        string  `json:"code"`
}

func (h *Handler) verifyOTP(w http.ResponseWriter, r *http.Request) error {
	var req verifyRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	res, err := h.svc.VerifyOTP(r.Context(), req.PhoneNumber, req.Purpose, req.Code)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, res)
	return nil
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) error {
	var req RegisterInput
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	pair, err := h.svc.Register(r.Context(), req)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, pair)
	return nil
}

type loginRequest struct {
	PhoneNumber string `json:"phone_number"`
	PIN         string `json:"pin"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) error {
	var req loginRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	pair, err := h.svc.Login(r.Context(), req.PhoneNumber, req.PIN)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, pair)
	return nil
}

type resetPINRequest struct {
	VerificationToken string `json:"verification_token"`
	PIN               string `json:"pin"`
}

func (h *Handler) resetPIN(w http.ResponseWriter, r *http.Request) error {
	var req resetPINRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	pair, err := h.svc.ResetPIN(r.Context(), req.VerificationToken, req.PIN)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, pair)
	return nil
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) error {
	var req refreshRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	pair, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, pair)
	return nil
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) error {
	var req refreshRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Middleware exige un jeton d'accès valide (« Authorization: Bearer … »).
func (t *Tokens) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || raw == "" {
			httpx.WriteError(w, r, httpx.ErrUnauthenticated)
			return
		}
		p, err := t.ParseAccess(raw)
		if err != nil {
			httpx.WriteError(w, r, httpx.ErrUnauthenticated)
			return
		}
		next.ServeHTTP(w, r.WithContext(httpx.WithPrincipal(r.Context(), p)))
	})
}
