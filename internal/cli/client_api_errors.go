package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	diagnosticspkg "github.com/compozy/agh/internal/diagnostics"
)

func readAPIError(response *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("cli: read api error response: %w", err)
	}
	return readAPIErrorBody(response.StatusCode, response.Status, body)
}

func readAPIErrorBody(statusCode int, status string, body []byte) error {
	var payload contract.ErrorPayload
	if len(body) > 0 && json.Unmarshal(body, &payload) == nil && strings.TrimSpace(payload.Error) != "" {
		cause := errors.New(redactToolDiagnostic(payload.Error))
		if payload.Diagnostic != nil {
			return diagnosticspkg.NewStructuredError(*payload.Diagnostic, cause)
		}
		return cause
	}
	var memoryPayload contract.MemoryErrorPayload
	if len(body) > 0 && json.Unmarshal(body, &memoryPayload) == nil &&
		strings.TrimSpace(memoryPayload.Code) != "" {
		message := strings.TrimSpace(memoryPayload.Message)
		if message == "" {
			message = strings.TrimSpace(memoryPayload.Code)
		}
		return fmt.Errorf("%s: %s", strings.TrimSpace(memoryPayload.Code), redactToolDiagnostic(message))
	}
	var toolPayload contract.ToolErrorResponse
	if len(body) > 0 && json.Unmarshal(body, &toolPayload) == nil && toolPayload.Error.Code != "" {
		return newToolAPIError(statusCode, status, toolPayload)
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		message = status
	}
	message = redactToolDiagnostic(message)
	if strings.TrimSpace(status) == "" {
		return errors.New(message)
	}
	return fmt.Errorf("daemon api %s: %s", status, message)
}

func drainResponseBody(method string, path string, body io.Reader) error {
	if _, err := io.Copy(io.Discard, body); err != nil {
		return fmt.Errorf("cli: drain %s %s response: %w", method, path, err)
	}
	return nil
}
