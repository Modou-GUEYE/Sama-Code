package examseries

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"samacode/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// AdminRoutes : montées sous /api/v1/admin, réservées aux rôles de contenu.
func (h *Handler) AdminRoutes(r chi.Router) {
	r.Get("/exam-series", httpx.Handle(h.list))
	r.Post("/exam-series", httpx.Handle(h.create))
	r.Get("/exam-series/{id}", httpx.Handle(h.get))
	r.Put("/exam-series/{id}/questions", httpx.Handle(h.setQuestions))
	r.Post("/exam-series/{id}/status", httpx.Handle(h.setStatus))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	items, err := h.svc.List(r.Context(), r.URL.Query().Get("license_category_id"))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
	return nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	d, err := h.svc.Get(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, d)
	return nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	var in CreateInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	d, err := h.svc.Create(r.Context(), httpx.MustPrincipal(r.Context()), in)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, d)
	return nil
}

type questionsRequest struct {
	QuestionIDs []string `json:"question_ids"`
}

func (h *Handler) setQuestions(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	var req questionsRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	d, err := h.svc.SetQuestions(r.Context(), httpx.MustPrincipal(r.Context()), id, req.QuestionIDs)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, d)
	return nil
}

type statusRequest struct {
	Status string `json:"status"`
}

func (h *Handler) setStatus(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	var req statusRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	d, err := h.svc.SetStatus(r.Context(), httpx.MustPrincipal(r.Context()), id, req.Status)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, d)
	return nil
}
