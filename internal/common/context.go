package common

import (
	"context"
	"errors"

	"gowatch/internal/models"
)

type ContextKey string

const UserKey ContextKey = "user"

// GetUser extracts userID from context
func GetUser(ctx context.Context) (*models.User, error) {
	userID, ok := ctx.Value(UserKey).(*models.User)
	if !ok {
		return nil, errors.New("user not found in context")
	}
	return userID, nil
}
