package auth

import (
	"net/http"
	"path"
)

// Role represents a user role in the RBAC system.
type Role string

const (
	RoleSuperAdmin   Role = "super_admin"
	RoleRegistryAdmin Role = "registry_admin"
	RolePublisher    Role = "publisher"
	RoleConsumer     Role = "consumer"
	RoleViewer       Role = "viewer"
)

// roleHierarchy defines the permission level for each role.
// Higher number = more permissions.
var roleHierarchy = map[Role]int{
	RoleSuperAdmin:   100,
	RoleRegistryAdmin: 80,
	RolePublisher:    60,
	RoleConsumer:     40,
	RoleViewer:       20,
}

// HasPermission checks if a role has at least the required permission level.
func HasPermission(userRole Role, requiredRole Role) bool {
	return roleHierarchy[userRole] >= roleHierarchy[requiredRole]
}

// RequireRole returns a middleware that enforces a minimum role.
func RequireRole(minRole Role) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				http.Error(w, `{"error":"authorization required"}`, http.StatusUnauthorized)
				return
			}

			if !HasPermission(Role(claims.Role), minRole) {
				http.Error(w, `{"error":"insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// MatchNamespace checks if a pattern matches a package name.
// Supports glob patterns like "@org/*", "express", "@types/*".
func MatchNamespace(pattern, name string) bool {
	if pattern == "*" {
		return true
	}
	matched, err := path.Match(pattern, name)
	if err != nil {
		return false
	}
	return matched
}
