package examengine

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"samacode/backend/internal/httpx"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Routes : candidat connecté, sous /api/v1.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/license-categories/{id}/series", httpx.Handle(h.listSeries))
	r.Post("/exam-sessions", httpx.Handle(h.start))
	r.Get("/exam-sessions", httpx.Handle(h.history))
	r.Get("/exam-sessions/current", httpx.Handle(h.current))
	r.Get("/exam-sessions/{id}", httpx.Handle(h.get))
	r.Put("/exam-sessions/{id}/answers/{questionId}", httpx.Handle(h.answer))
	r.Post("/exam-sessions/{id}/submit", httpx.Handle(h.submit))
	r.Get("/exam-sessions/{id}/result", httpx.Handle(h.result))
	r.Get("/exam-sessions/{id}/review", httpx.Handle(h.review))
	h.practiceRoutes(r)
}

func userID(r *http.Request) string { return httpx.MustPrincipal(r.Context()).UserID }

func (h *Handler) listSeries(w http.ResponseWriter, r *http.Request) error {
	items, err := h.svc.ListSeries(r.Context(), userID(r), chi.URLParam(r, "id"))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
	return nil
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) error {
	var in StartInput
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		return err
	}
	s, err := h.svc.Start(r.Context(), userID(r), in)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, s)
	return nil
}

func (h *Handler) current(w http.ResponseWriter, r *http.Request) error {
	s, err := h.svc.Current(r.Context(), userID(r))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, s)
	return nil
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	s, err := h.svc.Get(r.Context(), userID(r), chi.URLParam(r, "id"))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, s)
	return nil
}

type answerRequest struct {
	OptionIDs []string `json:"option_ids"`
}

func (h *Handler) answer(w http.ResponseWriter, r *http.Request) error {
	var req answerRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		return err
	}
	if err := h.svc.Answer(r.Context(), userID(r), chi.URLParam(r, "id"), chi.URLParam(r, "questionId"), req.OptionIDs); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *Handler) submit(w http.ResponseWriter, r *http.Request) error {
	s, err := h.svc.Submit(r.Context(), userID(r), chi.URLParam(r, "id"))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, s)
	return nil
}

func (h *Handler) result(w http.ResponseWriter, r *http.Request) error {
	s, err := h.svc.Get(r.Context(), userID(r), chi.URLParam(r, "id"))
	if err != nil {
		return err
	}
	if s.Result == nil {
		return errNotSubmitted
	}
	httpx.WriteJSON(w, http.StatusOK, s.Result)
	return nil
}

func (h *Handler) review(w http.ResponseWriter, r *http.Request) error {
	rv, err := h.svc.Review(r.Context(), userID(r), chi.URLParam(r, "id"))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, rv)
	return nil
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) error {
	items, err := h.svc.History(r.Context(), userID(r), httpx.PageParams(r))
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
	return nil
}
