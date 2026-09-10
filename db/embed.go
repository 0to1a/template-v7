// Embeds migrations so the binary always carries its schema
package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
