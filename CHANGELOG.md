# Changelog

All notable changes to this project will be documented in this file.

## [0.1.1] - 2026-04-01

### Added
- Connect all unused TypeScript types with full implementation
- Wire server-side actor and event filter query parameters for audit logs
- Wire package count stats to dashboard stat cards
- Admin user creation endpoint with role selection
- Implement upstream pull-through proxy with OAuth2 for Docker
- Wire policy engine into package approval flow
- Add upstream HTTP response caching to ResolveDependencies
- Add package read caching with invalidation on mutations
- Wire cache into Dependencies and plugins via WithCache

### Changed
- Extract shared HTTP client to internal/httpclient
- Move buildPublishBody to test helper file (Cargo)
- Remove unused OptionalMiddleware, MatchNamespace, and API token functions
- Delete stub kantarctl CLI and all references
- Update documentation for API, CLI, and build

### Fixed
- Propagate seed function errors and abort startup on failure
- Show error feedback when settings save fails
- Pass package version to sync job on approval
- Propagate database errors in processSyncJob
- Use configured upstream URL in NuGet ResolveDependencies
- Implement full hash chain verification in Verify()
- Persist JWT secret via env var to survive container restarts
- Move JWT to HttpOnly cookie with CSRF protection
- Stabilize load functions with useCallback for correct locale
- Add error handling to registry and policy toggle and save actions
- Clear error state before package approve and block actions
- Use per-job context independent of application lifecycle
- Handle empty default in env var interpolation
- Use configured upstream URL in Cargo ResolveDependencies
- Sort versions by semver before selecting latest (GoMod)
- Require explicit DB_PASSWORD via .env file
- Fix Turkish and German translations with correct special characters
- Encode package names in URLs to support slashes
- Reject login when user object is missing from response
- Deduplicate dependency resolution before upstream calls
- Hold read lock through expiry check and data copy (cache)
- Use prefix check to prevent path traversal (storage)
- Remove JWT token acceptance from URL query parameter
- Validate role against known set before update
- Prevent self-deletion and last super_admin removal
- Make auth middleware unconditional with nil-guard
- Add mutex to prevent hash chain race condition (audit)
