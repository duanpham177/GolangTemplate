package repository

import (
	"sync"

	"servergo/internal/model"
)

// UserRepository is the data-access contract. Swap the implementation
// (Postgres, MySQL, ...) without touching service/handler.
type UserRepository interface {
	Create(name string) model.User
	List() []model.User
}

type userMemory struct {
	mu     sync.RWMutex
	nextID int
	users  []model.User
}

func NewUserMemory() UserRepository {
	return &userMemory{nextID: 1}
}

func (r *userMemory) Create(name string) model.User {
	r.mu.Lock()
	defer r.mu.Unlock()
	u := model.User{ID: r.nextID, Name: name}
	r.nextID++
	r.users = append(r.users, u)
	return u
}

func (r *userMemory) List() []model.User {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]model.User, len(r.users))
	copy(out, r.users)
	return out
}
