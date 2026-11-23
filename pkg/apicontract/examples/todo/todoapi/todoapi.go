// Package todoapi is the example contract for the todo service: the single
// source from which the server binding, generated client, OpenAPI document,
// gateway routes, and compatibility baseline all derive (ref #137).
package todoapi

import (
	"net/http"
	"time"

	"github.com/krewire/krewire/pkg/apicontract"
)

// Todo is one task.
type Todo struct {
	ID       int64     `json:"id"`
	Title    string    `json:"title" validate:"required,max=200"`
	Done     bool      `json:"done"`
	Priority string    `json:"priority" validate:"oneof=low normal high"`
	Created  time.Time `json:"created"`
}

// TodoList is a page of todos.
type TodoList struct {
	Items []Todo `json:"items"`
}

// ListTodosRequest filters the list.
type ListTodosRequest struct {
	Done  *bool `json:"-" query:"done" doc:"Only todos with this completion state."`
	Limit int   `json:"-" query:"limit" validate:"omitempty,min=1,max=100"`
}

// TodoID addresses one todo.
type TodoID struct {
	ID int64 `json:"-" path:"id" validate:"min=1"`
}

// CreateTodoRequest creates a todo.
type CreateTodoRequest struct {
	Title    string `json:"title" validate:"required,min=1,max=200"`
	Priority string `json:"priority,omitempty" validate:"omitempty,oneof=low normal high"`
}

// UpdateTodoRequest patches a todo.
type UpdateTodoRequest struct {
	ID    int64   `json:"-" path:"id" validate:"min=1"`
	Title *string `json:"title,omitempty" validate:"omitempty,min=1,max=200"`
	Done  *bool   `json:"done,omitempty"`
}

// ErrTodoNotFound is returned for an unknown id.
var ErrTodoNotFound = apicontract.ErrorSpec{Status: http.StatusNotFound, Code: "todo_not_found", Description: "No todo has that id."}

// Endpoint definitions. Each variable is named after its endpoint, the
// convention apicontract/clientgen relies on.
var (
	ListTodos = apicontract.Define[ListTodosRequest, TodoList]("ListTodos", http.MethodGet, "/todos",
		apicontract.Summary("List todos."), apicontract.Tags("todos"))
	GetTodo = apicontract.Define[TodoID, Todo]("GetTodo", http.MethodGet, "/todos/{id}",
		apicontract.Summary("Fetch one todo."), apicontract.Tags("todos"), apicontract.Errors(ErrTodoNotFound))
	CreateTodo = apicontract.Define[CreateTodoRequest, Todo]("CreateTodo", http.MethodPost, "/todos",
		apicontract.Summary("Create a todo."), apicontract.Tags("todos"),
		apicontract.Status(http.StatusCreated), apicontract.RequireAuth("bearer"))
	UpdateTodo = apicontract.Define[UpdateTodoRequest, Todo]("UpdateTodo", http.MethodPatch, "/todos/{id}",
		apicontract.Summary("Update a todo."), apicontract.Tags("todos"),
		apicontract.RequireAuth("bearer"), apicontract.Errors(ErrTodoNotFound))
	DeleteTodo = apicontract.Define[TodoID, apicontract.Empty]("DeleteTodo", http.MethodDelete, "/todos/{id}",
		apicontract.Summary("Delete a todo."), apicontract.Tags("todos"),
		apicontract.RequireAuth("bearer", "admin"), apicontract.Errors(ErrTodoNotFound))
)

// Version is the contract version. Bump it as the compatibility report
// requires; contract tests fail until the decision is recorded.
const Version = "1.0.0"

// Service returns the todo contract.
func Service() *apicontract.Service {
	s := apicontract.NewService("todo", Version)
	s.Title = "Todo API"
	s.Description = "Example Krewire service contract."
	s.BasePath = "/api/todo"
	return s.Add(ListTodos, GetTodo, CreateTodo, UpdateTodo, DeleteTodo)
}
