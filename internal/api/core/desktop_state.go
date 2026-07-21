package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/clientstate"
	"github.com/gin-gonic/gin"
)

const (
	desktopStateDomain    = "os_shell"
	desktopStateMaxSafeID = uint64(contract.DesktopStateMaxSafeNumber)
)

var errDesktopStateUnsafeNumber = errors.New(
	"desktop state revision or sequence exceeds the JSON safe integer range",
)

// ListDesktopState returns a key-sorted workspace snapshot and its sequence fence.
func (h *BaseHandlers) ListDesktopState(c *gin.Context) {
	if h.DesktopState == nil {
		h.respondDesktopStateError(c, clientstate.ErrClosed, "")
		return
	}
	subscription, err := h.DesktopState.Watch(
		c.Request.Context(),
		desktopStateWorkspace(c),
		[]string{desktopStateDomain},
	)
	if err != nil {
		h.respondDesktopStateError(c, err, "")
		return
	}
	entries, convertErr := desktopStateEntriesFromEngine(subscription.Snapshot())
	closeErr := subscription.Close()
	if err := errors.Join(convertErr, closeErr); err != nil {
		h.respondDesktopStateError(c, err, "")
		return
	}
	if subscription.AsOfSeq() > desktopStateMaxSafeID {
		h.respondDesktopStateError(c, errDesktopStateUnsafeNumber, "")
		return
	}
	c.JSON(http.StatusOK, contract.DesktopStateListResponse{
		AsOfSeq: contract.DesktopStateSafeNumber(subscription.AsOfSeq()),
		Entries: entries,
	})
}

// GetDesktopState returns one live workspace desktop-state entry.
func (h *BaseHandlers) GetDesktopState(c *gin.Context) {
	key := strings.TrimSpace(c.Param("key"))
	if h.DesktopState == nil {
		h.respondDesktopStateError(c, clientstate.ErrClosed, key)
		return
	}
	entry, err := h.DesktopState.Get(
		c.Request.Context(),
		desktopStateWorkspace(c),
		desktopStateDomain,
		key,
	)
	if err != nil {
		h.respondDesktopStateError(c, err, key)
		return
	}
	payload, err := desktopStateEntryFromEngine(entry)
	if err != nil {
		h.respondDesktopStateError(c, err, key)
		return
	}
	c.JSON(http.StatusOK, payload)
}

// PutDesktopState creates or replaces one workspace desktop-state entry.
func (h *BaseHandlers) PutDesktopState(c *gin.Context) {
	key := strings.TrimSpace(c.Param("key"))
	if h.DesktopState == nil {
		h.respondDesktopStateError(c, clientstate.ErrClosed, key)
		return
	}
	var request contract.DesktopStatePutRequest
	if err := decodeDesktopStateJSON(c, &request); err != nil {
		h.respondDesktopStateError(c, fmt.Errorf("%w: %w", clientstate.ErrInvalidValue, err), key)
		return
	}
	if err := validateDesktopStateRevision(request.IfRev); err != nil {
		h.respondDesktopStateError(c, err, key)
		return
	}
	value, err := json.Marshal(request.Value)
	if err != nil {
		h.respondDesktopStateError(c, fmt.Errorf("%w: encode value: %w", clientstate.ErrInvalidValue, err), key)
		return
	}
	entries, err := h.DesktopState.Apply(
		c.Request.Context(),
		desktopStateWorkspace(c),
		desktopStateDomain,
		[]clientstate.Op{{
			Kind: clientstate.OpPut, Key: key, Value: value, IfRev: revisionValue(request.IfRev),
		}},
		clientstate.ApplyOptions{},
	)
	if err != nil {
		h.respondDesktopStateError(c, err, key)
		return
	}
	if len(entries) != 1 {
		h.respondDesktopStateError(c, unexpectedDesktopStateResultCount(1, len(entries)), key)
		return
	}
	payload, err := desktopStateEntryFromEngine(entries[0])
	if err != nil {
		h.respondDesktopStateError(c, err, key)
		return
	}
	c.JSON(http.StatusOK, payload)
}

// ApplyDesktopState atomically commits a desktop-state mutation batch.
func (h *BaseHandlers) ApplyDesktopState(c *gin.Context) {
	if h.DesktopState == nil {
		h.respondDesktopStateError(c, clientstate.ErrClosed, "")
		return
	}
	var request contract.DesktopStateApplyRequest
	if err := decodeDesktopStateJSON(c, &request); err != nil {
		h.respondDesktopStateError(c, fmt.Errorf("%w: %w", clientstate.ErrInvalidValue, err), "")
		return
	}
	ops, err := desktopStateOpsFromContract(request.Ops)
	if err != nil {
		h.respondDesktopStateError(c, err, desktopStateErrorKey(request.Ops))
		return
	}
	entries, err := h.DesktopState.Apply(
		c.Request.Context(),
		desktopStateWorkspace(c),
		desktopStateDomain,
		ops,
		clientstate.ApplyOptions{},
	)
	if err != nil {
		h.respondDesktopStateError(c, err, desktopStateErrorKey(request.Ops))
		return
	}
	if len(entries) != len(request.Ops) {
		h.respondDesktopStateError(c, unexpectedDesktopStateResultCount(len(request.Ops), len(entries)), "")
		return
	}
	results, err := desktopStateEntriesFromEngine(entries)
	if err != nil {
		h.respondDesktopStateError(c, err, "")
		return
	}
	c.JSON(http.StatusOK, contract.DesktopStateApplyResponse{Results: results})
}

// DeleteDesktopState writes a tombstone for one live workspace desktop-state entry.
func (h *BaseHandlers) DeleteDesktopState(c *gin.Context) {
	key := strings.TrimSpace(c.Param("key"))
	if h.DesktopState == nil {
		h.respondDesktopStateError(c, clientstate.ErrClosed, key)
		return
	}
	ifRev, err := desktopStateRevisionQuery(c.Query("if_rev"))
	if err != nil {
		h.respondDesktopStateError(c, err, key)
		return
	}
	entries, err := h.DesktopState.Apply(
		c.Request.Context(),
		desktopStateWorkspace(c),
		desktopStateDomain,
		[]clientstate.Op{{Kind: clientstate.OpDelete, Key: key, IfRev: ifRev}},
		clientstate.ApplyOptions{},
	)
	if err != nil {
		h.respondDesktopStateError(c, err, key)
		return
	}
	if len(entries) != 1 {
		h.respondDesktopStateError(c, unexpectedDesktopStateResultCount(1, len(entries)), key)
		return
	}
	c.Status(http.StatusNoContent)
}

func unexpectedDesktopStateResultCount(want, got int) error {
	return fmt.Errorf("desktop state service returned %d results, want %d", got, want)
}

func desktopStateEntryFromEngine(entry clientstate.Entry) (contract.DesktopStateEntry, error) {
	if entry.Rev > desktopStateMaxSafeID || entry.Seq > desktopStateMaxSafeID {
		return contract.DesktopStateEntry{}, errDesktopStateUnsafeNumber
	}
	var value map[string]any
	if !entry.Deleted {
		if err := json.Unmarshal(entry.Value, &value); err != nil || value == nil {
			return contract.DesktopStateEntry{}, fmt.Errorf(
				"desktop state value is not an object: %w",
				clientstate.ErrInvalidValue,
			)
		}
	}
	return contract.DesktopStateEntry{
		Key: entry.Key, Value: value,
		Rev: contract.DesktopStateSafeNumber(entry.Rev), Seq: contract.DesktopStateSafeNumber(entry.Seq),
		Deleted: entry.Deleted, UpdatedAt: entry.UpdatedAt.UTC(),
	}, nil
}

func desktopStateEntriesFromEngine(entries []clientstate.Entry) ([]contract.DesktopStateEntry, error) {
	result := make([]contract.DesktopStateEntry, 0, len(entries))
	for _, entry := range entries {
		payload, err := desktopStateEntryFromEngine(entry)
		if err != nil {
			return nil, err
		}
		result = append(result, payload)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result, nil
}

func desktopStateOpsFromContract(request []contract.DesktopStateApplyOp) ([]clientstate.Op, error) {
	ops := make([]clientstate.Op, 0, len(request))
	for _, operation := range request {
		if err := validateDesktopStateRevision(operation.IfRev); err != nil {
			return nil, err
		}
		op := clientstate.Op{Key: operation.Key, IfRev: revisionValue(operation.IfRev)}
		switch operation.Kind {
		case contract.DesktopStateOpPut:
			if operation.Value == nil {
				return nil, clientstate.ErrInvalidValue
			}
			value, err := json.Marshal(*operation.Value)
			if err != nil {
				return nil, fmt.Errorf("%w: encode value: %w", clientstate.ErrInvalidValue, err)
			}
			op.Kind = clientstate.OpPut
			op.Value = value
		case contract.DesktopStateOpDelete:
			if operation.Value != nil {
				return nil, clientstate.ErrInvalidValue
			}
			op.Kind = clientstate.OpDelete
		default:
			return nil, clientstate.ErrInvalidValue
		}
		ops = append(ops, op)
	}
	return ops, nil
}

func decodeDesktopStateJSON(c *gin.Context, target any) error {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func desktopStateRevisionQuery(value string) (uint64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, nil
	}
	revision, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil || revision > desktopStateMaxSafeID {
		return 0, clientstate.ErrInvalidValue
	}
	return revision, nil
}

func validateDesktopStateRevision(value *contract.DesktopStateSafeNumber) error {
	if value != nil && uint64(*value) > desktopStateMaxSafeID {
		return clientstate.ErrInvalidValue
	}
	return nil
}

func revisionValue(value *contract.DesktopStateSafeNumber) uint64 {
	if value == nil {
		return 0
	}
	return uint64(*value)
}

func desktopStateWorkspace(c *gin.Context) clientstate.WorkspaceID {
	return clientstate.WorkspaceID(strings.TrimSpace(c.Param("workspace_id")))
}

func desktopStateErrorKey(ops []contract.DesktopStateApplyOp) string {
	if len(ops) == 1 {
		return strings.TrimSpace(ops[0].Key)
	}
	return ""
}
