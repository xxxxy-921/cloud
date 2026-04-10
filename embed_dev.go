//go:build dev

package metis

import "embed"

// Empty FS for development — frontend runs via Vite dev server.
var WebDist embed.FS
