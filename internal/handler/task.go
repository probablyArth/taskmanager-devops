package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/arth/taskmanager/internal/model"
)

type TaskStore interface {
	Create(ctx context.Context, req *model.CreateTaskRequest) (*model.Task, error)
	Get(ctx context.Context, id string) (*model.Task, error)
	List(ctx context.Context) ([]*model.Task, error)
	Update(ctx context.Context, id string, req *model.UpdateTaskRequest) (*model.Task, error)
	Delete(ctx context.Context, id string) error
	Health(ctx context.Context) error
}

type TaskHandler struct {
	store  TaskStore
	logger *slog.Logger
}

func NewTaskHandler(store TaskStore, logger *slog.Logger) *TaskHandler {
	return &TaskHandler{
		store:  store,
		logger: logger,
	}
}

type Response struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func (h *TaskHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /api/v1/tasks", h.List)
	mux.HandleFunc("GET /api/v1/tasks/{id}", h.Get)
	mux.HandleFunc("POST /api/v1/tasks", h.Create)
	mux.HandleFunc("PUT /api/v1/tasks/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", h.Delete)
}

func (h *TaskHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.store.Health(ctx); err != nil {
		h.logger.Error("health check failed", "error", err)
		h.writeError(w, http.StatusServiceUnavailable, "database unhealthy")
		return
	}

	h.writeJSON(w, http.StatusOK, Response{Data: map[string]string{"status": "healthy"}})
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tasks, err := h.store.List(ctx)
	if err != nil {
		h.logger.Error("failed to list tasks", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	if tasks == nil {
		tasks = []*model.Task{}
	}

	h.writeJSON(w, http.StatusOK, Response{Data: tasks})
}

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "task id is required")
		return
	}

	task, err := h.store.Get(ctx, id)
	if err == model.ErrTaskNotFound {
		h.writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		h.logger.Error("failed to get task", "error", err, "id", id)
		h.writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}

	h.writeJSON(w, http.StatusOK, Response{Data: task})
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req model.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := req.Validate(); err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	task, err := h.store.Create(ctx, &req)
	if err != nil {
		h.logger.Error("failed to create task", "error", err)
		h.writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	h.logger.Info("task created", "id", task.ID, "title", task.Title)
	h.writeJSON(w, http.StatusCreated, Response{Data: task})
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "task id is required")
		return
	}

	var req model.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.store.Update(ctx, id, &req)
	if err == model.ErrTaskNotFound {
		h.writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		h.logger.Error("failed to update task", "error", err, "id", id)
		h.writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}

	h.logger.Info("task updated", "id", task.ID)
	h.writeJSON(w, http.StatusOK, Response{Data: task})
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "task id is required")
		return
	}

	err := h.store.Delete(ctx, id)
	if err == model.ErrTaskNotFound {
		h.writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		h.logger.Error("failed to delete task", "error", err, "id", id)
		h.writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}

	h.logger.Info("task deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

func (h *TaskHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, Response{Error: message})
}
