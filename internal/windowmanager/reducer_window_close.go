package windowmanager

import (
	"fmt"
	"slices"
)

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
