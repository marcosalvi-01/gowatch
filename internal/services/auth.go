package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gowatch/db"
	"gowatch/internal/common"
	"gowatch/internal/models"
	"gowatch/logging"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db                   db.DB
	listService          *ListService
	log                  *slog.Logger
	SessionExpiry        time.Duration
	HTTPS                bool
	DefaultAdminPassword string
}

func NewAuthService(db db.DB, listService *ListService, sessionExpiry time.Duration, https bool, defaultAdminPassword string) *AuthService {
	log := logging.Get("auth service")
	log.Debug("creating new AuthService instance")
	return &AuthService{
		db:                   db,
		listService:          listService,
		log:                  log,
		SessionExpiry:        sessionExpiry,
		HTTPS:                https,
		DefaultAdminPassword: defaultAdminPassword,
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

	user, err := a.db.CreateUser(ctx, email, name, hash)
	if err != nil {
		a.log.Error("failed to create user in database", "email", email, "error", err)
		return 0, fmt.Errorf("failed to create user %s: %w", email, err)
	}

	ctx = context.WithValue(ctx, common.UserKey, user)

	a.log.Info("user created, now creating watchlist", "userID", user.ID)

	// CRITICAL: Ensure watchlist exists. Failure prevents account creation
	err = a.listService.EnsureWatchlistExists(ctx)
	if err != nil {
		a.log.Error("failed to create watchlist for new user", "userID", user.ID, "email", email, "error", err)
		return 0, fmt.Errorf("failed to initialize user account (watchlist creation failed): %w", err)
	}

	a.log.Info("successfully created user account with watchlist", "userID", user.ID, "email", email)
	return user.ID, nil
}

func (a *AuthService) CountUsers(ctx context.Context) (int64, error) {
	count, err := a.db.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

func (a *AuthService) AssignNilUserWatched(ctx context.Context, userID *int64) error {
	err := a.db.AssignNilUserWatched(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to assign nil user to watched records for user %d: %w", *userID, err)
	}
	return nil
}

func (a *AuthService) AssignNilUserLists(ctx context.Context, userID *int64) error {
	err := a.db.AssignNilUserLists(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to assign nil user to list records for user %d: %w", *userID, err)
	}
	return nil
}

func (a *AuthService) SetUserAsAdmin(ctx context.Context, userID int64) error {
	err := a.db.SetAdmin(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to set user %d as admin: %w", userID, err)
	}
	return nil
}

func (a *AuthService) GetAllUsersWithStats(ctx context.Context) ([]models.UserWithStats, error) {
	users, err := a.db.GetAllUsersWithStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users with stats: %w", err)
	}
	return users, nil
}

func (a *AuthService) DeleteUser(ctx context.Context, userID int64) error {
	err := a.db.DeleteUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user %d: %w", userID, err)
	}
	return nil
}

// UpdateUserPassword updates the password of an user.
//
// The password should be passed as plain text, this function will hash it before updating the database
func (a *AuthService) UpdateUserPassword(ctx context.Context, userID int64, password string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password for user %d: %w", userID, err)
	}

	err = a.db.UpdateUserPassword(ctx, userID, hash)
	if err != nil {
		return fmt.Errorf("failed to update password for user %d: %w", userID, err)
	}
	return nil
}

// RequirePasswordReset resets the password of an user to the default email prefix + . + name, returns it and set the flag to reset the password for the user
func (a *AuthService) RequirePasswordReset(ctx context.Context, userID int64) (string, error) {
	user, err := a.GetUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user for password reset: %w", err)
	}

	// email prefix + . + name (john@example.com + doe = john.doe)
	newPass := fmt.Sprintf("%s.%s", strings.Split(user.Email, "@")[0], user.Name)

	err = a.UpdateUserPassword(ctx, userID, newPass)
	if err != nil {
		return "", fmt.Errorf("failed to update password during reset: %w", err)
	}

	err = a.db.UpdatePasswordResetRequired(ctx, userID, true)
	if err != nil {
		return "", fmt.Errorf("failed to set password reset flag: %w", err)
	}

	return newPass, nil
}

func (a *AuthService) ClearPasswordResetRequired(ctx context.Context, userID int64) error {
	err := a.db.UpdatePasswordResetRequired(ctx, userID, false)
	if err != nil {
		return fmt.Errorf("failed to clear password reset flag: %w", err)
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
