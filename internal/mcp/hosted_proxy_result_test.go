package mcp

import (
	"encoding/json"
	"testing"

	sdkmcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/pedronauck/agh/internal/tools"
)

func TestHostedToolResultContract(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve hosted MCP error flag", func(t *testing.T) {
		t.Parallel()

		result, err := hostedToolResult(tools.ToolResult{
			Content: []tools.ToolContent{{Type: "text", Text: "denied"}},
			Metadata: map[string]json.RawMessage{
				toolResultIsErrorKey: json.RawMessage(`true`),
			},
		})
		if err != nil {
			t.Fatalf("hostedToolResult() error = %v", err)
		}
		if result == nil || !result.IsError {
			t.Fatalf("hostedToolResult() = %#v, want error result", result)
		}
	})

	t.Run("Should preserve hosted MCP media content", func(t *testing.T) {
		t.Parallel()

		result, err := hostedToolResult(tools.ToolResult{
			Content: []tools.ToolContent{
				{Type: "image", Data: json.RawMessage(`"aW1n"`), MIMEType: "image/png"},
				{Type: "audio", Data: json.RawMessage(`"YXVkaW8="`), MIMEType: "audio/mpeg"},
			},
		})
		if err != nil {
			t.Fatalf("hostedToolResult() error = %v", err)
		}
		if result == nil || len(result.Content) != 2 {
			t.Fatalf("hostedToolResult() = %#v, want two media blocks", result)
		}

		imageContent, ok := result.Content[0].(sdkmcp.ImageContent)
		if !ok {
			t.Fatalf("result.Content[0] type = %T, want sdkmcp.ImageContent", result.Content[0])
		}
		if imageContent.Data != "aW1n" || imageContent.MIMEType != "image/png" {
			t.Fatalf("image content = %#v, want data and MIME type preserved", imageContent)
		}

		audioContent, ok := result.Content[1].(sdkmcp.AudioContent)
		if !ok {
			t.Fatalf("result.Content[1] type = %T, want sdkmcp.AudioContent", result.Content[1])
		}
		if audioContent.Data != "YXVkaW8=" || audioContent.MIMEType != "audio/mpeg" {
			t.Fatalf("audio content = %#v, want data and MIME type preserved", audioContent)
		}
	})
}
