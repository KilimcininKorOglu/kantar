package registry

import "errors"

var (
	// ErrPackageNotFound is returned when a requested package does not exist.
	ErrPackageNotFound = errors.New("package not found")

	// ErrVersionNotFound is returned when a requested version does not exist.
	ErrVersionNotFound = errors.New("version not found")

	// ErrAlreadyExists is returned when attempting to publish a package that already exists.
	ErrAlreadyExists = errors.New("package version already exists")

	// ErrNotConfigured is returned when a plugin is used before being configured.
	ErrNotConfigured = errors.New("plugin not configured")

	// ErrUpstreamUnavailable is returned when the upstream registry cannot be reached.
	ErrUpstreamUnavailable = errors.New("upstream registry unavailable")

	// ErrUnauthorized is returned when authentication fails.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when the user lacks permission.
	ErrForbidden = errors.New("forbidden")

	// ErrValidationFailed is returned when package validation fails.
	ErrValidationFailed = errors.New("package validation failed")

	// ErrChecksumMismatch is returned when package integrity verification fails.
	ErrChecksumMismatch = errors.New("checksum mismatch")
)
