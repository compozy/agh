package acp

import (
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/pedronauck/agh/internal/store"
)

func TestFailureFromErrorClassifiesFatalPromptRequestErrorsAsProcessExit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
	}{
		{
			name: "Should classify process exited guidance as process exit",
			err: &acpsdk.RequestError{
				Code:    -32603,
				Message: "Internal error: The Claude Agent process exited unexpectedly. Please start a new session.",
			},
		},
		{
			name: "Should classify session not found details as process exit",
			err: &acpsdk.RequestError{
				Code:    -32603,
				Message: "Internal error",
				Data:    map[string]any{"details": "Session not found"},
			},
		},
		{
			name: "Should classify resource not found details as process exit",
			err: &acpsdk.RequestError{
				Code:    -32603,
				Message: "Internal error",
				Data:    map[string]any{"details": "Resource not found: sess-dead"},
			},
		},
		{
			name: "Should classify peer disconnected before response as process exit",
			err: &acpsdk.RequestError{
				Code:    -32603,
				Message: "Internal error",
				Data:    map[string]any{"error": "peer disconnected before response"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			failure, ok := FailureFromError(tc.err, store.FailurePrompt)
			if !ok {
				t.Fatal("FailureFromError() ok = false, want true")
			}
			if got, want := failure.Kind, store.FailureProcess; got != want {
				t.Fatalf("FailureFromError() kind = %q, want %q", got, want)
			}
		})
	}
}

func TestFailureFromErrorPreservesGenericPromptErrors(t *testing.T) {
	t.Parallel()

	failure, ok := FailureFromError(&acpsdk.RequestError{
		Code:    -32603,
		Message: "Internal error",
		Data:    map[string]any{"details": "Tool invocation failed"},
	}, store.FailurePrompt)
	if !ok {
		t.Fatal("FailureFromError() ok = false, want true")
	}
	if got, want := failure.Kind, store.FailurePrompt; got != want {
		t.Fatalf("FailureFromError() kind = %q, want %q", got, want)
	}
}
