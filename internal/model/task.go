package model

import "time"

type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateTaskRequest struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Completed   *bool  `json:"completed,omitempty"`
}

func (r *CreateTaskRequest) Validate() error {
	if r.Title == "" {
		return ErrTitleRequired
	}
	return nil
}

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrTitleRequired = Error("title is required")
	ErrTaskNotFound  = Error("task not found")
)
