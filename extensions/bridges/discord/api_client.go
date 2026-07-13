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

	"github.com/compozy/agh/internal/bridgesdk"
)

func (c *discordBotClient) GetBotUser(ctx context.Context) (*discordBotIdentity, error) {
	var result discordBotIdentity
	if err := c.callJSON(ctx, http.MethodGet, "/users/@me", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *discordBotClient) PostMessage(
	ctx context.Context,
	req discordPostMessageRequest,
) (*discordPostedMessage, error) {
	var result discordPostedMessage
	if err := c.callJSON(
		ctx,
		http.MethodPost,
		"/channels/"+strings.TrimSpace(req.ChannelID)+"/messages",
		req,
		&result,
	); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *discordBotClient) UpdateMessage(ctx context.Context, req discordUpdateMessageRequest) error {
	return c.callJSON(
		ctx,
		http.MethodPatch,
		"/channels/"+strings.TrimSpace(req.ChannelID)+"/messages/"+strings.TrimSpace(req.MessageID),
		req,
		nil,
	)
}

func (c *discordBotClient) DeleteMessage(ctx context.Context, req discordDeleteMessageRequest) error {
	return c.callJSON(
		ctx,
		http.MethodDelete,
		"/channels/"+strings.TrimSpace(req.ChannelID)+"/messages/"+strings.TrimSpace(req.MessageID),
		nil,
		nil,
	)
}

func (c *discordBotClient) callJSON(ctx context.Context, method string, path string, payload any, out any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil {
		return errors.New("discord: api client is required")
	}

	var body io.Reader
	if payload != nil {
		buf := &bytes.Buffer{}
		encoder := json.NewEncoder(buf)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(payload); err != nil {
			return err
		}
		body = buf
	}

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.baseURL, "/")+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+strings.TrimSpace(c.botToken))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(readResponseBody(resp.Body))
		httpErr := &bridgesdk.HTTPError{
			StatusCode: resp.StatusCode,
			Message:    firstNonEmpty(message, fmt.Sprintf("discord: http %d", resp.StatusCode)),
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
		return httpErr
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("discord: decode response: %w", err)
	}
	return nil
}
