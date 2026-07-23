package contract

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/compozy/agh/internal/windowmanager"
)

// WindowManagerRevision is an unsigned wire integer that remains exact in JavaScript.
type WindowManagerRevision uint64

// WindowManagerMaxSafeRevision is the largest integer exactly representable by JavaScript numbers.
const WindowManagerMaxSafeRevision WindowManagerRevision = WindowManagerRevision(windowmanager.MaxWireRevision)

// ErrWindowManagerUnsafeRevision identifies a domain revision that cannot be represented on the wire.
var ErrWindowManagerUnsafeRevision = errors.New("window manager revision exceeds the JSON safe integer range")

// UnmarshalJSON rejects revision values that JavaScript clients cannot represent exactly.
func (revision *WindowManagerRevision) UnmarshalJSON(data []byte) error {
	var raw uint64
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("decode window manager revision: %w", err)
	}
	if raw > uint64(WindowManagerMaxSafeRevision) {
		return ErrWindowManagerUnsafeRevision
	}
	*revision = WindowManagerRevision(raw)
	return nil
}

// WindowManagerErrorCode is a stable failure identifier shared by every transport.
type WindowManagerErrorCode string

const (
	WindowManagerErrorWorkspaceNotFound WindowManagerErrorCode = "window_manager_workspace_not_found"
	WindowManagerErrorRevisionConflict  WindowManagerErrorCode = "window_manager_revision_conflict"
	WindowManagerErrorInvalidCommand    WindowManagerErrorCode = "window_manager_invalid_command"
	WindowManagerErrorInvalidTopology   WindowManagerErrorCode = "window_manager_invalid_topology"
	WindowManagerErrorDesktopNotFound   WindowManagerErrorCode = "window_manager_desktop_not_found"
	WindowManagerErrorWindowNotFound    WindowManagerErrorCode = "window_manager_window_not_found"
	WindowManagerErrorClientNotFound    WindowManagerErrorCode = "window_manager_client_not_found"
	WindowManagerErrorDestination       WindowManagerErrorCode = "window_manager_destination_required"
	WindowManagerErrorFinalDesktop      WindowManagerErrorCode = "window_manager_final_desktop"
	WindowManagerErrorHistoryBoundary   WindowManagerErrorCode = "window_manager_history_boundary"
	WindowManagerErrorLayoutNotFound    WindowManagerErrorCode = "window_manager_layout_not_found"
	WindowManagerErrorSlowConsumer      WindowManagerErrorCode = "window_manager_slow_consumer"
	WindowManagerErrorUnavailable       WindowManagerErrorCode = "window_manager_unavailable"
)

// WindowManagerErrorCodeValues returns every public window-manager error code.
func WindowManagerErrorCodeValues() []string {
	return []string{
		string(WindowManagerErrorWorkspaceNotFound),
		string(WindowManagerErrorRevisionConflict),
		string(WindowManagerErrorInvalidCommand),
		string(WindowManagerErrorInvalidTopology),
		string(WindowManagerErrorDesktopNotFound),
		string(WindowManagerErrorWindowNotFound),
		string(WindowManagerErrorClientNotFound),
		string(WindowManagerErrorDestination),
		string(WindowManagerErrorFinalDesktop),
		string(WindowManagerErrorHistoryBoundary),
		string(WindowManagerErrorLayoutNotFound),
		string(WindowManagerErrorSlowConsumer),
		string(WindowManagerErrorUnavailable),
	}
}

// WindowManagerErrorPayload is the deterministic error body shared by HTTP, UDS, CLI, and WebSocket.
type WindowManagerErrorPayload struct {
	Error           string                     `json:"error"`
	Code            WindowManagerErrorCode     `json:"code"`
	WorkspaceID     windowmanager.WorkspaceID  `json:"workspace_id"`
	CurrentRevision *WindowManagerRevision     `json:"current_revision,omitempty"`
	Conflicts       []windowmanager.Conflict   `json:"conflicts,omitempty"`
	Diagnostics     []windowmanager.Diagnostic `json:"diagnostics,omitempty"`
}

// WindowManagerActor records a command initiator without affecting command semantics.
type WindowManagerActor struct {
	Kind string `json:"kind,omitempty"`
	ID   string `json:"id,omitempty"`
}

// WindowManagerReturnAnchor is the stable wire form of a prior structural slot.
type WindowManagerReturnAnchor struct {
	DesktopID      windowmanager.DesktopID   `json:"desktop_id"`
	GroupID        *windowmanager.GroupID    `json:"group_id,omitempty"`
	ParentSplitID  *windowmanager.NodeID     `json:"parent_split_id,omitempty"`
	Axis           *windowmanager.Axis       `json:"axis,omitempty"`
	ChildIndex     *int                      `json:"child_index,omitempty"`
	Weight         *float64                  `json:"weight,omitempty"`
	NeighborIDs    []windowmanager.WindowID  `json:"neighbor_ids,omitempty"`
	SourceRevision WindowManagerRevision     `json:"source_revision"`
	SourceGroup    *WindowManagerLayoutGroup `json:"source_group,omitempty"`
}

// WindowManagerWindow is one durable application instance.
type WindowManagerWindow struct {
	ID           windowmanager.WindowID        `json:"id"`
	App          string                        `json:"app"`
	InstanceKey  *string                       `json:"instance_key,omitempty"`
	Route        windowmanager.RouteIntent     `json:"route"`
	Placement    windowmanager.WindowPlacement `json:"placement"`
	DesktopID    windowmanager.DesktopID       `json:"desktop_id"`
	FloatingRect windowmanager.NormalizedRect  `json:"floating_rect"`
	Minimized    bool                          `json:"minimized"`
	ReturnAnchor *WindowManagerReturnAnchor    `json:"return_anchor,omitempty"`
}

// WindowManagerState is the revision-independent content restored by history.
type WindowManagerState struct {
	Desktops  []WindowManagerDesktop         `json:"desktops"`
	Windows   map[string]WindowManagerWindow `json:"windows"`
	Overrides windowmanager.WorkspaceConfig  `json:"overrides"`
}

// WindowManagerHistoryEntry is one exact semantic operation boundary.
type WindowManagerHistoryEntry struct {
	CommandID WindowManagerCommandID `json:"command_id"`
	Before    WindowManagerState     `json:"before"`
	After     WindowManagerState     `json:"after"`
	Actor     WindowManagerActor     `json:"actor"`
	Origin    string                 `json:"origin,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// WindowManagerHistory contains bounded undo and redo stacks.
type WindowManagerHistory struct {
	Undo []WindowManagerHistoryEntry `json:"undo"`
	Redo []WindowManagerHistoryEntry `json:"redo"`
}

// WindowManagerSnapshot is the complete durable workspace aggregate.
type WindowManagerSnapshot struct {
	Version     uint32                         `json:"version"`
	WorkspaceID windowmanager.WorkspaceID      `json:"workspace_id"`
	Revision    WindowManagerRevision          `json:"revision"`
	Desktops    []WindowManagerDesktop         `json:"desktops"`
	Windows     map[string]WindowManagerWindow `json:"windows"`
	History     WindowManagerHistory           `json:"history"`
	Overrides   windowmanager.WorkspaceConfig  `json:"overrides"`
	UpdatedAt   time.Time                      `json:"updated_at"`
}

// WindowManagerLayoutDocument is the stable declarative layout wire contract.
type WindowManagerLayoutDocument struct {
	Version     uint32                         `json:"version"`
	WorkspaceID windowmanager.WorkspaceID      `json:"workspace_id"`
	Desktops    []WindowManagerDesktop         `json:"desktops"`
	Windows     map[string]WindowManagerWindow `json:"windows"`
	Overrides   windowmanager.WorkspaceConfig  `json:"overrides"`
}

// WindowManagerClientView is transient presentation state for one connected client.
type WindowManagerClientView struct {
	WorkspaceID          windowmanager.WorkspaceID `json:"workspace_id"`
	ClientID             windowmanager.ClientID    `json:"client_id"`
	PresentationRevision WindowManagerRevision     `json:"presentation_revision"`
	ActiveDesktopID      windowmanager.DesktopID   `json:"active_desktop_id"`
	FocusedWindowID      *windowmanager.WindowID   `json:"focused_window_id,omitempty"`
	FocusOrder           []windowmanager.WindowID  `json:"focus_order"`
	ConnectedAt          time.Time                 `json:"connected_at"`
}

// WindowManagerClientsResponse lists client-local views in one workspace partition.
type WindowManagerClientsResponse struct {
	WorkspaceID windowmanager.WorkspaceID `json:"workspace_id"`
	Clients     []WindowManagerClientView `json:"clients"`
}

// WindowManagerClientRegistration requests one live client view.
type WindowManagerClientRegistration struct {
	WorkspaceID     windowmanager.WorkspaceID `json:"workspace_id"`
	ClientID        windowmanager.ClientID    `json:"client_id,omitempty"`
	ActiveDesktopID windowmanager.DesktopID   `json:"active_desktop_id,omitempty"`
}

// WindowManagerLayoutValidationRequest validates a declarative document without writing it.
type WindowManagerLayoutValidationRequest struct {
	WorkspaceID windowmanager.WorkspaceID   `json:"workspace_id"`
	Document    WindowManagerLayoutDocument `json:"document"`
}

// WindowManagerLayoutValidationResponse returns stable topology diagnostics.
type WindowManagerLayoutValidationResponse struct {
	WorkspaceID windowmanager.WorkspaceID  `json:"workspace_id"`
	Valid       bool                       `json:"valid"`
	Diagnostics []windowmanager.Diagnostic `json:"diagnostics,omitempty"`
}

// WindowManagerLayoutReplaceRequest applies a declarative document through revision CAS.
type WindowManagerLayoutReplaceRequest struct {
	WorkspaceID      windowmanager.WorkspaceID   `json:"workspace_id"`
	ExpectedRevision *WindowManagerRevision      `json:"expected_revision"`
	ClientID         *windowmanager.ClientID     `json:"client_id,omitempty"`
	Actor            WindowManagerActor          `json:"actor"`
	Origin           string                      `json:"origin"`
	Document         WindowManagerLayoutDocument `json:"document"`
}

// WindowManagerResult reports one applied or no-op command.
type WindowManagerResult struct {
	Snapshot    WindowManagerSnapshot      `json:"snapshot"`
	Applied     bool                       `json:"applied"`
	Changes     windowmanager.ChangeSet    `json:"changes"`
	Diagnostics []windowmanager.Diagnostic `json:"diagnostics,omitempty"`
	Client      *WindowManagerClientView   `json:"client,omitempty"`
	RebasedFrom *WindowManagerRevision     `json:"rebased_from,omitempty"`
}

// WindowManagerPreview reports a validated proposal without a durable write.
type WindowManagerPreview struct {
	Snapshot    WindowManagerSnapshot      `json:"snapshot"`
	Changed     bool                       `json:"changed"`
	Changes     windowmanager.ChangeSet    `json:"changes"`
	Diagnostics []windowmanager.Diagnostic `json:"diagnostics,omitempty"`
	Client      *WindowManagerClientView   `json:"client,omitempty"`
}

// WindowManagerSnapshotFromDomain converts the authoritative domain snapshot to its stable wire form.
func WindowManagerSnapshotFromDomain(snapshot windowmanager.Snapshot) (WindowManagerSnapshot, error) {
	revision, err := windowManagerRevision(snapshot.Revision)
	if err != nil {
		return WindowManagerSnapshot{}, err
	}
	windows, err := windowManagerWindowsFromDomain(snapshot.Windows)
	if err != nil {
		return WindowManagerSnapshot{}, err
	}
	history, err := windowManagerHistoryFromDomain(snapshot.History)
	if err != nil {
		return WindowManagerSnapshot{}, err
	}
	return WindowManagerSnapshot{
		Version: snapshot.Version, WorkspaceID: snapshot.WorkspaceID, Revision: revision,
		Desktops: windowManagerDesktopsFromDomain(snapshot.Desktops), Windows: windows, History: history,
		Overrides: snapshot.Overrides, UpdatedAt: snapshot.UpdatedAt,
	}, nil
}

// WindowManagerLayoutFromDomain converts a domain layout to its stable wire form.
func WindowManagerLayoutFromDomain(document windowmanager.LayoutDocument) (WindowManagerLayoutDocument, error) {
	windows, err := windowManagerWindowsFromDomain(document.Windows)
	if err != nil {
		return WindowManagerLayoutDocument{}, err
	}
	return WindowManagerLayoutDocument{
		Version: document.Version, WorkspaceID: document.WorkspaceID,
		Desktops: windowManagerDesktopsFromDomain(document.Desktops),
		Windows:  windows, Overrides: document.Overrides,
	}, nil
}

// Domain converts a stable wire layout to the authoritative domain document.
func (document WindowManagerLayoutDocument) Domain() windowmanager.LayoutDocument {
	return windowmanager.LayoutDocument{
		Version: document.Version, WorkspaceID: document.WorkspaceID,
		Desktops: windowManagerDesktopsToDomain(document.Desktops),
		Windows:  windowManagerWindowsToDomain(document.Windows), Overrides: document.Overrides,
	}
}

// WindowManagerClientFromDomain converts transient client presentation state.
func WindowManagerClientFromDomain(client windowmanager.ClientView) (WindowManagerClientView, error) {
	revision, err := windowManagerRevision(windowmanager.Revision(client.PresentationRevision))
	if err != nil {
		return WindowManagerClientView{}, err
	}
	return WindowManagerClientView{
		WorkspaceID: client.WorkspaceID, ClientID: client.ClientID, ActiveDesktopID: client.ActiveDesktopID,
		PresentationRevision: revision, FocusedWindowID: client.FocusedWindowID,
		FocusOrder: append([]windowmanager.WindowID{}, client.FocusOrder...), ConnectedAt: client.ConnectedAt,
	}, nil
}

// WindowManagerResultFromDomain converts a semantic command result.
func WindowManagerResultFromDomain(result windowmanager.Result) (WindowManagerResult, error) {
	snapshot, err := WindowManagerSnapshotFromDomain(result.Snapshot)
	if err != nil {
		return WindowManagerResult{}, err
	}
	rebased, err := optionalWindowManagerRevision(result.RebasedFrom)
	if err != nil {
		return WindowManagerResult{}, err
	}
	var client *WindowManagerClientView
	if result.Client != nil {
		converted, err := WindowManagerClientFromDomain(*result.Client)
		if err != nil {
			return WindowManagerResult{}, fmt.Errorf("convert result client: %w", err)
		}
		client = &converted
	}
	return WindowManagerResult{
		Snapshot: snapshot, Applied: result.Applied, Changes: result.Changes,
		Diagnostics: result.Diagnostics, Client: client, RebasedFrom: rebased,
	}, nil
}

// WindowManagerPreviewFromDomain converts a command preview.
func WindowManagerPreviewFromDomain(preview windowmanager.Preview) (WindowManagerPreview, error) {
	snapshot, err := WindowManagerSnapshotFromDomain(preview.Snapshot)
	if err != nil {
		return WindowManagerPreview{}, err
	}
	var client *WindowManagerClientView
	if preview.Client != nil {
		converted, err := WindowManagerClientFromDomain(*preview.Client)
		if err != nil {
			return WindowManagerPreview{}, fmt.Errorf("convert preview client: %w", err)
		}
		client = &converted
	}
	return WindowManagerPreview{
		Snapshot: snapshot, Changed: preview.Changed, Changes: preview.Changes,
		Diagnostics: preview.Diagnostics, Client: client,
	}, nil
}

func windowManagerHistoryFromDomain(history windowmanager.History) (WindowManagerHistory, error) {
	undo, err := windowManagerHistoryEntriesFromDomain(history.Undo)
	if err != nil {
		return WindowManagerHistory{}, err
	}
	redo, err := windowManagerHistoryEntriesFromDomain(history.Redo)
	if err != nil {
		return WindowManagerHistory{}, err
	}
	return WindowManagerHistory{Undo: undo, Redo: redo}, nil
}

func windowManagerHistoryEntriesFromDomain(entries []windowmanager.HistoryEntry) ([]WindowManagerHistoryEntry, error) {
	result := make([]WindowManagerHistoryEntry, 0, len(entries))
	for _, entry := range entries {
		beforeWindows, err := windowManagerWindowsFromDomain(entry.Before.Windows)
		if err != nil {
			return nil, err
		}
		afterWindows, err := windowManagerWindowsFromDomain(entry.After.Windows)
		if err != nil {
			return nil, err
		}
		result = append(result, WindowManagerHistoryEntry{
			CommandID: WindowManagerCommandID(entry.CommandID),
			Before: WindowManagerState{
				Desktops: windowManagerDesktopsFromDomain(entry.Before.Desktops),
				Windows:  beforeWindows, Overrides: entry.Before.Overrides,
			},
			After: WindowManagerState{
				Desktops: windowManagerDesktopsFromDomain(entry.After.Desktops),
				Windows:  afterWindows, Overrides: entry.After.Overrides,
			},
			Actor:     WindowManagerActor{Kind: entry.Actor.Kind, ID: entry.Actor.ID},
			Origin:    entry.Origin,
			CreatedAt: entry.CreatedAt,
		})
	}
	return result, nil
}

func windowManagerWindowsFromDomain(
	windows map[windowmanager.WindowID]windowmanager.Window,
) (map[string]WindowManagerWindow, error) {
	result := make(map[string]WindowManagerWindow, len(windows))
	for id, window := range windows {
		converted := WindowManagerWindow{
			ID: window.ID, App: window.App, InstanceKey: window.InstanceKey,
			Route: cloneWindowManagerRoute(window.Route), Placement: window.Placement,
			DesktopID: window.DesktopID, FloatingRect: window.FloatingRect, Minimized: window.Minimized,
		}
		if window.ReturnAnchor != nil {
			revision, err := windowManagerRevision(window.ReturnAnchor.SourceRevision)
			if err != nil {
				return nil, fmt.Errorf("convert return anchor for window %q: %w", id, err)
			}
			converted.ReturnAnchor = &WindowManagerReturnAnchor{
				DesktopID: window.ReturnAnchor.DesktopID, GroupID: window.ReturnAnchor.GroupID,
				ParentSplitID: window.ReturnAnchor.ParentSplitID, Axis: window.ReturnAnchor.Axis,
				ChildIndex: window.ReturnAnchor.ChildIndex, Weight: window.ReturnAnchor.Weight,
				NeighborIDs: window.ReturnAnchor.NeighborIDs, SourceRevision: revision,
				SourceGroup: optionalWindowManagerLayoutGroupFromDomain(window.ReturnAnchor.SourceGroup),
			}
		}
		result[string(id)] = converted
	}
	return result, nil
}

func windowManagerWindowsToDomain(
	windows map[string]WindowManagerWindow,
) map[windowmanager.WindowID]windowmanager.Window {
	result := make(map[windowmanager.WindowID]windowmanager.Window, len(windows))
	for key, window := range windows {
		converted := windowmanager.Window{
			ID: window.ID, App: window.App, InstanceKey: window.InstanceKey,
			Route: cloneWindowManagerRoute(window.Route), Placement: window.Placement,
			DesktopID: window.DesktopID, FloatingRect: window.FloatingRect, Minimized: window.Minimized,
		}
		if window.ReturnAnchor != nil {
			converted.ReturnAnchor = &windowmanager.ReturnAnchor{
				DesktopID: window.ReturnAnchor.DesktopID, GroupID: window.ReturnAnchor.GroupID,
				ParentSplitID: window.ReturnAnchor.ParentSplitID, Axis: window.ReturnAnchor.Axis,
				ChildIndex: window.ReturnAnchor.ChildIndex, Weight: window.ReturnAnchor.Weight,
				NeighborIDs:    window.ReturnAnchor.NeighborIDs,
				SourceRevision: windowmanager.Revision(window.ReturnAnchor.SourceRevision),
				SourceGroup:    optionalWindowManagerLayoutGroupToDomain(window.ReturnAnchor.SourceGroup),
			}
		}
		result[windowmanager.WindowID(key)] = converted
	}
	return result
}

func cloneWindowManagerRoute(route windowmanager.RouteIntent) windowmanager.RouteIntent {
	if route.Search == nil {
		return windowmanager.RouteIntent{Pathname: route.Pathname}
	}
	search := make(windowmanager.RouteSearch, len(route.Search))
	for key, value := range route.Search {
		search[key] = append([]byte(nil), value...)
	}
	return windowmanager.RouteIntent{Pathname: route.Pathname, Search: search}
}

func windowManagerRevision(revision windowmanager.Revision) (WindowManagerRevision, error) {
	if revision > windowmanager.Revision(WindowManagerMaxSafeRevision) {
		return 0, ErrWindowManagerUnsafeRevision
	}
	return WindowManagerRevision(revision), nil
}

func optionalWindowManagerRevision(revision *windowmanager.Revision) (*WindowManagerRevision, error) {
	if revision == nil {
		return nil, nil
	}
	converted, err := windowManagerRevision(*revision)
	if err != nil {
		return nil, err
	}
	return &converted, nil
}
