package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/arth/taskmanager/internal/model"
)

// mockStore is a mock implementation of TaskStore for testing
type mockStore struct {
	tasks       map[string]*model.Task
	healthError error
}

func newMockStore() *mockStore {
	return &mockStore{
		tasks: make(map[string]*model.Task),
	}
}

func (m *mockStore) Create(ctx context.Context, req *model.CreateTaskRequest) (*model.Task, error) {
	task := &model.Task{
		ID:          "test-id-123",
		Title:       req.Title,
		Description: req.Description,
		Completed:   false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	m.tasks[task.ID] = task
	return task, nil
}

func (m *mockStore) Get(ctx context.Context, id string) (*model.Task, error) {
	task, ok := m.tasks[id]
	if !ok {
		return nil, model.ErrTaskNotFound
	}
	return task, nil
}

func (m *mockStore) List(ctx context.Context) ([]*model.Task, error) {
	tasks := make([]*model.Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (m *mockStore) Update(ctx context.Context, id string, req *model.UpdateTaskRequest) (*model.Task, error) {
	task, ok := m.tasks[id]
	if !ok {
		return nil, model.ErrTaskNotFound
	}
	if req.Title != "" {
		task.Title = req.Title
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	if req.Completed != nil {
		task.Completed = *req.Completed
	}
	task.UpdatedAt = time.Now().UTC()
	return task, nil
}

func (m *mockStore) Delete(ctx context.Context, id string) error {
	if _, ok := m.tasks[id]; !ok {
		return model.ErrTaskNotFound
	}
	delete(m.tasks, id)
	return nil
}

func (m *mockStore) Health(ctx context.Context) error {
	return m.healthError
}

func setupTestHandler() (*TaskHandler, *mockStore) {
	store := newMockStore()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := NewTaskHandler(store, logger)
	return handler, store
}

func TestHealth(t *testing.T) {
	h, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	if data["status"] != "healthy" {
		t.Errorf("expected status healthy, got %v", data["status"])
	}
}

func TestCreateTask(t *testing.T) {
	h, _ := setupTestHandler()

	body := `{"title": "Test Task", "description": "Test Description"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var resp Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	if data["title"] != "Test Task" {
		t.Errorf("expected title 'Test Task', got %v", data["title"])
	}
}

func TestCreateTaskMissingTitle(t *testing.T) {
	h, _ := setupTestHandler()

	body := `{"description": "Test Description"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestListTasks(t *testing.T) {
	h, store := setupTestHandler()

	// Add a task to the store
	store.tasks["test-id"] = &model.Task{
		ID:          "test-id",
		Title:       "Test Task",
		Description: "Test Description",
		Completed:   false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", http.NoBody)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatal("expected data to be a slice")
	}
	if len(data) != 1 {
		t.Errorf("expected 1 task, got %d", len(data))
	}
}

func TestGetTask(t *testing.T) {
	h, store := setupTestHandler()

	// Add a task to the store
	store.tasks["test-id"] = &model.Task{
		ID:          "test-id",
		Title:       "Test Task",
		Description: "Test Description",
		Completed:   false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/test-id", http.NoBody)
	req.SetPathValue("id", "test-id")
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	h, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/nonexistent", http.NoBody)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestUpdateTask(t *testing.T) {
	h, store := setupTestHandler()

	// Add a task to the store
	store.tasks["test-id"] = &model.Task{
		ID:          "test-id",
		Title:       "Test Task",
		Description: "Test Description",
		Completed:   false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	body := `{"title": "Updated Task"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/tasks/test-id", bytes.NewBufferString(body))
	req.SetPathValue("id", "test-id")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	if data["title"] != "Updated Task" {
		t.Errorf("expected title 'Updated Task', got %v", data["title"])
	}
}

func TestDeleteTask(t *testing.T) {
	h, store := setupTestHandler()

	// Add a task to the store
	store.tasks["test-id"] = &model.Task{
		ID:          "test-id",
		Title:       "Test Task",
		Description: "Test Description",
		Completed:   false,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/test-id", http.NoBody)
	req.SetPathValue("id", "test-id")
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	// Verify task was deleted
	if _, ok := store.tasks["test-id"]; ok {
		t.Error("expected task to be deleted")
	}
}

func TestDeleteTaskNotFound(t *testing.T) {
	h, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/nonexistent", http.NoBody)
	req.SetPathValue("id", "nonexistent")
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}
