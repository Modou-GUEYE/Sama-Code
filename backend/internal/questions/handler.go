package questions

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
	r.Get("/questions", httpx.Handle(h.list))
	r.Post("/questions", httpx.Handle(h.create))
	r.Get("/questions/{id}", httpx.Handle(h.get))
	r.Put("/questions/{id}", httpx.Handle(h.update))
	r.Post("/questions/{id}/status", httpx.Handle(h.setStatus))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	page := httpx.PageParams(r)
	items, total, err := h.svc.List(r.Context(), r.URL.Query().Get("status"), r.URL.Query().Get("search"), page)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page.Number, "page_size": page.Size})
	return nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	q, err := h.svc.Get(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, q)
	return nil
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	var in Input
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	q, err := h.svc.Create(r.Context(), httpx.MustPrincipal(r.Context()), in)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, q)
	return nil
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) error {
	id, err := httpx.UUIDParam(r, "id")
	if err != nil {
		return err
	}
	var in Input
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	q, err := h.svc.Update(r.Context(), httpx.MustPrincipal(r.Context()), id, in)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, q)
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
	q, err := h.svc.SetStatus(r.Context(), httpx.MustPrincipal(r.Context()), id, req.Status)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, q)
	return nil
}
