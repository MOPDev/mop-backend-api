package middleware

import (
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v4"
)

func TestTokenErrorClassification(t *testing.T) {
	expired := jwt.NewValidationError("expired", jwt.ValidationErrorExpired)
	if !errors.Is(expired, jwt.ErrTokenExpired) {
		t.Fatal("expected expired sentinel to match")
	}
}
