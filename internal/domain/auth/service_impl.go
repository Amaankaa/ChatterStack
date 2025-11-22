package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"chatterstack/internal/domain/models"
	redisadapter "chatterstack/internal/repository/redis"
	"chatterstack/internal/utils"
)

var (
	ErrEmailAlreadyUsed    = errors.New("auth: email already in use")
	ErrInvalidCredentials  = errors.New("auth: invalid email or password")
	ErrInvalidRefreshToken = errors.New("auth: invalid refresh token")
	ErrInvalidAccessToken  = errors.New("auth: invalid access token")
	ErrMissingUsername     = errors.New("auth: username is required")
	ErrMissingEmail        = errors.New("auth: email is required")
	ErrMissingPassword     = errors.New("auth: password is required")
)

// userRepository outlines the persistence operations needed by the domain service.
type userRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	UpdateStatus(ctx context.Context, id string, status models.UserStatus, lastSeen time.Time) error
}

// tokenCache abstracts the cache dependency used for session tracking.
type tokenCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type service struct {
	users      userRepository
	cache      tokenCache
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewService constructs an auth service with the provided dependencies.
func NewService(users userRepository, cache tokenCache, accessTTL, refreshTTL time.Duration) Service {
	return &service{users: users, cache: cache, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func (s *service) Register(ctx context.Context, input RegisterInput) (*models.User, error) {
	if err := validateRegisterInput(input); err != nil {
		return nil, err
	}

	email := normalizeEmail(input.Email)
	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return nil, ErrEmailAlreadyUsed
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	hash, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username:     strings.TrimSpace(input.Username),
		Email:        email,
		PasswordHash: hash,
		Status:       models.UserStatusOffline,
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	user.PasswordHash = ""
	return user, nil
}

func (s *service) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	email = normalizeEmail(email)
	if email == "" {
		return nil, ErrMissingEmail
	}
	if password == "" {
		return nil, ErrMissingPassword
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := utils.CompareHashAndPassword(user.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	tokens, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	if err := s.users.UpdateStatus(ctx, user.ID, models.UserStatusOnline, time.Now().UTC()); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, ErrInvalidRefreshToken
	}

	userID, err := s.cache.Get(ctx, refreshKey(refreshToken))
	if err != nil {
		if errors.Is(err, redisadapter.ErrCacheMiss) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	activeToken, err := s.cache.Get(ctx, sessionKey(userID))
	if err != nil {
		if errors.Is(err, redisadapter.ErrCacheMiss) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	if activeToken != refreshToken {
		return nil, ErrInvalidRefreshToken
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	tokens, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, err
	}

	if err := s.users.UpdateStatus(ctx, user.ID, models.UserStatusOnline, time.Now().UTC()); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *service) Logout(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("auth: user id is required")
	}

	if err := s.invalidateSession(ctx, userID); err != nil {
		return err
	}

	return s.users.UpdateStatus(ctx, userID, models.UserStatusOffline, time.Now().UTC())
}

func (s *service) issueTokens(ctx context.Context, user *models.User) (*TokenPair, error) {
	if err := s.invalidateSession(ctx, user.ID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	accessClaims := map[string]interface{}{
		"sub":   user.ID,
		"email": user.Email,
		"iat":   now.Unix(),
		"exp":   now.Add(s.accessTTL).Unix(),
	}

	refreshClaims := map[string]interface{}{
		"sub": user.ID,
		"iat": now.Unix(),
		"exp": now.Add(s.refreshTTL).Unix(),
	}

	accessToken, err := utils.GenerateAccessToken(accessClaims)
	if err != nil {
		return nil, err
	}

	refreshToken, err := utils.GenerateRefreshToken(refreshClaims)
	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, sessionKey(user.ID), refreshToken, s.refreshTTL); err != nil {
		return nil, err
	}
	if err := s.cache.Set(ctx, refreshKey(refreshToken), user.ID, s.refreshTTL); err != nil {
		_ = s.cache.Delete(ctx, sessionKey(user.ID))
		return nil, err
	}
	if err := s.cache.Set(ctx, accessKey(accessToken), user.ID, s.accessTTL); err != nil {
		_ = s.cache.Delete(ctx, sessionKey(user.ID))
		_ = s.cache.Delete(ctx, refreshKey(refreshToken))
		_ = s.cache.Delete(ctx, accessKey(accessToken))
		return nil, err
	}
	if err := s.cache.Set(ctx, accessSessionKey(user.ID), accessToken, s.accessTTL); err != nil {
		_ = s.cache.Delete(ctx, sessionKey(user.ID))
		_ = s.cache.Delete(ctx, refreshKey(refreshToken))
		_ = s.cache.Delete(ctx, accessKey(accessToken))
		_ = s.cache.Delete(ctx, accessSessionKey(user.ID))
		return nil, err
	}

	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *service) ValidateAccessToken(ctx context.Context, accessToken string) (string, error) {
	token := strings.TrimSpace(accessToken)
	if token == "" {
		return "", ErrInvalidAccessToken
	}

	userID, err := s.cache.Get(ctx, accessKey(token))
	if err != nil {
		if errors.Is(err, redisadapter.ErrCacheMiss) {
			return "", ErrInvalidAccessToken
		}
		return "", err
	}

	if err := s.cache.Set(ctx, accessKey(token), userID, s.accessTTL); err != nil {
		return "", err
	}
	if err := s.cache.Set(ctx, accessSessionKey(userID), token, s.accessTTL); err != nil {
		return "", err
	}

	return userID, nil
}

func (s *service) invalidateSession(ctx context.Context, userID string) error {
	token, err := s.cache.Get(ctx, sessionKey(userID))
	if err != nil && !errors.Is(err, redisadapter.ErrCacheMiss) {
		return err
	}

	if err == nil {
		_ = s.cache.Delete(ctx, sessionKey(userID))
		_ = s.cache.Delete(ctx, refreshKey(token))
	}

	accessToken, err := s.cache.Get(ctx, accessSessionKey(userID))
	if err != nil && !errors.Is(err, redisadapter.ErrCacheMiss) {
		return err
	}

	if err == nil {
		_ = s.cache.Delete(ctx, accessSessionKey(userID))
		_ = s.cache.Delete(ctx, accessKey(accessToken))
	}

	return nil
}

func sessionKey(userID string) string {
	return fmt.Sprintf("auth:session:%s", userID)
}

func accessKey(token string) string {
	return fmt.Sprintf("auth:access:%s", token)
}

func accessSessionKey(userID string) string {
	return fmt.Sprintf("auth:access-session:%s", userID)
}

func refreshKey(token string) string {
	return fmt.Sprintf("auth:refresh:%s", token)
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func validateRegisterInput(input RegisterInput) error {
	if strings.TrimSpace(input.Username) == "" {
		return ErrMissingUsername
	}
	if strings.TrimSpace(input.Email) == "" {
		return ErrMissingEmail
	}
	if input.Password == "" {
		return ErrMissingPassword
	}
	return nil
}
