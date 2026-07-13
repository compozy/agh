// Package schema embeds the global database schema and migration directory.
package schema

import "embed"

// Files contains declarative schema fragments, SQL migrations, and atlas.sum.
//
//go:embed definitions/*.sql migrations/*.sql migrations/atlas.sum
var Files embed.FS
