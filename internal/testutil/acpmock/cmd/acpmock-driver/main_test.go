package main

import (
	"strings"
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/testutil/acpmock"
)

func TestExtractPromptTextPreservesAugmentedPromptDiagnostics(t *testing.T) {
	t.Parallel()

	prompt := "Session instructions\n\n" +
		"User request:\n\n" +
		"<agh-situation-context>{}</agh-situation-context>\n\n" +
		"Relevant durable memory for this turn:\n" +
		"- Auth [workspace]\n\n" +
		"User message:\n" +
		"hello alpha"
	blocks := []acpsdk.ContentBlock{
		acpsdk.TextBlock("ignored"),
		acpsdk.TextBlock(prompt),
	}

	if got, want := extractPromptText(blocks), prompt; got != want {
		t.Fatalf("extractPromptText() = %q, want %q", got, want)
	}
}

func TestExtractPromptTextPreservesAugmentedPromptWithoutNestedMessageMarker(t *testing.T) {
	t.Parallel()

	prompt := "Session instructions\n\n" +
		"User request:\n\n" +
		"<agh-situation-context>{}</agh-situation-context>\n\n" +
		"hello alpha"
	blocks := []acpsdk.ContentBlock{
		acpsdk.TextBlock(prompt),
	}

	if got, want := extractPromptText(blocks), prompt; got != want {
		t.Fatalf("extractPromptText() = %q, want %q", got, want)
	}
}

func TestMockAgentSelectTurnDoesNotCountUnmatchedPrompts(t *testing.T) {
	t.Parallel()

	agent := &mockAgent{
		agent: acpmock.AgentFixture{
			Name: "alpha",
			Turns: []acpmock.TurnFixture{
				{
					Name: "first",
					Match: acpmock.TurnMatch{
						TurnSource: acp.PromptTurnSourceUser,
						UserText:   "first prompt",
						Occurrence: 1,
					},
				},
				{
					Name: "second",
					Match: acpmock.TurnMatch{
						TurnSource: acp.PromptTurnSourceUser,
						UserText:   "second prompt",
						Occurrence: 2,
					},
				},
			},
		},
		sessions: map[string]*sessionState{},
	}
	meta := acp.PromptMeta{TurnSource: acp.PromptTurnSourceUser}

	first, occurrence, err := agent.selectTurn("acp-session-1", "first prompt", meta)
	if err != nil {
		t.Fatalf("selectTurn(first) error = %v", err)
	}
	if first.Name != "first" || occurrence != 1 {
		t.Fatalf("selectTurn(first) = (%q, %d), want (first, 1)", first.Name, occurrence)
	}

	_, occurrence, err = agent.selectTurn("acp-session-1", "extractor internal prompt", meta)
	if err == nil || !strings.Contains(err.Error(), "no turn matched") {
		t.Fatalf("selectTurn(unmatched) error = %v, want no-match error", err)
	}
	if occurrence != 2 {
		t.Fatalf("selectTurn(unmatched) occurrence = %d, want next occurrence 2", occurrence)
	}

	second, occurrence, err := agent.selectTurn("acp-session-1", "second prompt", meta)
	if err != nil {
		t.Fatalf("selectTurn(second) error = %v", err)
	}
	if second.Name != "second" || occurrence != 2 {
		t.Fatalf("selectTurn(second) = (%q, %d), want (second, 2)", second.Name, occurrence)
	}
}
