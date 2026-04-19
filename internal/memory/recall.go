package memory

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/session"
)

const (
	maxRecallResults    = 3
	maxRecallCharacters = 1500
	// RecallAugmenterBudget is the declared daemon-side budget for durable
	// memory recall when it participates in the prompt augmentation composite.
	RecallAugmenterBudget = maxRecallCharacters
)

// NewRecallAugmenter returns a bounded prompt augmenter that prepends durable
// memory recall ahead of the live user message.
func NewRecallAugmenter(store *Store) session.PromptInputAugmenter {
	if store == nil {
		return nil
	}

	return func(ctx context.Context, sess *session.Session, message string) (string, error) {
		if sess == nil {
			return message, nil
		}

		query := strings.TrimSpace(message)
		if query == "" {
			return message, nil
		}

		info := sess.Info()
		workspaceRoot := strings.TrimSpace(info.Workspace)
		target := store
		if workspaceRoot != "" {
			target = store.ForWorkspace(workspaceRoot)
		}

		results, err := target.Search(ctx, query, SearchOptions{
			Workspace: workspaceRoot,
			Limit:     maxRecallResults,
		})
		if err != nil {
			return message, err
		}

		now := time.Now().UTC()
		block := buildRecallBlock(results, now)
		if block == "" {
			return message, nil
		}
		return block + "\n\nUser message:\n" + query, nil
	}
}

func buildRecallBlock(results []SearchResult, now time.Time) string {
	if len(results) == 0 {
		return ""
	}

	lines := make([]string, 0, len(results))
	used := 0
	for _, result := range results {
		if result.Score <= 0 {
			continue
		}
		entryLines := []string{
			fmt.Sprintf("- %s [%s]", strings.TrimSpace(result.Name), result.Scope.Normalize()),
		}
		if snippet := strings.TrimSpace(result.Snippet); snippet != "" {
			entryLines = append(entryLines, "  Snippet: "+snippet)
		}
		if warning := FreshnessWarning(result.ModTime, now); warning != "" {
			entryLines = append(entryLines, "  Freshness: "+warning)
		}
		entry := strings.Join(entryLines, "\n")
		if used > 0 && used+2+len(entry) > maxRecallCharacters {
			break
		}
		lines = append(lines, entry)
		used += len(entry)
		if len(lines) == maxRecallResults {
			break
		}
	}

	if len(lines) == 0 {
		return ""
	}

	return strings.Join([]string{
		"Relevant durable memory for this turn:",
		strings.Join(lines, "\n"),
		"Use recalled memory only when it remains consistent with the current repository and runtime state.",
	}, "\n")
}
