package audit

import "context"

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

