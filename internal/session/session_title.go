package session

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	automaticSessionTitleMaxRunes = 64
	automaticSessionTitleMaxWords = 8
)

type sessionTitleClaim struct {
	title             string
	previousUpdatedAt time.Time
}

func (m *Manager) ensureAutomaticSessionTitle(
	ctx context.Context,
	session *Session,
	message string,
) error {
	if m == nil || session == nil {
		return nil
	}
	claim, ok := session.claimAutomaticTitle(message, m.now())
	if !ok {
		return nil
	}
	if err := m.persistSessionIdentity(ctx, session); err != nil {
		session.rollbackAutomaticTitle(claim)
		rollbackErr := m.writeMeta(session)
		if rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("session: roll back automatic title: %w", rollbackErr))
		}
		return err
	}
	m.publishSessionCatalogEvent(sessionCatalogEventFromInfo(CatalogEventUpserted, session.Info()))
	return nil
}

func (s *Session) claimAutomaticTitle(message string, now time.Time) (sessionTitleClaim, bool) {
	if s == nil {
		return sessionTitleClaim{}, false
	}
	title := automaticSessionTitle(message)
	if title == "" {
		return sessionTitleClaim{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if normalizeSessionType(s.Type) != SessionTypeUser || strings.TrimSpace(s.Name) != "" {
		return sessionTitleClaim{}, false
	}
	claim := sessionTitleClaim{title: title, previousUpdatedAt: s.UpdatedAt}
	s.Name = title
	s.UpdatedAt = now.UTC()
	return claim, true
}

func (s *Session) rollbackAutomaticTitle(claim sessionTitleClaim) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Name != claim.title {
		return
	}
	s.Name = ""
	s.UpdatedAt = claim.previousUpdatedAt
}

func automaticSessionTitle(message string) string {
	words := strings.Fields(message)
	if len(words) == 0 {
		return ""
	}
	if strings.EqualFold(words[0], "/goal") {
		words = words[1:]
	}
	if len(words) == 0 {
		return ""
	}

	truncated := len(words) > automaticSessionTitleMaxWords
	if truncated {
		words = words[:automaticSessionTitleMaxWords]
	}
	title := strings.Trim(strings.Join(words, " "), " \t\n\r#>*_`-.,;:!?")
	if title == "" {
		return ""
	}
	if utf8.RuneCountInString(title) > automaticSessionTitleMaxRunes {
		runes := []rune(title)
		title = strings.TrimSpace(string(runes[:automaticSessionTitleMaxRunes-1]))
		truncated = true
	}
	if truncated {
		title = strings.TrimRight(title, " \t\n\r.,;:!?") + "…"
	}
	return title
}
