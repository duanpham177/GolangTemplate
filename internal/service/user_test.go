package service

import (
	"testing"

	"servergo/internal/repository"
)

func TestUserService_Create(t *testing.T) {
	svc := NewUserService(repository.NewUserMemory())

	if _, err := svc.Create("  "); err != ErrInvalidName {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}

	u, err := svc.Create("An")
	if err != nil || u.ID != 1 || u.Name != "An" {
		t.Fatalf("unexpected result: %+v, %v", u, err)
	}
}
