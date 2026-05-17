package common

import (
	"context"
	"errors"

	"github.com/marcosalvi-01/gowatch/internal/account"
)

type ContextKey string

const UserKey ContextKey = "user"

// GetUser extracts userID from context
func GetUser(ctx context.Context) (*account.User, error) {
	userID, ok := ctx.Value(UserKey).(*account.User)
	if !ok {
		return nil, errors.New("user not found in context")
	}
	return userID, nil
}
