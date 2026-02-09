package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/nfedorov/port_server/internal/config"
	"github.com/nfedorov/port_server/internal/model"
	"github.com/nfedorov/port_server/internal/store"
)

type Handler struct {
	store   store.Store
	portMin int
	portMax int
}

func New(s store.Store) *Handler {
	return &Handler{
		store:   s,
		portMin: config.DefaultPortMin,
		portMax: config.DefaultPortMax,
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/healthz", h.Health)
	r.Route("/v1", func(r chi.Router) {
		r.Post("/allocations", h.Allocate)
		r.Get("/allocations", h.List)
		r.Delete("/allocations", h.ReleaseByFilter)
		r.Delete("/allocations/{id}", h.ReleaseByID)
		r.Get("/ports/{port}", h.CheckPort)
	})
	return r
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Allocate(w http.ResponseWriter, r *http.Request) {
	var req model.AllocateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "invalid JSON"})
		return
	}
	if req.App == "" || req.Instance == "" || req.Service == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "app, instance, and service are required"})
		return
	}

	alloc, err := h.store.Allocate(req, h.portMin, h.portMax)
	if err == store.ErrServiceAllocated {
		writeJSON(w, http.StatusConflict, model.ErrorResponse{
			Error:  "service already allocated",
			Holder: alloc,
		})
		return
	}
	if err == store.ErrPortTaken {
		writeJSON(w, http.StatusConflict, model.ErrorResponse{
			Error:  "port already allocated",
			Holder: alloc,
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, alloc)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	f := store.Filter{
		App:      r.URL.Query().Get("app"),
		Instance: r.URL.Query().Get("instance"),
		Service:  r.URL.Query().Get("service"),
	}

	allocs, err := h.store.List(f)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
		return
	}
	if allocs == nil {
		allocs = []model.Allocation{}
	}
	writeJSON(w, http.StatusOK, allocs)
}

func (h *Handler) ReleaseByFilter(w http.ResponseWriter, r *http.Request) {
	var req model.ReleaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "invalid JSON"})
		return
	}

	f := store.Filter{
		App:      req.App,
		Instance: req.Instance,
		Service:  req.Service,
		Port:     req.Port,
	}

	n, err := h.store.DeleteByFilter(f)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]int64{"deleted": n})
}

func (h *Handler) ReleaseByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "invalid id"})
		return
	}

	if err := h.store.DeleteByID(id); err == store.ErrNotFound {
		writeJSON(w, http.StatusNotFound, model.ErrorResponse{Error: "allocation not found"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) CheckPort(w http.ResponseWriter, r *http.Request) {
	portStr := chi.URLParam(r, "port")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "invalid port"})
		return
	}

	alloc, err := h.store.GetByPort(port)
	if err == store.ErrNotFound {
		writeJSON(w, http.StatusOK, model.PortStatus{Port: port, Available: true})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, model.PortStatus{Port: port, Available: false, Holder: alloc})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
