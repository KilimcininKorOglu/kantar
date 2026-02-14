// Package migrations embeds SQL migration files for use by the database package.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
