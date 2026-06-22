package services

import (
	"context"
	"strings"
	"testing"

	"github.com/muhammedfazall/Sendr/internal/core/domain"
)

func TestAPIKeyServiceCreate(t *testing.T) {
	mock := newMockDeps()
	svc := NewAPIKeyService(mock.apikeys, mock.users)

	user := mock.addUserWithPlan("free")
	fullKey, key, err := svc.Create(context.Background(), user.ID, "my-key")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if !strings.HasPrefix(fullKey, "mk_live_") {
		t.Fatalf("expected mk_live_ prefix, got %q", fullKey)
	}

	if key.Name != "my-key" {
		t.Fatalf("expected key name 'my-key', got %q", key.Name)
	}
}

func TestAPIKeyServiceCreateExceedsPlanLimit(t *testing.T) {
	mock := newMockDeps()
	svc := NewAPIKeyService(mock.apikeys, mock.users)

	user := mock.addUserWithPlan("free") // MaxAPIKeys = 1

	// Create first key — should succeed
	_, _, err := svc.Create(context.Background(), user.ID, "key-1")
	if err != nil {
		t.Fatalf("first key creation failed: %v", err)
	}

	// Create second key — should fail (limit reached)
	_, _, err = svc.Create(context.Background(), user.ID, "key-2")
	if err == nil {
		t.Fatal("expected error when exceeding plan limit, got nil")
	}
	if !strings.Contains(err.Error(), "API key limit reached") {
		t.Fatalf("expected limit error, got %v", err)
	}
}

func TestAPIKeyServiceCreateUnlimitedPlan(t *testing.T) {
	mock := newMockDeps()
	svc := NewAPIKeyService(mock.apikeys, mock.users)

	user := mock.addUserWithPlan("max") // MaxAPIKeys = -1 (unlimited)

	for i := 0; i < 5; i++ {
		_, _, err := svc.Create(context.Background(), user.ID, "key")
		if err != nil {
			t.Fatalf("key creation %d failed for unlimited plan: %v", i, err)
		}
	}
}

func TestAPIKeyServiceList(t *testing.T) {
	mock := newMockDeps()
	svc := NewAPIKeyService(mock.apikeys, mock.users)

	user := mock.addUserWithPlan("max")

	_, _, err := svc.Create(context.Background(), user.ID, "key-a")
	if err != nil {
		t.Fatalf("create key-a: %v", err)
	}
	_, _, err = svc.Create(context.Background(), user.ID, "key-b")
	if err != nil {
		t.Fatalf("create key-b: %v", err)
	}

	keys, err := svc.List(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
}

func TestAPIKeyServiceRevoke(t *testing.T) {
	mock := newMockDeps()
	svc := NewAPIKeyService(mock.apikeys, mock.users)

	user := mock.addUserWithPlan("max")
	_, key, err := svc.Create(context.Background(), user.ID, "to-revoke")
	if err != nil {
		t.Fatalf("create key: %v", err)
	}

	if err := svc.Revoke(context.Background(), key.ID, user.ID); err != nil {
		t.Fatalf("Revoke returned error: %v", err)
	}

	// Verify it's revoked in the repo
	keys, _ := mock.apikeys.ListByUser(context.Background(), user.ID)
	if !keys[0].Revoked {
		t.Fatal("expected key to be revoked")
	}
}

func TestAPIKeyServiceRevokeWrongUser(t *testing.T) {
	mock := newMockDeps()
	svc := NewAPIKeyService(mock.apikeys, mock.users)

	user1 := mock.addUserWithPlan("max")
	user2 := &domain.User{ID: "user-different", Email: "other@test.com", Name: "Other"}
	mock.users.users[user2.ID] = user2

	_, key, err := svc.Create(context.Background(), user1.ID, "my-key")
	if err != nil {
		t.Fatalf("create key: %v", err)
	}

	err = svc.Revoke(context.Background(), key.ID, user2.ID)
	if err == nil {
		t.Fatal("expected error revoking another user's key, got nil")
	}
}

func TestAPIKeyServiceValidate(t *testing.T) {
	mock := newMockDeps()
	svc := NewAPIKeyService(mock.apikeys, mock.users)

	user := mock.addUserWithPlan("free")
	fullKey, _, err := svc.Create(context.Background(), user.ID, "test")
	if err != nil {
		t.Fatalf("create key: %v", err)
	}

	validated, err := svc.Validate(context.Background(), fullKey)
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if validated.UserID != user.ID {
		t.Fatalf("expected user %q, got %q", user.ID, validated.UserID)
	}
}

func TestAPIKeyServiceValidateInvalidFormat(t *testing.T) {
	mock := newMockDeps()
	svc := NewAPIKeyService(mock.apikeys, mock.users)

	_, err := svc.Validate(context.Background(), "invalid-key-format")
	if err == nil {
		t.Fatal("expected error for invalid key format, got nil")
	}
}

func TestAPIKeyServiceValidateRevokedKey(t *testing.T) {
	mock := newMockDeps()
	svc := NewAPIKeyService(mock.apikeys, mock.users)

	user := mock.addUserWithPlan("free")
	fullKey, key, err := svc.Create(context.Background(), user.ID, "test")
	if err != nil {
		t.Fatalf("create key: %v", err)
	}

	mock.apikeys.Revoke(context.Background(), key.ID, user.ID)

	_, err = svc.Validate(context.Background(), fullKey)
	if err == nil {
		t.Fatal("expected error for revoked key, got nil")
	}
}
