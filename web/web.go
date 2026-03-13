// Package web provides the embedded SPA for Kantar's admin dashboard.
package web

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var embeddedFS embed.FS

// FS returns the embedded web UI filesystem rooted at dist/.
func FS() (fs.FS, error) {
	return fs.Sub(embeddedFS, "dist")
}
