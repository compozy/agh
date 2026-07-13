package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/compozy/agh/internal/bridgesdk"
)

func (c *teamsBotClient) ValidateAuth(ctx context.Context) error {
	_, err := c.accessToken(ctx)
	return err
}

func (c *teamsBotClient) CreateConversation(
	ctx context.Context,
	serviceURL string,
	req teamsCreateConversationRequest,
) (*teamsConversationResourceResponse, error) {
	var out teamsConversationResourceResponse
	if err := c.callJSON(ctx, http.MethodPost, serviceURL, "/v3/conversations", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *teamsBotClient) SendActivity(
	ctx context.Context,
	serviceURL string,
	conversationID string,
	replyToID string,
	activity teamsOutboundActivity,
) (*teamsResourceResponse, error) {
	path := "/v3/conversations/" + url.PathEscape(strings.TrimSpace(conversationID)) + "/activities"
	if strings.TrimSpace(replyToID) != "" {
		path = "/v3/conversations/" + url.PathEscape(
			strings.TrimSpace(conversationID),
		) + "/activities/" + url.PathEscape(
			strings.TrimSpace(replyToID),
		)
	}
	var out teamsResourceResponse
	if err := c.callJSON(ctx, http.MethodPost, serviceURL, path, activity, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *teamsBotClient) UpdateActivity(
	ctx context.Context,
	serviceURL string,
	conversationID string,
	activityID string,
	activity teamsOutboundActivity,
) error {
	return c.callJSON(
		ctx,
		http.MethodPut,
		serviceURL,
		"/v3/conversations/"+url.PathEscape(
			strings.TrimSpace(conversationID),
		)+"/activities/"+url.PathEscape(
			strings.TrimSpace(activityID),
		),
		activity,
		nil,
	)
}

func (c *teamsBotClient) DeleteActivity(
	ctx context.Context,
	serviceURL string,
	conversationID string,
	activityID string,
) error {
	return c.callJSON(
		ctx,
		http.MethodDelete,
		serviceURL,
		"/v3/conversations/"+url.PathEscape(
			strings.TrimSpace(conversationID),
		)+"/activities/"+url.PathEscape(
			strings.TrimSpace(activityID),
		),
		nil,
		nil,
	)
}

func (c *teamsBotClient) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.cachedToken != "" && c.tokenExpiry.After(time.Now().UTC().Add(30*time.Second)) {
		token := c.cachedToken
		c.mu.Unlock()
		return token, nil
	}
	c.mu.Unlock()

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", c.cfg.appID)
	form.Set("client_secret", c.cfg.appPassword)
	form.Set("scope", teamsDefaultScope)

	tokenURL, err := validatedTeamsCredentialedURL(c.cfg.tokenURL, "token url")
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		tokenURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := teamsCredentialedHTTPClient(c.httpClient).Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", classifyTeamsHTTPError(
			resp.StatusCode,
			resp.Header.Get("Retry-After"),
			readResponseBody(resp.Body),
		)
	}
	var tokenResp teamsTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", &bridgesdk.AuthError{
			Err: errors.New("teams: token response omitted access token"),
		}
	}
	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	c.mu.Lock()
	c.cachedToken = strings.TrimSpace(tokenResp.AccessToken)
	c.tokenExpiry = time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)
	token := c.cachedToken
	c.mu.Unlock()
	return token, nil
}

func (c *teamsBotClient) callJSON(
	ctx context.Context,
	method string,
	serviceURL string,
	path string,
	payload any,
	out any,
) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	base := strings.TrimRight(normalizeURL(serviceURL), "/")
	if base == "" {
		return errors.New("teams: service url is required")
	}
	fullURL := base + path
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return classifyTeamsHTTPError(
			resp.StatusCode,
			resp.Header.Get("Retry-After"),
			readResponseBody(resp.Body),
		)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			return fmt.Errorf("teams: drain response body: %w", err)
		}
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
