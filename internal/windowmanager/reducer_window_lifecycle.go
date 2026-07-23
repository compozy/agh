package windowmanager

import (
	"fmt"
	"slices"
	"strings"
)

func (r *reducer) openWindow(snapshot *Snapshot, command OpenWindowCommand) (bool, error) {
	if command.RestoreWindowID != nil {
		return r.restoreWindow(snapshot, *command.RestoreWindowID)
	}
	window, err := r.newOpenWindow(snapshot, command.Window)
	if err != nil {
		return false, err
	}
	windowID := window.ID
	desktopID := window.DesktopID
	snapshot.Windows[windowID] = window
	insertTiled := command.Window.InsertTiled || r.config.NewWindowPolicy == NewWindowInsert
	if insertTiled && r.focusedWindow != nil {
		focused, exists := snapshot.Windows[*r.focusedWindow]
		if exists && focused.DesktopID == desktopID && focused.Placement != WindowPlacementFloating {
			if err := insertRelative(snapshot, focused.ID, windowID, DropAfter, r.generate); err != nil {
				delete(snapshot.Windows, windowID)
				return false, err
			}
			r.changes.window(focused.ID)
		} else {
			insertTiled = false
		}
	} else if insertTiled {
		desktopIndex, exists := desktopIndexByID(snapshot, desktopID)
		if !exists {
			delete(snapshot.Windows, windowID)
			return false, fmt.Errorf("desktop %q: %w", desktopID, ErrDesktopNotFound)
		}
		if len(snapshot.Desktops[desktopIndex].Groups) == 0 {
			leaf, leafErr := newLeaf(windowID, r.generate)
			if leafErr != nil {
				delete(snapshot.Windows, windowID)
				return false, leafErr
			}
			groupID, generateErr := r.generate("group")
			if generateErr != nil {
				delete(snapshot.Windows, windowID)
				return false, fmt.Errorf("generate group ID: %w", generateErr)
			}
			snapshot.Desktops[desktopIndex].Groups = append(
				snapshot.Desktops[desktopIndex].Groups,
				LayoutGroup{ID: GroupID(groupID), Frame: fullRect(), Root: leaf},
			)
			r.changes.group(GroupID(groupID))
			r.changes.node(leaf.ID)
		} else {
			insertTiled = false
		}
	}
	if !insertTiled {
		desktopIndex, _ := desktopIndexByID(snapshot, desktopID)
		snapshot.Desktops[desktopIndex].Floating = append(snapshot.Desktops[desktopIndex].Floating, windowID)
	}
	r.changes.window(windowID)
	r.changes.desktop(desktopID)
	return true, nil
}

func (r *reducer) newOpenWindow(snapshot *Snapshot, spec WindowSpec) (Window, error) {
	windowID := spec.ID
	if windowID == "" {
		generated, err := r.generate("window")
		if err != nil {
			return Window{}, fmt.Errorf("generate window ID: %w", err)
		}
		windowID = WindowID(generated)
	}
	if _, exists := snapshot.Windows[windowID]; exists {
		return Window{}, fmt.Errorf("window %q already exists: %w", windowID, ErrInvalidCommand)
	}
	app := strings.TrimSpace(spec.App)
	if app == "" {
		return Window{}, fmt.Errorf("window app is required: %w", ErrInvalidCommand)
	}
	desktopID, err := r.resolveOpenDesktop(snapshot, spec.DesktopID)
	if err != nil {
		return Window{}, err
	}
	route, err := CanonicalRouteIntent(spec.Route)
	if err != nil {
		return Window{}, fmt.Errorf("window route: %w: %w", err, ErrInvalidCommand)
	}
	return Window{
		ID:           windowID,
		App:          app,
		InstanceKey:  clonePointer(spec.InstanceKey),
		Route:        route,
		DesktopID:    desktopID,
		Placement:    WindowPlacementFloating,
		FloatingRect: clampRect(spec.FloatingRect),
	}, nil
}

func (r *reducer) restoreWindow(snapshot *Snapshot, windowID WindowID) (bool, error) {
	window, exists := snapshot.Windows[windowID]
	if !exists {
		return false, fmt.Errorf("window %q: %w", windowID, ErrWindowNotFound)
	}
	if !window.Minimized {
		return false, nil
	}
	removeWindow(snapshot, windowID)
	window.Minimized = false
	if window.ReturnAnchor != nil {
		if _, exists := desktopIndexByID(snapshot, window.ReturnAnchor.DesktopID); exists {
			window.DesktopID = window.ReturnAnchor.DesktopID
		}
	}
	snapshot.Windows[windowID] = window
	if err := insertAtAnchor(snapshot, windowID, window.ReturnAnchor, r.focusedWindow, r.generate); err != nil {
		return false, err
	}
	window = snapshot.Windows[windowID]
	window.ReturnAnchor = nil
	snapshot.Windows[windowID] = window
	r.changes.window(windowID)
	r.changes.desktop(window.DesktopID)
	return true, nil
}

func (r *reducer) closeWindow(snapshot *Snapshot, command CloseWindowCommand) (bool, error) {
	window, exists := snapshot.Windows[command.WindowID]
	if !exists {
		return false, fmt.Errorf("window %q: %w", command.WindowID, ErrWindowNotFound)
	}
	if command.Minimize && window.Minimized {
		return false, nil
	}
	anchor := captureReturnAnchor(snapshot, command.WindowID)
	if !removeWindow(snapshot, command.WindowID) {
		return false, fmt.Errorf("window %q has no placement: %w", command.WindowID, ErrInvalidTopology)
	}
	if command.Minimize {
		window.Minimized = true
		window.Placement = WindowPlacementFloating
		window.ReturnAnchor = anchor
		window.FloatingRect = clampRect(window.FloatingRect)
		snapshot.Windows[command.WindowID] = window
		desktopIndex, _ := desktopIndexByID(snapshot, window.DesktopID)
		snapshot.Desktops[desktopIndex].Floating = append(snapshot.Desktops[desktopIndex].Floating, command.WindowID)
	} else {
		delete(snapshot.Windows, command.WindowID)
		r.removeClosedFocusDesktop(snapshot, command.WindowID)
	}
	r.changes.window(command.WindowID)
	r.changes.desktop(window.DesktopID)
	return true, nil
}

func (r *reducer) resolveOpenDesktop(snapshot *Snapshot, requested DesktopID) (DesktopID, error) {
	if requested != "" {
		if _, exists := desktopIndexByID(snapshot, requested); !exists {
			return "", fmt.Errorf("desktop %q: %w", requested, ErrDesktopNotFound)
		}
		return requested, nil
	}
	if r.focusedWindow != nil {
		if focused, exists := snapshot.Windows[*r.focusedWindow]; exists {
			return focused.DesktopID, nil
		}
	}
	for _, desktop := range snapshot.Desktops {
		if desktop.Purpose == DesktopPurposeStandard {
			return desktop.ID, nil
		}
	}
	if len(snapshot.Desktops) == 0 {
		return "", ErrFinalDesktop
	}
	return snapshot.Desktops[0].ID, nil
}

func (r *reducer) removeClosedFocusDesktop(snapshot *Snapshot, windowID WindowID) {
	for index := range slices.Backward(snapshot.Desktops) {
		desktop := snapshot.Desktops[index]
		if desktop.Purpose != DesktopPurposeFocus || desktop.FocusOwner == nil || *desktop.FocusOwner != windowID {
			continue
		}
		if len(snapshot.Desktops) == 1 {
			snapshot.Desktops[index].Purpose = DesktopPurposeStandard
			snapshot.Desktops[index].FocusOwner = nil
		} else {
			snapshot.Desktops = append(snapshot.Desktops[:index], snapshot.Desktops[index+1:]...)
		}
		r.changes.desktop(desktop.ID)
	}
}
