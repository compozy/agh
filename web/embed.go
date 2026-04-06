// Package webassets embeds the built AGH frontend bundle for the daemon HTTP server.
package webassets

import "embed"

// DistFS embeds the built frontend assets under web/dist.
//
//go:embed all:dist
var DistFS embed.FS
