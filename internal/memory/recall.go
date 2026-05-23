package memory

import (
	"context"
	"fmt"
	"strings"
	"time"

	memcontract "github.com/compozy/agh/internal/memory/contract"
	"github.com/compozy/agh/internal/session"
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

		packaged, err := target.Recall(ctx, memcontract.Query{
			AgentName: sAgentName(target),
			QueryText: query,
		}, memcontract.RecallOptions{
			TopK:          maxRecallResults,
			RawCandidates: 20,
		})
		if err != nil {
			return message, err
		}

		block := buildPackagedRecallBlock(packaged)
		if block == "" {
			return message, nil
		}
		return block + "\n\nUser message:\n" + query, nil
	}
}

func sAgentName(store *Store) string {
	if store == nil {
		return ""
	}
	return strings.TrimSpace(store.agentName)
}

func buildPackagedRecallBlock(packaged memcontract.Packaged) string {
	return RenderRecallPromptSection(packaged, RecallPromptOptions{
		MaxEntries:    maxRecallResults,
		MaxCharacters: maxRecallCharacters,
	})
}

func buildRecallBlock(results []memcontract.SearchResult, now time.Time) string {
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
