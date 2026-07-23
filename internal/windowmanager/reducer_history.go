package windowmanager

import "fmt"

func (r *reducer) undo(snapshot *Snapshot) (bool, error) {
	if len(snapshot.History.Undo) == 0 {
		return false, ErrHistoryBoundary
	}
	index := len(snapshot.History.Undo) - 1
	entry := cloneHistoryEntry(snapshot.History.Undo[index])
	snapshot.History.Undo = snapshot.History.Undo[:index]
	restoreStatePreservingRoutes(snapshot, entry.Before)
	snapshot.History.Redo = append(snapshot.History.Redo, entry)
	r.markState(snapshot)
	return true, nil
}

func (r *reducer) redo(snapshot *Snapshot) (bool, error) {
	if len(snapshot.History.Redo) == 0 {
		return false, ErrHistoryBoundary
	}
	index := len(snapshot.History.Redo) - 1
	entry := cloneHistoryEntry(snapshot.History.Redo[index])
	snapshot.History.Redo = snapshot.History.Redo[:index]
	restoreStatePreservingRoutes(snapshot, entry.After)
	snapshot.History.Undo = append(snapshot.History.Undo, entry)
	r.markState(snapshot)
	return true, nil
}

func (r *reducer) replace(snapshot *Snapshot, command ReplaceLayoutCommand) (bool, error) {
	document := command.Document
	if document.Version != SnapshotVersion || document.WorkspaceID != snapshot.WorkspaceID {
		return false, fmt.Errorf("layout document identity: %w", ErrInvalidTopology)
	}
	candidate := Snapshot{
		Version:     document.Version,
		WorkspaceID: document.WorkspaceID,
		Revision:    snapshot.Revision,
		Desktops:    cloneDesktops(document.Desktops),
		Windows:     cloneWindows(document.Windows),
		History:     cloneHistory(snapshot.History),
		Overrides:   cloneWorkspaceConfig(document.Overrides),
		UpdatedAt:   snapshot.UpdatedAt,
	}
	for windowID, current := range snapshot.Windows {
		window, exists := candidate.Windows[windowID]
		if !exists {
			continue
		}
		window.Route = cloneRouteIntent(current.Route)
		candidate.Windows[windowID] = window
	}
	if err := ValidateSnapshot(candidate); err != nil {
		return false, err
	}
	equal, err := statesEqual(snapshotState(*snapshot), snapshotState(candidate))
	if err != nil {
		return false, err
	}
	if equal {
		return false, nil
	}
	restoreStatePreservingRoutes(snapshot, snapshotState(candidate))
	r.markState(snapshot)
	return true, nil
}

func (r *reducer) markState(snapshot *Snapshot) {
	for _, desktop := range snapshot.Desktops {
		r.changes.desktop(desktop.ID)
		for _, group := range desktop.Groups {
			r.changes.group(group.ID)
			markNodeChanges(group.Root, &r.changes)
		}
	}
	for windowID := range snapshot.Windows {
		r.changes.window(windowID)
	}
}

func markNodeChanges(node LayoutNode, changes *changeBuilder) {
	changes.node(node.ID)
	for _, child := range node.Children {
		markNodeChanges(child, changes)
	}
}
