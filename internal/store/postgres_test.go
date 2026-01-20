//go:build integration

package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/arth/taskmanager/internal/model"
)

func getTestDatabaseURL() string {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/taskmanager_test?sslmode=disable"
	}
	return url
}

func setupTestStore(t *testing.T) *PostgresStore {
	store, err := NewPostgresStore(getTestDatabaseURL())
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Clean up table before tests
	_, err = store.db.Exec("DELETE FROM tasks")
	if err != nil {
		t.Fatalf("failed to clean up tasks: %v", err)
	}

	return store
}

func TestPostgresStore_Create(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &model.CreateTaskRequest{
		Title:       "Test Task",
		Description: "Test Description",
	}

	task, err := store.Create(ctx, req)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	if task.ID == "" {
		t.Error("expected task ID to be set")
	}
	if task.Title != req.Title {
		t.Errorf("expected title %s, got %s", req.Title, task.Title)
	}
	if task.Description != req.Description {
		t.Errorf("expected description %s, got %s", req.Description, task.Description)
	}
	if task.Completed {
		t.Error("expected task to not be completed")
	}
}

func TestPostgresStore_Get(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a task first
	req := &model.CreateTaskRequest{
		Title:       "Test Task",
		Description: "Test Description",
	}
	created, err := store.Create(ctx, req)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Get the task
	task, err := store.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}

	if task.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, task.ID)
	}
	if task.Title != created.Title {
		t.Errorf("expected title %s, got %s", created.Title, task.Title)
	}
}

func TestPostgresStore_GetNotFound(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := store.Get(ctx, "nonexistent-id")
	if err != model.ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestPostgresStore_List(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create some tasks
	for i := 0; i < 3; i++ {
		req := &model.CreateTaskRequest{
			Title:       "Test Task",
			Description: "Test Description",
		}
		_, err := store.Create(ctx, req)
		if err != nil {
			t.Fatalf("failed to create task: %v", err)
		}
	}

	// List tasks
	tasks, err := store.List(ctx)
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
}

func TestPostgresStore_Update(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a task first
	req := &model.CreateTaskRequest{
		Title:       "Original Title",
		Description: "Original Description",
	}
	created, err := store.Create(ctx, req)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Update the task
	completed := true
	updateReq := &model.UpdateTaskRequest{
		Title:     "Updated Title",
		Completed: &completed,
	}
	updated, err := store.Update(ctx, created.ID, updateReq)
	if err != nil {
		t.Fatalf("failed to update task: %v", err)
	}

	if updated.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %s", updated.Title)
	}
	if !updated.Completed {
		t.Error("expected task to be completed")
	}
}

func TestPostgresStore_Delete(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a task first
	req := &model.CreateTaskRequest{
		Title:       "Test Task",
		Description: "Test Description",
	}
	created, err := store.Create(ctx, req)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	// Delete the task
	err = store.Delete(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}

	// Verify task is deleted
	_, err = store.Get(ctx, created.ID)
	if err != model.ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestPostgresStore_DeleteNotFound(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := store.Delete(ctx, "nonexistent-id")
	if err != model.ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestPostgresStore_Health(t *testing.T) {
	store := setupTestStore(t)
	defer store.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := store.Health(ctx)
	if err != nil {
		t.Errorf("expected health check to pass, got error: %v", err)
	}
}
