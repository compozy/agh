package windowmanager

import (
	"fmt"
	"strings"
)

func (r *reducer) createDesktop(snapshot *Snapshot, command CreateDesktopCommand) (bool, error) {
	desktopID := command.DesktopID
	if desktopID == "" {
		generated, err := r.generate("desktop")
		if err != nil {
			return false, fmt.Errorf("generate desktop ID: %w", err)
		}
		desktopID = DesktopID(generated)
	}
	if _, exists := desktopIndexByID(snapshot, desktopID); exists {
		return false, fmt.Errorf("desktop %q already exists: %w", desktopID, ErrInvalidCommand)
	}
	purpose := command.Purpose
	if purpose == "" {
		purpose = DesktopPurposeStandard
	}
	if purpose != DesktopPurposeStandard && purpose != DesktopPurposeFocus {
		return false, fmt.Errorf("desktop purpose %q: %w", purpose, ErrInvalidCommand)
	}
	name := strings.TrimSpace(command.Name)
	if name == "" {
		name = fmt.Sprintf("Desktop %d", len(snapshot.Desktops)+1)
	}
	desktop := Desktop{
		ID:         desktopID,
		Name:       name,
		Order:      len(snapshot.Desktops),
		Purpose:    purpose,
		FocusOwner: clonePointer(command.FocusOwner),
		Groups:     []LayoutGroup{},
		Floating:   []WindowID{},
	}
	insertIndex := len(snapshot.Desktops)
	if command.AfterID != nil {
		index, exists := desktopIndexByID(snapshot, *command.AfterID)
		if !exists {
			return false, fmt.Errorf("desktop %q: %w", *command.AfterID, ErrDesktopNotFound)
		}
		insertIndex = index + 1
	}
	snapshot.Desktops = append(snapshot.Desktops, Desktop{})
	copy(snapshot.Desktops[insertIndex+1:], snapshot.Desktops[insertIndex:])
	snapshot.Desktops[insertIndex] = desktop
	setDesktopOrders(snapshot)
	r.changes.desktop(desktopID)
	return true, nil
}

func (r *reducer) updateDesktop(snapshot *Snapshot, command UpdateDesktopCommand) (bool, error) {
	index, exists := desktopIndexByID(snapshot, command.DesktopID)
	if !exists {
		return false, fmt.Errorf("desktop %q: %w", command.DesktopID, ErrDesktopNotFound)
	}
	name := strings.TrimSpace(command.Name)
	if name == "" {
		return false, fmt.Errorf("desktop name is required: %w", ErrInvalidCommand)
	}
	if snapshot.Desktops[index].Name == name {
		return false, nil
	}
	snapshot.Desktops[index].Name = name
	r.changes.desktop(command.DesktopID)
	return true, nil
}

func (r *reducer) reorderDesktop(snapshot *Snapshot, command ReorderDesktopCommand) (bool, error) {
	index, exists := desktopIndexByID(snapshot, command.DesktopID)
	if !exists {
		return false, fmt.Errorf("desktop %q: %w", command.DesktopID, ErrDesktopNotFound)
	}
	if command.Order < 0 || command.Order >= len(snapshot.Desktops) {
		return false, fmt.Errorf("desktop order %d: %w", command.Order, ErrInvalidCommand)
	}
	if index == command.Order {
		return false, nil
	}
	desktop := snapshot.Desktops[index]
	snapshot.Desktops = append(snapshot.Desktops[:index], snapshot.Desktops[index+1:]...)
	snapshot.Desktops = append(snapshot.Desktops, Desktop{})
	copy(snapshot.Desktops[command.Order+1:], snapshot.Desktops[command.Order:])
	snapshot.Desktops[command.Order] = desktop
	setDesktopOrders(snapshot)
	r.changes.desktop(command.DesktopID)
	return true, nil
}

func (r *reducer) deleteDesktop(snapshot *Snapshot, command DeleteDesktopCommand) (bool, error) {
	index, exists := desktopIndexByID(snapshot, command.DesktopID)
	if !exists {
		return false, fmt.Errorf("desktop %q: %w", command.DesktopID, ErrDesktopNotFound)
	}
	if len(snapshot.Desktops) == 1 {
		return false, ErrFinalDesktop
	}
	source := snapshot.Desktops[index]
	nonEmpty := len(source.Groups) > 0 || len(source.Floating) > 0
	if nonEmpty && command.DestinationID == nil {
		return false, ErrDestinationRequired
	}
	if command.DestinationID != nil && *command.DestinationID == command.DesktopID {
		return false, fmt.Errorf("destination must differ from source: %w", ErrDestinationRequired)
	}
	if command.DestinationID != nil {
		destinationIndex, found := desktopIndexByID(snapshot, *command.DestinationID)
		if !found {
			return false, fmt.Errorf("desktop %q: %w", *command.DestinationID, ErrDesktopNotFound)
		}
		destination := &snapshot.Desktops[destinationIndex]
		transferredGroups := cloneDesktop(source).Groups
		destination.Groups = append(destination.Groups, transferredGroups...)
		for _, group := range transferredGroups {
			r.changes.group(group.ID)
		}
		if layoutGroupsOverlap(destination.Groups) {
			reflowLayoutGroupFrames(destination.Groups)
			for _, group := range destination.Groups {
				r.changes.group(group.ID)
			}
		}
		destination.Floating = append(destination.Floating, source.Floating...)
		for windowID, window := range snapshot.Windows {
			if window.DesktopID != source.ID {
				continue
			}
			window.DesktopID = destination.ID
			snapshot.Windows[windowID] = window
			r.changes.window(windowID)
		}
		r.changes.desktop(destination.ID)
	}
	snapshot.Desktops = append(snapshot.Desktops[:index], snapshot.Desktops[index+1:]...)
	setDesktopOrders(snapshot)
	r.changes.desktop(source.ID)
	return true, nil
}

func layoutGroupsOverlap(groups []LayoutGroup) bool {
	for left := range groups {
		for right := left + 1; right < len(groups); right++ {
			if rectsOverlap(groups[left].Frame, groups[right].Frame) {
				return true
			}
		}
	}
	return false
}

func reflowLayoutGroupFrames(groups []LayoutGroup) {
	if len(groups) == 0 {
		return
	}
	width := 1 / float64(len(groups))
	for index := range groups {
		groups[index].Frame = NormalizedRect{
			X:      float64(index) * width,
			Width:  width,
			Height: 1,
		}
	}
}

func setDesktopOrders(snapshot *Snapshot) {
	for index := range snapshot.Desktops {
		snapshot.Desktops[index].Order = index
	}
}
