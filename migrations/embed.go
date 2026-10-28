package migrations

import "embed"

// Files contains the SQL migrations used by both the CLI and deployment tooling.
//
//go:embed *.sql
var Files embed.FS
