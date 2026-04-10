//go:build !dev

package metis

import "embed"

//go:embed all:web/dist
var WebDist embed.FS
