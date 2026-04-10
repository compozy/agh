package httpapi

import (
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
)

func requestLoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(c *gin.Context) {
		started := time.Now()
		c.Next()

		logger.Info(
			"httpapi: request",
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", c.Writer.Status(),
			"latency_ms", time.Since(started).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}

func corsMiddleware(boundHost string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		headers := c.Writer.Header()
		headers.Set("Access-Control-Allow-Headers", "Content-Type, Last-Event-ID, Accept")
		headers.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		headers.Set("Access-Control-Expose-Headers", "Content-Type, Last-Event-ID, x-vercel-ai-ui-message-stream")
		headers.Set("Vary", "Origin")
		if origin != "" {
			allowedOrigin, ok := resolveAllowedOrigin(origin, c.Request.Host, boundHost)
			if !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, contract.ErrorPayload{Error: "origin not allowed"})
				return
			}
			headers.Set("Access-Control-Allow-Origin", allowedOrigin)
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func resolveAllowedOrigin(origin string, requestHost string, boundHost string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}

	originHost := canonicalHost(parsed.Hostname())
	requestHostname := canonicalHost(hostOnly(requestHost))
	boundHostname := canonicalHost(hostOnly(boundHost))

	switch {
	case originHost == "" || requestHostname == "":
		return "", false
	case originHost == requestHostname:
		return origin, true
	case isLoopbackHost(originHost) && isLoopbackHost(requestHostname):
		return origin, true
	case boundHostname != "" && !isWildcardHost(boundHostname) && originHost == boundHostname:
		return origin, true
	default:
		return "", false
	}
}

func hostOnly(value string) string {
	host := strings.TrimSpace(value)
	if host == "" {
		return ""
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		return parsedHost
	}
	return host
}

func canonicalHost(value string) string {
	return strings.Trim(strings.TrimSpace(value), "[]")
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isWildcardHost(host string) bool {
	switch host {
	case "", "0.0.0.0", "::":
		return true
	default:
		return false
	}
}

func errorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}
		core.RespondError(c, http.StatusInternalServerError, c.Errors.Last(), true)
	}
}
