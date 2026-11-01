package service

import (
	"errors"
	"strings"

	"servergo/internal/model"
	"servergo/internal/repository"
)

var ErrInvalidName = errors.New("name is required")

// UserService holds business logic.
type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Create(name string) (model.User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return model.User{}, ErrInvalidName
	}
	return s.repo.Create(name), nil
}

func (s *UserService) List() []model.User {
	return s.repo.List()
}
