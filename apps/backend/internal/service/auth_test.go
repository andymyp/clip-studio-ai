package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type memoryUserRepository struct {
	user *model.User
}

func (repo *memoryUserRepository) Create(_ context.Context, user *model.User) error {
	if repo.user != nil {
		return repository.ErrConflict
	}
	user.EnsureID()
	user.CreatedAt = time.Now()
	repo.user = user
	return nil
}
func (repo *memoryUserRepository) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	if repo.user == nil || repo.user.ID != id {
		return nil, repository.ErrNotFound
	}
	return repo.user, nil
}
func (repo *memoryUserRepository) GetByEmail(_ context.Context, email string) (*model.User, error) {
	if repo.user == nil || repo.user.Email != email {
		return nil, repository.ErrNotFound
	}
	return repo.user, nil
}
func (repo *memoryUserRepository) Update(context.Context, *model.User) error { return nil }
func (repo *memoryUserRepository) Delete(context.Context, uuid.UUID) error   { return nil }

type memoryRefreshStore struct {
	tokens map[string]uuid.UUID
}

func (store *memoryRefreshStore) Save(
	_ context.Context,
	tokenID string,
	userID uuid.UUID,
	_ time.Duration,
) error {
	if store.tokens == nil {
		store.tokens = make(map[string]uuid.UUID)
	}
	store.tokens[tokenID] = userID
	return nil
}
func (store *memoryRefreshStore) Consume(
	_ context.Context,
	tokenID string,
) (uuid.UUID, error) {
	userID, exists := store.tokens[tokenID]
	if !exists {
		return uuid.Nil, repository.ErrNotFound
	}
	delete(store.tokens, tokenID)
	return userID, nil
}
func (store *memoryRefreshStore) Delete(_ context.Context, tokenID string) error {
	delete(store.tokens, tokenID)
	return nil
}

func newTestAuthService(t *testing.T) (*AuthService, *memoryUserRepository) {
	t.Helper()
	users := &memoryUserRepository{}
	auth, err := NewAuthService(users, &memoryRefreshStore{}, AuthConfig{
		Secret: "test-secret-that-is-at-least-32-characters",
		Issuer: "clipstudio-test", AccessTTL: 15 * time.Minute,
		RefreshTTL: time.Hour, BcryptCost: bcrypt.MinCost,
	})
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}
	return auth, users
}

func TestRegisterHashesPasswordAndIssuesTokens(t *testing.T) {
	auth, users := newTestAuthService(t)

	user, tokens, err := auth.Register(context.Background(), "USER@example.com", "password123")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if user.Email != "user@example.com" {
		t.Fatalf("normalized email = %q", user.Email)
	}
	if users.user.PasswordHash == "password123" {
		t.Fatal("password was stored as plaintext")
	}
	if bcrypt.CompareHashAndPassword([]byte(users.user.PasswordHash), []byte("password123")) != nil {
		t.Fatal("stored password hash does not match")
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("tokens were not issued")
	}
	if got, err := auth.ValidateAccessToken(tokens.AccessToken); err != nil || got != user.ID {
		t.Fatalf("ValidateAccessToken() = %s, %v", got, err)
	}
}

func TestRefreshTokenRotatesOnce(t *testing.T) {
	auth, _ := newTestAuthService(t)
	_, tokens, err := auth.Register(context.Background(), "user@example.com", "password123")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	rotated, err := auth.Refresh(context.Background(), tokens.RefreshToken)
	if err != nil || rotated.RefreshToken == tokens.RefreshToken {
		t.Fatalf("Refresh() = %#v, %v", rotated, err)
	}
	if _, err := auth.Refresh(context.Background(), tokens.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("replayed Refresh() error = %v, want ErrInvalidToken", err)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	auth, _ := newTestAuthService(t)
	_, _, _ = auth.Register(context.Background(), "user@example.com", "password123")

	if _, _, err := auth.Login(
		context.Background(),
		"user@example.com",
		"not-the-password",
	); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}
