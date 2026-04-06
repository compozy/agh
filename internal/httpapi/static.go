package httpapi

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	webassets "github.com/pedronauck/agh/web"
)

func newStaticFS() (fs.FS, error) {
	return fs.Sub(webassets.DistFS, "dist")
}

func (h *Handlers) serveStaticRoute(c *gin.Context) {
	if c == nil {
		return
	}
	if h == nil || h.staticFS == nil {
		respondNotFound(c)
		return
	}

	requestPath := normalizedRequestPath(c.Request.URL.Path)
	if isStaticBypassPath(requestPath) || !isStaticRequestMethod(c.Request.Method) {
		respondNotFound(c)
		return
	}

	if asset, ok := h.resolveStaticAsset(requestPath); ok {
		h.serveAsset(c, asset)
		return
	}
	if shouldServeSPAIndex(requestPath) {
		h.serveAsset(c, "index.html")
		return
	}

	respondNotFound(c)
}

func (h *Handlers) resolveStaticAsset(requestPath string) (string, bool) {
	if h == nil || h.staticFS == nil {
		return "", false
	}

	asset := strings.TrimPrefix(path.Clean("/"+strings.TrimSpace(requestPath)), "/")
	if asset == "." || asset == "" {
		return "index.html", true
	}
	if info, err := fs.Stat(h.staticFS, asset); err == nil && !info.IsDir() {
		return asset, true
	}

	return "", false
}

func (h *Handlers) serveAsset(c *gin.Context, asset string) {
	data, err := fs.ReadFile(h.staticFS, strings.TrimPrefix(asset, "/"))
	if err != nil {
		respondNotFound(c)
		return
	}

	http.ServeContent(c.Writer, c.Request, path.Base(asset), h.startedAt, bytes.NewReader(data))
}

func normalizedRequestPath(rawPath string) string {
	clean := path.Clean("/" + strings.TrimSpace(rawPath))
	if clean == "." {
		return "/"
	}
	return clean
}

func isStaticBypassPath(requestPath string) bool {
	return requestPath == "/api" ||
		strings.HasPrefix(requestPath, "/api/") ||
		requestPath == "/ws" ||
		strings.HasPrefix(requestPath, "/ws/")
}

func isStaticRequestMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead:
		return true
	default:
		return false
	}
}

func shouldServeSPAIndex(requestPath string) bool {
	if requestPath == "/" {
		return true
	}

	lastSegment := path.Base(requestPath)
	return !strings.Contains(lastSegment, ".")
}

func respondNotFound(c *gin.Context) {
	c.String(http.StatusNotFound, "404 page not found")
}
