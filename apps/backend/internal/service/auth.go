package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenType  = "access"
	refreshTokenType = "refresh"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid token")
	ErrEmailTaken         = errors.New("email is already registered")
	ErrInvalidPassword    = errors.New("password must contain between 8 and 72 bytes")
)

type AuthConfig struct {
	Secret     string
	Issuer     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	BcryptCost int
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type tokenClaims struct {
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

type AuthService struct {
	users      repository.UserRepositoryContract
	refresh    repository.RefreshTokenStore
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	bcryptCost int
	dummyHash  []byte
	now        func() time.Time
}

func NewAuthService(
	users repository.UserRepositoryContract,
	refresh repository.RefreshTokenStore,
	cfg AuthConfig,
) (*AuthService, error) {
	if len(cfg.Secret) < 32 {
		return nil, errors.New("JWT secret must contain at least 32 characters")
	}
	if cfg.AccessTTL <= 0 || cfg.RefreshTTL <= 0 {
		return nil, errors.New("JWT token lifetimes must be positive")
	}
	if cfg.BcryptCost < bcrypt.MinCost || cfg.BcryptCost > bcrypt.MaxCost {
		return nil, errors.New("bcrypt cost is outside the supported range")
	}
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("invalid-password"), cfg.BcryptCost)
	if err != nil {
		return nil, fmt.Errorf("create password comparison hash: %w", err)
	}
	return &AuthService{
		users: users, refresh: refresh, secret: []byte(cfg.Secret), issuer: cfg.Issuer,
		accessTTL: cfg.AccessTTL, refreshTTL: cfg.RefreshTTL,
		bcryptCost: cfg.BcryptCost, dummyHash: dummyHash, now: time.Now,
	}, nil
}

func (service *AuthService) Register(
	ctx context.Context,
	email string,
	password string,
) (*model.User, *TokenPair, error) {
	if len(password) < 8 || len(password) > 72 {
		return nil, nil, ErrInvalidPassword
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), service.bcryptCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}
	user := &model.User{
		Email:        normalizeEmail(email),
		PasswordHash: string(passwordHash),
	}
	if err := service.users.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, nil, ErrEmailTaken
		}
		return nil, nil, fmt.Errorf("create user: %w", err)
	}
	tokens, err := service.issueTokenPair(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	return user, tokens, nil
}

func (service *AuthService) Login(
	ctx context.Context,
	email string,
	password string,
) (*model.User, *TokenPair, error) {
	if len(password) > 72 {
		_ = bcrypt.CompareHashAndPassword(service.dummyHash, []byte(password[:72]))
		return nil, nil, ErrInvalidCredentials
	}
	user, err := service.users.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		_ = bcrypt.CompareHashAndPassword(service.dummyHash, []byte(password))
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("get user: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, nil, ErrInvalidCredentials
	}
	tokens, err := service.issueTokenPair(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	return user, tokens, nil
}

func (service *AuthService) Refresh(ctx context.Context, rawToken string) (*TokenPair, error) {
	claims, err := service.parse(rawToken, refreshTokenType)
	if err != nil {
		return nil, ErrInvalidToken
	}
	storedUserID, err := service.refresh.Consume(ctx, claims.ID)
	if err != nil || storedUserID.String() != claims.Subject {
		return nil, ErrInvalidToken
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if _, err := service.users.GetByID(ctx, userID); err != nil {
		return nil, ErrInvalidToken
	}
	return service.issueTokenPair(ctx, userID)
}

func (service *AuthService) ValidateAccessToken(rawToken string) (uuid.UUID, error) {
	claims, err := service.parse(rawToken, accessTokenType)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	return userID, nil
}

func (service *AuthService) issueTokenPair(
	ctx context.Context,
	userID uuid.UUID,
) (*TokenPair, error) {
	now := service.now().UTC()
	accessToken, _, err := service.sign(userID, accessTokenType, now, service.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}
	refreshToken, refreshID, err := service.sign(
		userID,
		refreshTokenType,
		now,
		service.refreshTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("sign refresh token: %w", err)
	}
	if err := service.refresh.Save(ctx, refreshID, userID, service.refreshTTL); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}
	return &TokenPair{
		AccessToken: accessToken, RefreshToken: refreshToken,
		ExpiresIn: int64(service.accessTTL.Seconds()),
	}, nil
}

func (service *AuthService) sign(
	userID uuid.UUID,
	tokenType string,
	now time.Time,
	ttl time.Duration,
) (string, string, error) {
	tokenID := uuid.NewString()
	claims := tokenClaims{
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: service.issuer, Subject: userID.String(), ID: tokenID,
			IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(service.secret)
	return signed, tokenID, err
}

func (service *AuthService) parse(rawToken string, expectedType string) (*tokenClaims, error) {
	claims := &tokenClaims{}
	token, err := jwt.ParseWithClaims(
		rawToken,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}
			return service.secret, nil
		},
		jwt.WithIssuer(service.issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid || claims.TokenType != expectedType ||
		claims.Subject == "" || claims.ID == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
