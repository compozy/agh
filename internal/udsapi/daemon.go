package udsapi

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type daemonStatusPayload struct {
	Status         string    `json:"status"`
	PID            int       `json:"pid"`
	StartedAt      time.Time `json:"started_at"`
	Socket         string    `json:"socket"`
	HTTPHost       string    `json:"http_host"`
	HTTPPort       int       `json:"http_port"`
	ActiveSessions int       `json:"active_sessions"`
	TotalSessions  int       `json:"total_sessions"`
	Version        string    `json:"version,omitempty"`
}

func (h *Handlers) daemonStatus(c *gin.Context) {
	health, err := h.observer.Health(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	sessions, err := h.sessions.ListAll(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"daemon": daemonStatusPayload{
			Status:         "running",
			PID:            os.Getpid(),
			StartedAt:      h.startedAt,
			Socket:         h.config.Daemon.Socket,
			HTTPHost:       h.config.HTTP.Host,
			HTTPPort:       h.config.HTTP.Port,
			ActiveSessions: health.ActiveSessions,
			TotalSessions:  len(sessions),
			Version:        health.Version,
		},
	})
}
