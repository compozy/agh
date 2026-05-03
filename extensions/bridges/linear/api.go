package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/bridgesdk"
)

type linearAPI interface {
	ValidateAuth(ctx context.Context) (*linearViewer, error)
	CreateComment(ctx context.Context, issueID string, body string, parentID string) (*linearComment, error)
	UpdateComment(ctx context.Context, commentID string, body string) (*linearComment, error)
	DeleteComment(ctx context.Context, commentID string) error
	CreateAgentActivity(ctx context.Context, agentSessionID string, body string) (*linearAgentActivity, error)
}

type linearClient struct {
	cfg        resolvedInstanceConfig
	httpClient *http.Client
	now        func() time.Time
}

type linearViewer struct {
	ID             string
	DisplayName    string
	OrganizationID string
}

type linearComment struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	ParentID  string    `json:"parentId"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Issue     struct {
		ID string `json:"id"`
	} `json:"issue"`
}

type linearAgentActivity struct {
	ID            string `json:"id"`
	SourceComment *struct {
		ID string `json:"id"`
	} `json:"sourceComment"`
}

type linearOAuthTokenCache struct {
	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

type linearOAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type linearGraphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type linearGraphQLError struct {
	Message string `json:"message"`
}

type linearGraphQLResponse[T any] struct {
	Data   T                    `json:"data"`
	Errors []linearGraphQLError `json:"errors,omitempty"`
}

func linearCredentialedHTTPClient(base *http.Client) *http.Client {
	if base == nil {
		return &http.Client{
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}

	client := *base
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &client
}

func (c *linearClient) ValidateAuth(ctx context.Context) (*linearViewer, error) {
	type viewerResponse struct {
		Viewer struct {
			ID           string `json:"id"`
			DisplayName  string `json:"displayName"`
			Organization struct {
				ID string `json:"id"`
			} `json:"organization"`
		} `json:"viewer"`
	}

	response, err := doLinearGraphQL[viewerResponse](ctx, c, linearGraphQLRequest{
		Query: `
query LinearProviderViewer {
  viewer {
    id
    displayName
    organization {
      id
    }
  }
}`,
	})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(response.Viewer.ID) == "" {
		return nil, &bridgesdk.AuthError{Err: errors.New("linear: viewer id is missing")}
	}
	if strings.TrimSpace(response.Viewer.Organization.ID) == "" {
		return nil, &bridgesdk.AuthError{Err: errors.New("linear: viewer organization id is missing")}
	}
	return &linearViewer{
		ID:             strings.TrimSpace(response.Viewer.ID),
		DisplayName:    strings.TrimSpace(response.Viewer.DisplayName),
		OrganizationID: strings.TrimSpace(response.Viewer.Organization.ID),
	}, nil
}

func (c *linearClient) CreateComment(
	ctx context.Context,
	issueID string,
	body string,
	parentID string,
) (*linearComment, error) {
	type createCommentResponse struct {
		CommentCreate struct {
			Success bool          `json:"success"`
			Comment linearComment `json:"comment"`
		} `json:"commentCreate"`
	}

	variables := map[string]any{
		"issueId": strings.TrimSpace(issueID),
		"body":    body,
	}
	if strings.TrimSpace(parentID) != "" {
		variables["parentId"] = strings.TrimSpace(parentID)
	}
	response, err := doLinearGraphQL[createCommentResponse](ctx, c, linearGraphQLRequest{
		Query: `
mutation LinearProviderCreateComment($issueId: String!, $body: String!, $parentId: String) {
  commentCreate(input: { issueId: $issueId, body: $body, parentId: $parentId }) {
    success
    comment {
      id
      body
      parentId
      url
      createdAt
      updatedAt
      issue {
        id
      }
    }
  }
}`,
		Variables: variables,
	})
	if err != nil {
		return nil, err
	}
	if !response.CommentCreate.Success || strings.TrimSpace(response.CommentCreate.Comment.ID) == "" {
		return nil, &bridgesdk.PermanentError{Err: errors.New("linear: comment creation failed")}
	}
	comment := response.CommentCreate.Comment
	return &comment, nil
}

func (c *linearClient) UpdateComment(ctx context.Context, commentID string, body string) (*linearComment, error) {
	type updateCommentResponse struct {
		CommentUpdate struct {
			Success bool          `json:"success"`
			Comment linearComment `json:"comment"`
		} `json:"commentUpdate"`
	}

	response, err := doLinearGraphQL[updateCommentResponse](ctx, c, linearGraphQLRequest{
		Query: `
mutation LinearProviderUpdateComment($id: String!, $body: String!) {
  commentUpdate(id: $id, input: { body: $body }) {
    success
    comment {
      id
      body
      parentId
      url
      createdAt
      updatedAt
      issue {
        id
      }
    }
  }
}`,
		Variables: map[string]any{
			"id":   strings.TrimSpace(commentID),
			"body": body,
		},
	})
	if err != nil {
		return nil, err
	}
	if !response.CommentUpdate.Success || strings.TrimSpace(response.CommentUpdate.Comment.ID) == "" {
		return nil, &bridgesdk.PermanentError{Err: errors.New("linear: comment update failed")}
	}
	comment := response.CommentUpdate.Comment
	return &comment, nil
}

func (c *linearClient) DeleteComment(ctx context.Context, commentID string) error {
	type deleteCommentResponse struct {
		CommentDelete struct {
			Success bool `json:"success"`
		} `json:"commentDelete"`
	}

	response, err := doLinearGraphQL[deleteCommentResponse](ctx, c, linearGraphQLRequest{
		Query: `
mutation LinearProviderDeleteComment($id: String!) {
  commentDelete(id: $id) {
    success
  }
}`,
		Variables: map[string]any{
			"id": strings.TrimSpace(commentID),
		},
	})
	if err != nil {
		return err
	}
	if !response.CommentDelete.Success {
		return &bridgesdk.PermanentError{Err: errors.New("linear: comment delete failed")}
	}
	return nil
}

func (c *linearClient) CreateAgentActivity(
	ctx context.Context,
	agentSessionID string,
	body string,
) (*linearAgentActivity, error) {
	type createActivityResponse struct {
		AgentActivityCreate struct {
			Success       bool                `json:"success"`
			AgentActivity linearAgentActivity `json:"agentActivity"`
		} `json:"agentActivityCreate"`
	}

	response, err := doLinearGraphQL[createActivityResponse](ctx, c, linearGraphQLRequest{
		Query: `
mutation LinearProviderCreateAgentActivity($agentSessionId: String!, $body: String!) {
  agentActivityCreate(input: {
    agentSessionId: $agentSessionId
    content: {
      type: response
      body: $body
    }
  }) {
    success
    agentActivity {
      id
      sourceComment {
        id
      }
    }
  }
}`,
		Variables: map[string]any{
			"agentSessionId": strings.TrimSpace(agentSessionID),
			"body":           body,
		},
	})
	if err != nil {
		return nil, err
	}
	if !response.AgentActivityCreate.Success || strings.TrimSpace(response.AgentActivityCreate.AgentActivity.ID) == "" {
		return nil, &bridgesdk.PermanentError{Err: errors.New("linear: agent activity creation failed")}
	}
	activity := response.AgentActivityCreate.AgentActivity
	return &activity, nil
}

func doLinearGraphQL[T any](ctx context.Context, c *linearClient, request linearGraphQLRequest) (T, error) {
	var zero T

	payload, err := json.Marshal(request)
	if err != nil {
		return zero, fmt.Errorf("linear: marshal graphql request: %w", err)
	}

	if !validLinearCredentialedURL(c.cfg.apiBaseURL) {
		return zero, &bridgesdk.PermanentError{Err: errors.New("linear: api base url is invalid")}
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.graphqlURL(), bytes.NewReader(payload))
	if err != nil {
		return zero, fmt.Errorf("linear: build graphql request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+c.authToken(ctx))

	httpResponse, err := linearCredentialedHTTPClient(c.httpClient).Do(httpRequest)
	if err != nil {
		return zero, classifyLinearTransportError(err)
	}
	defer func() {
		_ = httpResponse.Body.Close()
	}()

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return zero, fmt.Errorf("linear: read graphql response: %w", err)
	}
	if httpResponse.StatusCode >= 400 {
		return zero, classifyLinearHTTPError(httpResponse.StatusCode, body)
	}

	envelope := linearGraphQLResponse[T]{}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return zero, fmt.Errorf("linear: decode graphql response: %w", err)
	}
	if len(envelope.Errors) > 0 {
		messages := make([]string, 0, len(envelope.Errors))
		for _, item := range envelope.Errors {
			if text := strings.TrimSpace(item.Message); text != "" {
				messages = append(messages, text)
			}
		}
		return zero, &bridgesdk.PermanentError{
			Err: fmt.Errorf("linear: graphql error: %s", strings.Join(messages, "; ")),
		}
	}

	return envelope.Data, nil
}

func (c *linearClient) authToken(ctx context.Context) string {
	if c.cfg.authMode != linearAuthModeOAuth {
		return c.cfg.apiKey
	}
	return c.ensureOAuthToken(ctx)
}

func (c *linearClient) ensureOAuthToken(ctx context.Context) string {
	cache := c.cfg.oauthTokenCache
	if cache == nil {
		return ""
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	now := c.now()
	if strings.TrimSpace(cache.token) != "" &&
		(cache.expiresAt.IsZero() || cache.expiresAt.After(now.Add(time.Minute))) {
		return cache.token
	}

	values := url.Values{}
	values.Set("grant_type", "client_credentials")
	values.Set("client_id", c.cfg.clientID)
	values.Set("client_secret", c.cfg.clientSecret)
	values.Set("scope", strings.Join(defaultLinearOAuthScopes(c.cfg.mode), ","))

	if !validLinearCredentialedURL(c.cfg.oauthTokenURL) {
		cache.token = ""
		cache.expiresAt = time.Time{}
		return ""
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.cfg.oauthTokenURL,
		strings.NewReader(values.Encode()),
	)
	if err != nil {
		cache.token = ""
		cache.expiresAt = time.Time{}
		return ""
	}
	httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResponse, err := linearCredentialedHTTPClient(c.httpClient).Do(httpRequest)
	if err != nil {
		cache.token = ""
		cache.expiresAt = time.Time{}
		return ""
	}
	defer func() {
		_ = httpResponse.Body.Close()
	}()

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		cache.token = ""
		cache.expiresAt = time.Time{}
		return ""
	}
	if httpResponse.StatusCode >= 400 {
		cache.token = ""
		cache.expiresAt = time.Time{}
		return ""
	}

	response := linearOAuthTokenResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		cache.token = ""
		cache.expiresAt = time.Time{}
		return ""
	}
	cache.token = strings.TrimSpace(response.AccessToken)
	if response.ExpiresIn > 0 {
		cache.expiresAt = now.Add(time.Duration(response.ExpiresIn) * time.Second)
	} else {
		cache.expiresAt = time.Time{}
	}
	return cache.token
}

func defaultLinearOAuthScopes(mode string) []string {
	scopes := []string{"read", "write", "comments:create", "issues:create"}
	if strings.TrimSpace(mode) == linearModeAgentSessions {
		scopes = append(scopes, "app:mentionable")
	}
	return scopes
}

func classifyLinearHTTPError(statusCode int, body []byte) error {
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = http.StatusText(statusCode)
	}

	switch statusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &bridgesdk.AuthError{Err: errors.New(message)}
	case http.StatusTooManyRequests:
		return &bridgesdk.RateLimitError{Err: errors.New(message)}
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return &bridgesdk.HTTPError{StatusCode: http.StatusRequestTimeout, Message: message}
	}
	if statusCode >= 500 {
		return &bridgesdk.TransientError{Err: errors.New(message)}
	}
	return &bridgesdk.PermanentError{Err: errors.New(message)}
}

func classifyLinearTransportError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &bridgesdk.HTTPError{StatusCode: http.StatusRequestTimeout, Message: err.Error()}
	}
	if errors.Is(err, context.Canceled) {
		return &bridgesdk.TransientError{Err: err}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return &bridgesdk.HTTPError{StatusCode: http.StatusRequestTimeout, Message: err.Error()}
		}
		return &bridgesdk.TransientError{Err: err}
	}
	return &bridgesdk.TransientError{Err: err}
}
