package account

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/marcosalvi-01/gowatch/logging"

	"golang.org/x/crypto/bcrypt"
)

type Store interface {
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateSession(ctx context.Context, sessionID string, userID int64, expiresAt time.Time) error
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	DeleteSession(ctx context.Context, sessionID string) error
	GetUserByID(ctx context.Context, userID int64) (*User, error)
	CleanupExpiredSessions(ctx context.Context) error
	CreateUser(ctx context.Context, email, name, passwordHash string) (*User, error)
	CountUsers(ctx context.Context) (int64, error)
	AssignNilUserWatched(ctx context.Context, userID *int64) error
	AssignNilUserLists(ctx context.Context, userID *int64) error
	SetAdmin(ctx context.Context, userID int64) error
	GetAllUsersWithStats(ctx context.Context) ([]UserWithStats, error)
	DeleteUser(ctx context.Context, userID int64) error
	UpdateUserPassword(ctx context.Context, userID int64, passwordHash string) error
	UpdatePasswordResetRequired(ctx context.Context, userID int64, reset bool) error
}

type WatchlistInitializer interface {
	EnsureWatchlistExistsForUser(ctx context.Context, userID int64) error
}

type Service struct {
	store                Store
	watchlists           WatchlistInitializer
	log                  *slog.Logger
	SessionExpiry        time.Duration
	HTTPS                bool
	DefaultAdminPassword string
}

func NewService(store Store, watchlists WatchlistInitializer, sessionExpiry time.Duration, https bool, defaultAdminPassword string) *Service {
	log := logging.Get("auth service")
	log.Debug("creating new AuthService instance")
	return &Service{
		store:                store,
		watchlists:           watchlists,
		log:                  log,
		SessionExpiry:        sessionExpiry,
		HTTPS:                https,
		DefaultAdminPassword: defaultAdminPassword,
	}
}

func (a *Service) AuthenticateUser(ctx context.Context, email, password string) (*User, error) {
	user, err := a.store.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user for email %s: %w", email, err)
	}
	err = verifyPassword(user.PasswordHash, password)
	if err != nil {
		return nil, fmt.Errorf("password verification failed for user %s: %w", email, err)
	}
	return user, nil
}

func (a *Service) CreateSession(ctx context.Context, userID int64) (string, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	expiresAt := time.Now().Add(a.SessionExpiry)
	err = a.store.CreateSession(ctx, sessionID, userID, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to create session for user %d: %w", userID, err)
	}

	return sessionID, nil
}

func (a *Service) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	session, err := a.store.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve session %s: %w", sessionID, err)
	}

	return session, nil
}

func (a *Service) Logout(ctx context.Context, sessionID string) error {
	err := a.store.DeleteSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session %s: %w", sessionID, err)
	}
	return nil
}

func (a *Service) GetUserByID(ctx context.Context, id int64) (*User, error) {
	user, err := a.store.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user %d: %w", id, err)
	}
	return user, nil
}

func (a *Service) CleanupExpiredSessions(ctx context.Context) error {
	err := a.store.CleanupExpiredSessions(ctx)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}
	return nil
}

func (a *Service) CreateUser(ctx context.Context, email, name, password string) (int64, error) {
	hash, err := hashPassword(password)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password for user %s: %w", email, err)
	}

	user, err := a.store.CreateUser(ctx, email, name, hash)
	if err != nil {
		a.log.Error("failed to create user in database", "email", email, "error", err)
		return 0, fmt.Errorf("failed to create user %s: %w", email, err)
	}

	a.log.Info("user created, now creating watchlist", "userID", user.ID)

	if err := a.watchlists.EnsureWatchlistExistsForUser(ctx, user.ID); err != nil {
		a.log.Error("failed to create watchlist for new user", "userID", user.ID, "email", email, "error", err)
		return 0, fmt.Errorf("failed to initialize user account (watchlist creation failed): %w", err)
	}

	a.log.Info("successfully created user account with watchlist", "userID", user.ID, "email", email)
	return user.ID, nil
}

func (a *Service) CountUsers(ctx context.Context) (int64, error) {
	count, err := a.store.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

func (a *Service) AssignNilUserWatched(ctx context.Context, userID *int64) error {
	err := a.store.AssignNilUserWatched(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to assign nil user to watched records for user %d: %w", *userID, err)
	}
	return nil
}

func (a *Service) AssignNilUserLists(ctx context.Context, userID *int64) error {
	err := a.store.AssignNilUserLists(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to assign nil user to list records for user %d: %w", *userID, err)
	}
	return nil
}

func (a *Service) SetUserAsAdmin(ctx context.Context, userID int64) error {
	err := a.store.SetAdmin(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to set user %d as admin: %w", userID, err)
	}
	return nil
}

func (a *Service) GetAllUsersWithStats(ctx context.Context) ([]UserWithStats, error) {
	users, err := a.store.GetAllUsersWithStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all users with stats: %w", err)
	}
	return users, nil
}

func (a *Service) DeleteUser(ctx context.Context, userID int64) error {
	err := a.store.DeleteUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user %d: %w", userID, err)
	}
	return nil
}

// UpdateUserPassword updates the password of an user.
//
// The password should be passed as plain text, this function will hash it before updating the database.
func (a *Service) UpdateUserPassword(ctx context.Context, userID int64, password string) error {
	hash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password for user %d: %w", userID, err)
	}

	err = a.store.UpdateUserPassword(ctx, userID, hash)
	if err != nil {
		return fmt.Errorf("failed to update password for user %d: %w", userID, err)
	}
	return nil
}

// RequirePasswordReset resets the password of an user to the default email prefix + . + name,
// returns it and sets the reset-password flag for the user.
func (a *Service) RequirePasswordReset(ctx context.Context, userID int64) (string, error) {
	user, err := a.GetUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user for password reset: %w", err)
	}

	newPass := fmt.Sprintf("%s.%s", strings.Split(user.Email, "@")[0], user.Name)

	err = a.UpdateUserPassword(ctx, userID, newPass)
	if err != nil {
		return "", fmt.Errorf("failed to update password during reset: %w", err)
	}

	err = a.store.UpdatePasswordResetRequired(ctx, userID, true)
	if err != nil {
		return "", fmt.Errorf("failed to set password reset flag: %w", err)
	}

	return newPass, nil
}

func (a *Service) ClearPasswordResetRequired(ctx context.Context, userID int64) error {
	err := a.store.UpdatePasswordResetRequired(ctx, userID, false)
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

// hashPassword accepts max 72 bytes passwords.
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
