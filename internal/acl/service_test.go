package acl

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestCanAccessOwnedResourceNilService(t *testing.T) {
	var s *Service
	owner := uuid.New()
	self := uuid.New()
	ok, err := s.CanAccessOwnedResource(context.Background(), nil, self, owner, "x", false)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false when not owner")
	}
	ok2, err := s.CanAccessOwnedResource(context.Background(), nil, self, self, "x", false)
	if err != nil || !ok2 {
		t.Fatalf("owner match: ok=%v err=%v", ok2, err)
	}
}

func TestHasPermissionNilService(t *testing.T) {
	var s *Service
	ok, err := s.HasPermission(context.Background(), nil, uuid.New(), "any", false)
	if err != nil || ok {
		t.Fatalf("expected false, got ok=%v err=%v", ok, err)
	}
}
