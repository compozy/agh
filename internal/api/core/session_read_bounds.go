package core

import (
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/store"
	"github.com/gin-gonic/gin"
)

const (
	defaultSessionReadLimit = 200
	maxSessionReadLimit     = 1000
)

func applyBoundedSessionReadDefault(query store.EventQuery) (store.EventQuery, error) {
	if query.Limit <= 0 {
		query.Limit = defaultSessionReadLimit
	}
	if query.Limit > maxSessionReadLimit {
		return store.EventQuery{}, fmt.Errorf("session read limit must be <= %d", maxSessionReadLimit)
	}
	if err := query.Validate(); err != nil {
		return store.EventQuery{}, err
	}
	return query, nil
}

func rejectTranscriptBackwardCursor(c *gin.Context) error {
	if strings.TrimSpace(c.Query("before_sequence")) == "" {
		return nil
	}
	return fmt.Errorf("before_sequence is only supported on /transcript")
}

func parseSessionTranscriptQuery(c *gin.Context) (store.EventQuery, error) {
	limit, err := ParseOptionalInt(c.Query("limit"))
	if err != nil {
		return store.EventQuery{}, err
	}
	afterSequence, err := ParseOptionalInt64(c.Query("after_sequence"))
	if err != nil {
		return store.EventQuery{}, err
	}
	beforeSequence, err := ParseOptionalInt64(c.Query("before_sequence"))
	if err != nil {
		return store.EventQuery{}, err
	}
	query := store.EventQuery{
		Limit:          limit,
		AfterSequence:  afterSequence,
		BeforeSequence: beforeSequence,
	}
	return applyBoundedSessionReadDefault(query)
}
