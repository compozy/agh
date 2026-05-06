package main

import (
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
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
