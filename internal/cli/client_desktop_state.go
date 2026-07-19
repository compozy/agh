package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/gorilla/websocket"
)

const desktopStateClientHandshakeTimeout = 10 * time.Second

// DesktopStateClient is the CLI transport contract for workspace desktop state.
type DesktopStateClient interface {
	ListDesktopState(context.Context, string) (contract.DesktopStateListResponse, error)
	GetDesktopState(context.Context, string, string) (contract.DesktopStateEntry, error)
	PutDesktopState(
		context.Context,
		string,
		string,
		contract.DesktopStatePutRequest,
	) (contract.DesktopStateEntry, error)
	ApplyDesktopState(
		context.Context,
		string,
		contract.DesktopStateApplyRequest,
	) (contract.DesktopStateApplyResponse, error)
	DeleteDesktopState(context.Context, string, string, *contract.DesktopStateSafeNumber) error
	WatchDesktopState(context.Context, string, func(contract.DesktopStateEventFrame) error) error
}

func (c *unixSocketClient) ListDesktopState(
	ctx context.Context,
	workspace string,
) (contract.DesktopStateListResponse, error) {
	var response contract.DesktopStateListResponse
	if err := c.doJSON(ctx, http.MethodGet, desktopStateClientPath(workspace), nil, nil, &response); err != nil {
		return contract.DesktopStateListResponse{}, err
	}
	return response, nil
}

func (c *unixSocketClient) GetDesktopState(
	ctx context.Context,
	workspace string,
	key string,
) (contract.DesktopStateEntry, error) {
	var response contract.DesktopStateEntry
	path := desktopStateClientPath(workspace) + "/" + url.PathEscape(strings.TrimSpace(key))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return contract.DesktopStateEntry{}, err
	}
	return response, nil
}

func (c *unixSocketClient) PutDesktopState(
	ctx context.Context,
	workspace string,
	key string,
	request contract.DesktopStatePutRequest,
) (contract.DesktopStateEntry, error) {
	var response contract.DesktopStateEntry
	path := desktopStateClientPath(workspace) + "/" + url.PathEscape(strings.TrimSpace(key))
	if err := c.doJSON(ctx, http.MethodPut, path, nil, request, &response); err != nil {
		return contract.DesktopStateEntry{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ApplyDesktopState(
	ctx context.Context,
	workspace string,
	request contract.DesktopStateApplyRequest,
) (contract.DesktopStateApplyResponse, error) {
	var response contract.DesktopStateApplyResponse
	path := desktopStateClientPath(workspace) + "/apply"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return contract.DesktopStateApplyResponse{}, err
	}
	return response, nil
}

func (c *unixSocketClient) DeleteDesktopState(
	ctx context.Context,
	workspace string,
	key string,
	ifRev *contract.DesktopStateSafeNumber,
) error {
	path := desktopStateClientPath(workspace) + "/" + url.PathEscape(strings.TrimSpace(key))
	query := url.Values{}
	if ifRev != nil {
		query.Set("if_rev", strconv.FormatUint(uint64(*ifRev), 10))
	}
	return c.doJSON(ctx, http.MethodDelete, path, query, nil, nil)
}

func (c *unixSocketClient) WatchDesktopState(
	ctx context.Context,
	workspace string,
	handler func(contract.DesktopStateEventFrame) error,
) (returnErr error) {
	if handler == nil {
		return errors.New("cli: desktop-state watch handler is required")
	}
	dialer := websocket.Dialer{
		HandshakeTimeout: desktopStateClientHandshakeTimeout,
		NetDialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", c.socketPath)
		},
	}
	path := desktopStateClientPath(workspace) + "/stream"
	conn, response, err := dialer.DialContext(ctx, "ws://unix"+path, nil)
	if err != nil {
		if response != nil {
			return readAndCloseDesktopStateHandshakeError(response)
		}
		return fmt.Errorf("cli: dial desktop-state stream: %w", err)
	}
	defer func() {
		closeErr := conn.Close()
		if closeErr != nil {
			closeErr = fmt.Errorf("cli: close desktop-state stream: %w", closeErr)
		}
		returnErr = errors.Join(returnErr, closeErr)
	}()
	if err := conn.WriteJSON(contract.DesktopStateSubscribeFrame{Op: "sub"}); err != nil {
		return fmt.Errorf("cli: subscribe to desktop-state stream: %w", err)
	}
	return readDesktopStateFrames(ctx, conn, handler)
}

func readAndCloseDesktopStateHandshakeError(response *http.Response) error {
	apiErr := readAPIError(response)
	closeErr := response.Body.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("cli: close desktop-state handshake response: %w", closeErr)
	}
	return errors.Join(apiErr, closeErr)
}

func readDesktopStateFrames(
	ctx context.Context,
	conn *websocket.Conn,
	handler func(contract.DesktopStateEventFrame) error,
) error {
	snapshotReceived := false
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return fmt.Errorf("cli: read desktop-state stream: %w", err)
		}
		var envelope struct {
			Op string `json:"op"`
		}
		if err := json.Unmarshal(payload, &envelope); err != nil {
			return fmt.Errorf("cli: decode desktop-state frame: %w", err)
		}
		switch strings.TrimSpace(envelope.Op) {
		case cliSnapshotKey:
			if snapshotReceived {
				return errors.New("cli: desktop-state stream sent more than one snapshot")
			}
			var snapshot contract.DesktopStateSnapshotFrame
			if err := json.Unmarshal(payload, &snapshot); err != nil {
				return fmt.Errorf("cli: decode desktop-state snapshot: %w", err)
			}
			snapshotReceived = true
		case automationEventKey:
			if !snapshotReceived {
				return errors.New("cli: desktop-state event arrived before snapshot")
			}
			var event contract.DesktopStateEventFrame
			if err := json.Unmarshal(payload, &event); err != nil {
				return fmt.Errorf("cli: decode desktop-state event: %w", err)
			}
			if err := handler(event); err != nil {
				return err
			}
		case clientErrorKey:
			var frame contract.DesktopStateErrorFrame
			if err := json.Unmarshal(payload, &frame); err != nil {
				return fmt.Errorf("cli: decode desktop-state error: %w", err)
			}
			return &desktopStateAPIError{
				statusCode: http.StatusConflict,
				payload: contract.DesktopStateErrorPayload{
					Error: string(frame.Code), Code: frame.Code, Key: frame.Key,
				},
			}
		case "pong":
		default:
			return fmt.Errorf("cli: unsupported desktop-state frame %q", envelope.Op)
		}
	}
}

func desktopStateClientPath(workspace string) string {
	return "/api/workspaces/" + url.PathEscape(strings.TrimSpace(workspace)) + "/desktop-state"
}
