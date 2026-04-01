package auth

import (
	"net/http"
)

// Role represents a user role in the RBAC system.
type Role string

const (
	RoleSuperAdmin    Role = "super_admin"
	RoleRegistryAdmin Role = "registry_admin"
	RolePublisher     Role = "publisher"
	RoleConsumer      Role = "consumer"
	RoleViewer        Role = "viewer"
)

// roleHierarchy defines the permission level for each role.
// Higher number = more permissions.
var roleHierarchy = map[Role]int{
	RoleSuperAdmin:    100,
	RoleRegistryAdmin: 80,
	RolePublisher:     60,
	RoleConsumer:      40,
	RoleViewer:        20,
}

// IsValidRole checks if a string is one of the defined roles.
func IsValidRole(role string) bool {
	_, ok := roleHierarchy[Role(role)]
	return ok
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

