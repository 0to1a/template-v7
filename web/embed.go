// Embeds the built frontend SPA
package web

import "embed"

//go:embed all:dist
var Dist embed.FS
