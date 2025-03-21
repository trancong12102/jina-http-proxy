package key

import (
	"context"
	"encoding/json"
	"net/http"
)

type InsertKeyRequest struct {
	Key string `json:"key"`
}

// Convert InsertKeyRequest to InsertKeyParams
func (r InsertKeyRequest) ToParams() InsertKeyParams {
	return InsertKeyParams(r)
}

type KeyBiz interface {
	GetKeyStats(ctx context.Context) (*KeyStats, error)
	InsertKey(ctx context.Context, params InsertKeyParams) error
}

type KeyHandler struct {
	service KeyBiz
}

func (h *KeyHandler) GetKeyStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetKeyStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *KeyHandler) InsertKey(w http.ResponseWriter, r *http.Request) {
	var req InsertKeyRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.service.InsertKey(r.Context(), req.ToParams())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func NewKeyHandler(service KeyBiz) *KeyHandler {
	return &KeyHandler{service: service}
}
