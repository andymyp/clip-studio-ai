package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/model"
	"github.com/clipstudio-ai/clipstudio-ai/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AuthHandler struct {
	auth     *service.AuthService
	validate *validator.Validate
	logger   *zap.Logger
}

type credentialsRequest struct {
	Email    string `json:"email" validate:"required,email,max=320"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type authUserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

type authResponse struct {
	User   authUserResponse `json:"user"`
	Tokens tokenResponse    `json:"tokens"`
}

func NewAuthHandler(auth *service.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, validate: validator.New(), logger: logger}
}

func (handler *AuthHandler) Register(c *gin.Context) {
	var request credentialsRequest
	if !handler.bindAndValidate(c, &request) {
		return
	}
	user, tokens, err := handler.auth.Register(
		c.Request.Context(),
		request.Email,
		request.Password,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalidPassword) {
			c.JSON(http.StatusBadRequest, errorResponse{
				Error: "password must contain between 8 and 72 bytes",
			})
			return
		}
		if errors.Is(err, service.ErrEmailTaken) {
			c.JSON(http.StatusConflict, errorResponse{Error: "email is already registered"})
			return
		}
		writeError(c, handler.logger, err)
		return
	}
	c.JSON(http.StatusCreated, newAuthResponse(user, tokens))
}

func (handler *AuthHandler) Login(c *gin.Context) {
	var request credentialsRequest
	if !handler.bindAndValidate(c, &request) {
		return
	}
	user, tokens, err := handler.auth.Login(
		c.Request.Context(),
		request.Email,
		request.Password,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, errorResponse{Error: "invalid email or password"})
			return
		}
		writeError(c, handler.logger, err)
		return
	}
	c.JSON(http.StatusOK, newAuthResponse(user, tokens))
}

func (handler *AuthHandler) Refresh(c *gin.Context) {
	var request refreshRequest
	if !handler.bindAndValidate(c, &request) {
		return
	}
	tokens, err := handler.auth.Refresh(c.Request.Context(), request.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			c.JSON(http.StatusUnauthorized, errorResponse{Error: "invalid refresh token"})
			return
		}
		writeError(c, handler.logger, err)
		return
	}
	c.JSON(http.StatusOK, newTokenResponse(tokens))
}

func (handler *AuthHandler) bindAndValidate(c *gin.Context, destination any) bool {
	decoder := json.NewDecoder(io.LimitReader(c.Request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid request body"})
		return false
	}
	if err := handler.validate.Struct(destination); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid email or password format"})
		return false
	}
	return true
}

func newAuthResponse(user *model.User, tokens *service.TokenPair) authResponse {
	return authResponse{
		User:   authUserResponse{ID: user.ID, Email: user.Email, CreatedAt: user.CreatedAt},
		Tokens: newTokenResponse(tokens),
	}
}

func newTokenResponse(tokens *service.TokenPair) tokenResponse {
	return tokenResponse{
		AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken,
		TokenType: "Bearer", ExpiresIn: tokens.ExpiresIn,
	}
}
