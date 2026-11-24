// Package todoservice implements the example todo contract on pkg/web.
package todoservice

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/krewire/krewire/apicontract"
	"github.com/krewire/krewire/apicontract/examples/todo/todoapi"
	"github.com/krewire/krewire/apicontract/httpbind"
	"github.com/krewire/krewire/auth"
	"github.com/krewire/krewire/web"
)

// Store is an in-memory todo store.
type Store struct {
	mu    sync.Mutex
	next  int64
	items map[int64]todoapi.Todo
	now   func() time.Time
}

// NewStore returns an empty store. now defaults to time.Now.
func NewStore(now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{items: map[int64]todoapi.Todo{}, now: now}
}

// Handler serves the contract under its base path. Bearer tokens signed
// with secret authenticate callers; anonymous calls reach public endpoints.
func Handler(s *Store, secret []byte, opts ...httpbind.Option) http.Handler {
	r := web.NewRouter()
	r.Use(web.Middleware(auth.JWTAuth(secret, func(o *auth.JWTOptions) { o.ContinueOnMissing = true })))
	g := r.Group(todoapi.Service().BasePath)
	httpbind.Handle(g, todoapi.ListTodos, s.List, opts...)
	httpbind.Handle(g, todoapi.GetTodo, s.Get, opts...)
	httpbind.Handle(g, todoapi.CreateTodo, s.Create, opts...)
	httpbind.Handle(g, todoapi.UpdateTodo, s.Update, opts...)
	httpbind.Handle(g, todoapi.DeleteTodo, s.Delete, opts...)
	return r
}

// List implements todoapi.ListTodos.
func (s *Store) List(_ context.Context, req *todoapi.ListTodosRequest) (*todoapi.TodoList, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := &todoapi.TodoList{Items: []todoapi.Todo{}}
	for _, t := range s.items {
		if req.Done == nil || *req.Done == t.Done {
			out.Items = append(out.Items, t)
		}
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].ID < out.Items[j].ID })
	if req.Limit > 0 && len(out.Items) > req.Limit {
		out.Items = out.Items[:req.Limit]
	}
	return out, nil
}

// Get implements todoapi.GetTodo.
func (s *Store) Get(_ context.Context, req *todoapi.TodoID) (*todoapi.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.items[req.ID]
	if !ok {
		return nil, todoapi.ErrTodoNotFound.New("no todo with that id")
	}
	return &t, nil
}

// Create implements todoapi.CreateTodo.
func (s *Store) Create(_ context.Context, req *todoapi.CreateTodoRequest) (*todoapi.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	t := todoapi.Todo{ID: s.next, Title: req.Title, Priority: req.Priority, Created: s.now().UTC()}
	if t.Priority == "" {
		t.Priority = "normal"
	}
	s.items[t.ID] = t
	return &t, nil
}

// Update implements todoapi.UpdateTodo.
func (s *Store) Update(_ context.Context, req *todoapi.UpdateTodoRequest) (*todoapi.Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.items[req.ID]
	if !ok {
		return nil, todoapi.ErrTodoNotFound.New("no todo with that id")
	}
	if req.Title != nil {
		t.Title = *req.Title
	}
	if req.Done != nil {
		t.Done = *req.Done
	}
	s.items[t.ID] = t
	return &t, nil
}

// Delete implements todoapi.DeleteTodo.
func (s *Store) Delete(_ context.Context, req *todoapi.TodoID) (*apicontract.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[req.ID]; !ok {
		return nil, todoapi.ErrTodoNotFound.New("no todo with that id")
	}
	delete(s.items, req.ID)
	return nil, nil
}
