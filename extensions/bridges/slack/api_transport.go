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

const slackAPIResponseLimit = 1 << 20

func (c *slackBotClient) callSlack(
	ctx context.Context,
	method string,
	payload any,
	result any,
	responseHeaders *http.Header,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil {
		return errors.New("slack: api client is required")
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("slack: marshal %s payload: %w", method, err)
	}
	endpoint := strings.TrimRight(strings.TrimSpace(c.baseURL), "/") + "/" + strings.TrimSpace(method)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack: build %s request: %w", method, err)
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.botToken))
	request.Header.Set("Content-Type", "application/json; charset=utf-8")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, slackAPIResponseLimit+1))
	closeErr := response.Body.Close()
	if readErr != nil {
		return fmt.Errorf("slack: read %s response: %w", method, readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("slack: close %s response: %w", method, closeErr)
	}
	if len(responseBody) > slackAPIResponseLimit {
		return fmt.Errorf("slack: %s response exceeds %d bytes", method, slackAPIResponseLimit)
	}

	var envelope slackAPIEnvelope
	if len(bytes.TrimSpace(responseBody)) > 0 {
		if err := json.Unmarshal(responseBody, &envelope); err != nil {
			return fmt.Errorf("slack: decode %s response: %w", method, err)
		}
	}

	retryAfter := parseRetryAfter(response.Header.Get("Retry-After"))
	if response.StatusCode == http.StatusTooManyRequests ||
		strings.EqualFold(strings.TrimSpace(envelope.Error), "ratelimited") {
		return &bridgesdk.RateLimitError{
			Err:        fmt.Errorf("slack api %s rate limited", strings.TrimSpace(method)),
			RetryAfter: retryAfter,
		}
	}
	if response.StatusCode >= http.StatusBadRequest {
		return classifySlackAPIError(response.StatusCode, envelope.Error, retryAfter)
	}
	if !envelope.OK {
		return classifySlackAPIError(response.StatusCode, envelope.Error, retryAfter)
	}
	if result != nil && len(bytes.TrimSpace(responseBody)) > 0 {
		if err := json.Unmarshal(responseBody, result); err != nil {
			return fmt.Errorf("slack: decode %s result: %w", method, err)
		}
	}
	if responseHeaders != nil {
		*responseHeaders = response.Header.Clone()
	}
	return nil
}
