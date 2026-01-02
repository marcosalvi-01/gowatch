package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"gowatch/db"
	"gowatch/internal/models"
	"gowatch/logging"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db            db.DB
	log           *slog.Logger
	SessionExpiry time.Duration
	HTTPS         bool
}

func NewAuthService(db db.DB, sessionExpiry time.Duration, https bool) *AuthService {
	log := logging.Get("auth service")
	log.Debug("creating new AuthService instance")
	return &AuthService{
		db:            db,
		log:           log,
		SessionExpiry: sessionExpiry,
		HTTPS:         https,
	}
}

func (a *AuthService) AuthenticateUser(ctx context.Context, email, password string) (*models.User, error) {
	user, err := a.db.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user for email %s: %w", email, err)
	}
	err = verifyPassword(user.PasswordHash, password)
	if err != nil {
		return nil, fmt.Errorf("password verification failed for user %s: %w", email, err)
	}
	return user, nil
}

func (a *AuthService) CreateSession(ctx context.Context, userID int64) (string, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	expiresAt := time.Now().Add(a.SessionExpiry)
	err = a.db.CreateSession(ctx, sessionID, userID, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to create session for user %d: %w", userID, err)
	}

	return sessionID, nil
}

func (a *AuthService) GetSession(ctx context.Context, sessionID string) (*models.Session, error) {
	session, err := a.db.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session %s: %w", sessionID, err)
	}

	return session, nil
}

func (a *AuthService) Logout(ctx context.Context, sessionID string) error {
	err := a.db.DeleteSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session %s: %w", sessionID, err)
	}
	return nil
}

func (a *AuthService) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	user, err := a.db.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user %d: %w", id, err)
	}
	return user, nil
}

func (a *AuthService) CleanupExpiredSessions(ctx context.Context) error {
	err := a.db.CleanupExpiredSessions(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}
	return nil
}

func (a *AuthService) CreateUser(ctx context.Context, email, name, password string) (int64, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password for user %s: %w", email, err)
	}

	userID, err := a.db.CreateUser(ctx, email, name, hash)
	if err != nil {
		return 0, fmt.Errorf("failed to create user %s: %w", email, err)
	}

	return userID, nil
}

func (a *AuthService) CountUsers(ctx context.Context) (int64, error) {
	count, err := a.db.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("TODO: %w", err)
	}

	return count, nil
}

func (a *AuthService) AssignNilUserWatched(ctx context.Context, userID *int64) error {
	err := a.db.AssignNilUserWatched(ctx, userID)
	if err != nil {
		return fmt.Errorf("TODO: %w", err)
	}
	return nil
}

func (a *AuthService) AssignNilUserLists(ctx context.Context, userID *int64) error {
	err := a.db.AssignNilUserLists(ctx, userID)
	if err != nil {
		return fmt.Errorf("TODO: %w", err)
	}
	return nil
}

func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashPassword accepts max 72 bytes passwords
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to generate password hash: %w", err)
	}
	return string(hash), nil
}

func verifyPassword(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return fmt.Errorf("password verification failed: %w", err)
	}
	return nil
}
