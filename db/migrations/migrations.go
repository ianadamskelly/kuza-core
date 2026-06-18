package migrations

import "embed"

// Files contains the SQL migrations applied by Kuza Core at startup.
//
//go:embed *.sql
var Files embed.FS
