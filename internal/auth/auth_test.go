package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("securepassword123")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}

	if !VerifyPassword(hash, "securepassword123") {
		t.Error("expected password verification to succeed")
	}

	if VerifyPassword(hash, "wrongpassword") {
		t.Error("expected password verification to fail for wrong password")
	}
}

func TestPasswordTooShort(t *testing.T) {
	_, err := HashPassword("short")
	if err == nil {
		t.Error("expected error for short password")
	}
}

func TestJWTGenerateAndValidate(t *testing.T) {
	mgr, err := NewJWTManager("this-is-a-very-secret-key-at-least-32-chars", 24*time.Hour)
	if err != nil {
		t.Fatalf("create JWT manager: %v", err)
	}

	token, expiresAt, err := mgr.GenerateToken(1, "admin", "super_admin")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	if token == "" {
		t.Error("expected non-empty token")
	}
	if expiresAt.Before(time.Now()) {
		t.Error("expected future expiry")
	}

	claims, err := mgr.ValidateToken(token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}

	if claims.UserID != 1 {
		t.Errorf("expected userID 1, got %d", claims.UserID)
	}
	if claims.Username != "admin" {
		t.Errorf("expected username admin, got %s", claims.Username)
	}
	if claims.Role != "super_admin" {
		t.Errorf("expected role super_admin, got %s", claims.Role)
	}
}

func TestJWTInvalidToken(t *testing.T) {
	mgr, _ := NewJWTManager("this-is-a-very-secret-key-at-least-32-chars", 24*time.Hour)
	_, err := mgr.ValidateToken("invalid.token.here")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestJWTSecretTooShort(t *testing.T) {
	_, err := NewJWTManager("short", 24*time.Hour)
	if err == nil {
		t.Error("expected error for short secret")
	}
}

func TestAuthMiddleware(t *testing.T) {
	mgr, _ := NewJWTManager("this-is-a-very-secret-key-at-least-32-chars", 24*time.Hour)
	token, _, _ := mgr.GenerateToken(1, "admin", "super_admin")

	handler := Middleware(mgr)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r.Context())
		if claims == nil {
			t.Error("expected claims in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	// With valid token
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with valid token, got %d", w.Code)
	}

	// Without token
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", w2.Code)
	}
}

func TestRequireRole(t *testing.T) {
	mgr, _ := NewJWTManager("this-is-a-very-secret-key-at-least-32-chars", 24*time.Hour)

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := Middleware(mgr)(RequireRole(RoleRegistryAdmin)(innerHandler))

	// Admin should pass
	adminToken, _, _ := mgr.GenerateToken(1, "admin", "super_admin")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for super_admin, got %d", w.Code)
	}

	// Consumer should be forbidden
	consumerToken, _, _ := mgr.GenerateToken(2, "user", "consumer")
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Authorization", "Bearer "+consumerToken)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403 for consumer, got %d", w2.Code)
	}
}

func TestRoleHierarchy(t *testing.T) {
	tests := []struct {
		userRole     Role
		requiredRole Role
		expected     bool
	}{
		{RoleSuperAdmin, RoleSuperAdmin, true},
		{RoleSuperAdmin, RoleViewer, true},
		{RoleConsumer, RolePublisher, false},
		{RoleViewer, RoleConsumer, false},
		{RolePublisher, RolePublisher, true},
	}

	for _, tt := range tests {
		result := HasPermission(tt.userRole, tt.requiredRole)
		if result != tt.expected {
			t.Errorf("HasPermission(%s, %s) = %v, want %v", tt.userRole, tt.requiredRole, result, tt.expected)
		}
	}
}

