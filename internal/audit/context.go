package audit

import (
	"context"
	"strconv"

	"gorm.io/gorm"
)

type ctxKey string

const userIDKey ctxKey = "audit:user_id"

// WithUserID adds user ID to context for audit logging
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext extracts user ID from context
func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}

// SetUserID sets the audit.user_id session variable in the transaction
func SetUserID(tx *gorm.DB, ctx context.Context) error {
	if userID, ok := UserIDFromContext(ctx); ok && userID != 0 {
		return tx.Exec("SELECT set_config('audit.user_id', ?, true)", strconv.FormatInt(userID, 10)).Error
	}
	return nil
}
