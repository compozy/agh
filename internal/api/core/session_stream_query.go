package core

import (
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
	"github.com/gin-gonic/gin"
)

func (h *BaseHandlers) parseSessionStreamEventQuery(c *gin.Context) (store.EventQuery, error) {
	query, err := ParseSessionEventQuery(c)
	if err != nil {
		return store.EventQuery{}, err
	}
	if lastEventID := strings.TrimSpace(c.GetHeader("Last-Event-ID")); lastEventID != "" {
		query.AfterSequence, err = parseLastEventID(lastEventID, h.transportName())
		if err != nil {
			return store.EventQuery{}, err
		}
	}
	if err := query.Validate(); err != nil {
		return store.EventQuery{}, fmt.Errorf("invalid stream query: %w", err)
	}
	return query, nil
}
