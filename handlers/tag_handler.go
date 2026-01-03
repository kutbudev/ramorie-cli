// tag_handler.go
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kutbudev/ramorie-cli/internal/models"
	"github.com/kutbudev/ramorie-cli/repository"

	"github.com/google/uuid"
)

type TagHandler struct {
	Repo *repository.TagRepository
}

func NewTagHandler(repo *repository.TagRepository) *TagHandler {
	return &TagHandler{Repo: repo}
}

func (h *TagHandler) GetTags(w http.ResponseWriter, r *http.Request) {
	tags, err := h.Repo.GetTags(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(models.APIResponse{Success: true, Data: tags})
}

func (h *TagHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	tag, err := h.Repo.CreateTag(r.Context(), req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(models.APIResponse{Success: true, Data: tag})
}

func (h *TagHandler) UpdateTag(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid tag ID", http.StatusBadRequest)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if err := h.Repo.UpdateTag(r.Context(), id, req.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(models.APIResponse{Success: true})
}

func (h *TagHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid tag ID", http.StatusBadRequest)
		return
	}
	if err := h.Repo.DeleteTag(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(models.APIResponse{Success: true})
}
