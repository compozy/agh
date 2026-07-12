package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/compozy/agh/internal/bridgesdk"
)

const telegramAPIResponseLimit = 1 << 20

func (c *telegramBotClient) callTelegram(
	ctx context.Context,
	method string,
	payload any,
	result any,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil {
		return errors.New("telegram: bot api client is required")
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram: marshal %s payload: %w", method, err)
	}
	endpoint := strings.TrimRight(strings.TrimSpace(c.baseURL), "/") +
		"/bot" + strings.TrimSpace(c.botToken) + "/" + strings.TrimSpace(method)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: build %s request", method)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return &bridgesdk.TransientError{Err: errors.New("telegram bot api transport failed")}
	}
	raw, readErr := io.ReadAll(io.LimitReader(response.Body, telegramAPIResponseLimit+1))
	closeErr := response.Body.Close()
	if readErr != nil {
		return fmt.Errorf("telegram: read %s response: %w", method, readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("telegram: close %s response: %w", method, closeErr)
	}
	if len(raw) > telegramAPIResponseLimit {
		return fmt.Errorf("telegram: %s response exceeds %d bytes", method, telegramAPIResponseLimit)
	}

	responseEnvelope := telegramAPIEnvelope[json.RawMessage]{}
	if err := json.Unmarshal(raw, &responseEnvelope); err != nil {
		return &bridgesdk.HTTPError{
			StatusCode: response.StatusCode,
			Message:    fmt.Sprintf("telegram %s returned invalid JSON", method),
		}
	}
	if response.StatusCode >= http.StatusBadRequest || !responseEnvelope.OK {
		return classifyTelegramHTTPError(response.StatusCode, responseEnvelope)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(responseEnvelope.Result, result); err != nil {
		return fmt.Errorf("telegram: decode %s result: %w", method, err)
	}
	return nil
}
