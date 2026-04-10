package handler

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	metis "metis"
)

// RegisterStatic serves embedded frontend assets with SPA fallback.
// In dev mode (empty FS), this is a no-op — frontend runs via Vite dev server.
func RegisterStatic(r *gin.Engine) {
	// Skip when WebDist is empty (dev mode)
	if _, err := metis.WebDist.ReadDir("."); err != nil {
		return
	}

	// Strip the "web/dist" prefix so files are served from root
	sub, err := fs.Sub(metis.WebDist, "web/dist")
	if err != nil {
		panic("failed to create sub filesystem: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(sub))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API routes should 404 normally, not fall through to SPA
		if strings.HasPrefix(path, "/api/") {
			Fail(c, http.StatusNotFound, "not found")
			return
		}

		// Try to serve the static file
		f, err := sub.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		// SPA fallback: serve index.html for all other paths
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
