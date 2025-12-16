package audit

import (
	"context"
	"strconv"

	"gorm.io/gorm"
)

// Auth method constants
const (
	AuthSession = "session" // Web UI session/cookie
	AuthAPIKey  = "api_key" // API key authentication
	AuthBasic   = "basic"   // HTTP Basic auth (superuser)
)

type ctxKey string

const (
	userIDKey     ctxKey = "audit:user_id"
	authMethodKey ctxKey = "audit:auth_method"
	apiKeyIDKey   ctxKey = "audit:api_key_id"
)

// WithUserID adds user ID to context for audit logging
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext extracts user ID from context
func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}

// WithAuthMethod adds auth method to context
func WithAuthMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, authMethodKey, method)
}

// AuthMethodFromContext extracts auth method from context
func AuthMethodFromContext(ctx context.Context) string {
	method, _ := ctx.Value(authMethodKey).(string)
	return method
}

// WithAPIKeyID adds API key ID to context
func WithAPIKeyID(ctx context.Context, keyID string) context.Context {
	return context.WithValue(ctx, apiKeyIDKey, keyID)
}

// APIKeyIDFromContext extracts API key ID from context
func APIKeyIDFromContext(ctx context.Context) string {
	keyID, _ := ctx.Value(apiKeyIDKey).(string)
	return keyID
}

// SetContext sets all audit session variables in the transaction
func SetContext(tx *gorm.DB, ctx context.Context) error {
	if userID, ok := UserIDFromContext(ctx); ok && userID != 0 {
		if err := tx.Exec("SELECT set_config('audit.user_id', ?, true)", strconv.FormatInt(userID, 10)).Error; err != nil {
			return err
		}
	}
	if method := AuthMethodFromContext(ctx); method != "" {
		if err := tx.Exec("SELECT set_config('audit.auth_method', ?, true)", method).Error; err != nil {
			return err
		}
	}
	if keyID := APIKeyIDFromContext(ctx); keyID != "" {
		if err := tx.Exec("SELECT set_config('audit.api_key_id', ?, true)", keyID).Error; err != nil {
			return err
		}
	}
	return nil
}
