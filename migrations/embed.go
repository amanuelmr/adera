// Package migrations embeds the versioned SQL schema migrations.
package migrations

import "embed"

// FS contains every *.sql migration, applied in lexical order.
//
//go:embed *.sql
var FS embed.FS
