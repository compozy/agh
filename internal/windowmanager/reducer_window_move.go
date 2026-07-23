package windowmanager

import "fmt"

func (r *reducer) moveWindow(snapshot *Snapshot, command MoveWindowCommand) (bool, error) {
	window, exists := snapshot.Windows[command.WindowID]
	if !exists {
		return false, fmt.Errorf("window %q: %w", command.WindowID, ErrWindowNotFound)
	}
	if _, exists := desktopIndexByID(snapshot, command.DestinationDesktopID); !exists {
		return false, fmt.Errorf("desktop %q: %w", command.DestinationDesktopID, ErrDesktopNotFound)
	}
	if command.MoveGroup {
		return r.moveWindowGroup(snapshot, command)
	}
	if command.TargetWindowID != nil {
		if *command.TargetWindowID == command.WindowID {
			return false, fmt.Errorf("window cannot target itself: %w", ErrInvalidCommand)
		}
		target, exists := snapshot.Windows[*command.TargetWindowID]
		if !exists {
			return false, fmt.Errorf("target window %q: %w", *command.TargetWindowID, ErrWindowNotFound)
		}
		if target.DesktopID != command.DestinationDesktopID {
			return false, fmt.Errorf("target window belongs to another desktop: %w", ErrInvalidCommand)
		}
	}
	anchor := captureReturnAnchor(snapshot, command.WindowID)
	if !removeWindow(snapshot, command.WindowID) {
		return false, fmt.Errorf("window %q has no placement: %w", command.WindowID, ErrInvalidTopology)
	}
	window.DesktopID = command.DestinationDesktopID
	window.Minimized = false
	window.ReturnAnchor = anchor
	if command.FloatingRect != nil {
		window.FloatingRect = clampRect(*command.FloatingRect)
	}
	snapshot.Windows[command.WindowID] = window
	if command.Placement == "" {
		command.Placement = DropFloating
	}
	if command.Placement == DropFloating {
		destinationIndex, _ := desktopIndexByID(snapshot, command.DestinationDesktopID)
		snapshot.Desktops[destinationIndex].Floating = append(
			snapshot.Desktops[destinationIndex].Floating,
			command.WindowID,
		)
	} else {
		if command.TargetWindowID == nil {
			return false, fmt.Errorf("structural move requires target: %w", ErrInvalidCommand)
		}
		if err := insertRelative(
			snapshot,
			*command.TargetWindowID,
			command.WindowID,
			command.Placement,
			r.generate,
		); err != nil {
			return false, err
		}
	}
	r.changes.window(command.WindowID)
	r.changes.desktop(window.DesktopID)
	if anchor != nil {
		r.changes.desktop(anchor.DesktopID)
	}
	return true, nil
}

func (r *reducer) moveWindowGroup(snapshot *Snapshot, command MoveWindowCommand) (bool, error) {
	placement, found := findWindowPlacement(snapshot, command.WindowID)
	if !found || placement.groupIndex < 0 {
		return false, fmt.Errorf("window %q is not grouped: %w", command.WindowID, ErrInvalidCommand)
	}
	sourceDesktop := &snapshot.Desktops[placement.desktopIndex]
	if sourceDesktop.ID == command.DestinationDesktopID {
		return false, nil
	}
	group := sourceDesktop.Groups[placement.groupIndex]
	sourceDesktop.Groups = append(
		sourceDesktop.Groups[:placement.groupIndex],
		sourceDesktop.Groups[placement.groupIndex+1:]...)
	destinationIndex, _ := desktopIndexByID(snapshot, command.DestinationDesktopID)
	snapshot.Desktops[destinationIndex].Groups = append(snapshot.Desktops[destinationIndex].Groups, group)
	for _, memberID := range nodeWindowIDs(group.Root) {
		member := snapshot.Windows[memberID]
		member.DesktopID = command.DestinationDesktopID
		snapshot.Windows[memberID] = member
		r.changes.window(memberID)
	}
	r.changes.group(group.ID)
	r.changes.desktop(sourceDesktop.ID)
	r.changes.desktop(command.DestinationDesktopID)
	return true, nil
}

func (r *reducer) swapWindows(snapshot *Snapshot, command SwapWindowsCommand) (bool, error) {
	if command.FirstWindowID == command.SecondWindowID {
		return false, nil
	}
	first, firstExists := snapshot.Windows[command.FirstWindowID]
	second, secondExists := snapshot.Windows[command.SecondWindowID]
	if !firstExists || !secondExists {
		return false, fmt.Errorf("swap window is missing: %w", ErrWindowNotFound)
	}
	firstPlacement, firstFound := findWindowPlacement(snapshot, first.ID)
	secondPlacement, secondFound := findWindowPlacement(snapshot, second.ID)
	if !firstFound || !secondFound {
		return false, fmt.Errorf("swap placement is missing: %w", ErrInvalidTopology)
	}
	if err := swapMemberships(
		snapshot,
		firstPlacement,
		secondPlacement,
		command.FirstWindowID,
		command.SecondWindowID,
	); err != nil {
		return false, err
	}
	first.DesktopID = snapshot.Desktops[secondPlacement.desktopIndex].ID
	second.DesktopID = snapshot.Desktops[firstPlacement.desktopIndex].ID
	snapshot.Windows[first.ID] = first
	snapshot.Windows[second.ID] = second
	r.changes.window(first.ID)
	r.changes.window(second.ID)
	r.changes.desktop(first.DesktopID)
	r.changes.desktop(second.DesktopID)
	return true, nil
}

func swapMemberships(
	snapshot *Snapshot,
	firstPlacement windowPlacement,
	secondPlacement windowPlacement,
	firstID WindowID,
	secondID WindowID,
) error {
	if firstPlacement.desktopIndex == secondPlacement.desktopIndex &&
		firstPlacement.floatingIndex >= 0 && secondPlacement.floatingIndex >= 0 {
		floating := snapshot.Desktops[firstPlacement.desktopIndex].Floating
		floating[firstPlacement.floatingIndex], floating[secondPlacement.floatingIndex] =
			floating[secondPlacement.floatingIndex], floating[firstPlacement.floatingIndex]
		return nil
	}
	if firstPlacement.desktopIndex == secondPlacement.desktopIndex &&
		firstPlacement.groupIndex >= 0 && firstPlacement.groupIndex == secondPlacement.groupIndex {
		root := &snapshot.Desktops[firstPlacement.desktopIndex].Groups[firstPlacement.groupIndex].Root
		if swapped := swapNodeWindows(root, firstID, secondID); swapped != 2 {
			return fmt.Errorf("swap group membership count %d: %w", swapped, ErrInvalidTopology)
		}
		return nil
	}
	setMembershipAtPlacement(snapshot, firstPlacement, secondID, firstID)
	setMembershipAtPlacement(snapshot, secondPlacement, firstID, secondID)
	return nil
}

func setMembershipAtPlacement(snapshot *Snapshot, placement windowPlacement, newID, oldID WindowID) {
	desktop := &snapshot.Desktops[placement.desktopIndex]
	if placement.floatingIndex >= 0 {
		desktop.Floating[placement.floatingIndex] = newID
		return
	}
	replaceNodeWindow(&desktop.Groups[placement.groupIndex].Root, oldID, newID)
}

func replaceNodeWindow(node *LayoutNode, oldID, newID WindowID) bool {
	if node.Kind == NodeKindLeaf && node.WindowID != nil && *node.WindowID == oldID {
		node.WindowID = &newID
		return true
	}
	if node.Kind == NodeKindStack {
		for index, id := range node.WindowIDs {
			if id == oldID {
				node.WindowIDs[index] = newID
				if node.ActiveID != nil && *node.ActiveID == oldID {
					node.ActiveID = &newID
				}
				return true
			}
		}
	}
	for index := range node.Children {
		if replaceNodeWindow(&node.Children[index], oldID, newID) {
			return true
		}
	}
	return false
}

func swapNodeWindows(node *LayoutNode, firstID, secondID WindowID) int {
	swapped := 0
	if node.Kind == NodeKindLeaf && node.WindowID != nil {
		switch *node.WindowID {
		case firstID:
			node.WindowID = &secondID
			swapped++
		case secondID:
			node.WindowID = &firstID
			swapped++
		}
	}
	if node.Kind == NodeKindStack {
		for index, windowID := range node.WindowIDs {
			switch windowID {
			case firstID:
				node.WindowIDs[index] = secondID
				swapped++
			case secondID:
				node.WindowIDs[index] = firstID
				swapped++
			}
		}
		if node.ActiveID != nil {
			switch *node.ActiveID {
			case firstID:
				node.ActiveID = &secondID
			case secondID:
				node.ActiveID = &firstID
			}
		}
	}
	for index := range node.Children {
		swapped += swapNodeWindows(&node.Children[index], firstID, secondID)
	}
	return swapped
}

func (r *reducer) toggleFloating(snapshot *Snapshot, command ToggleFloatingCommand) (bool, error) {
	window, exists := snapshot.Windows[command.WindowID]
	if !exists {
		return false, fmt.Errorf("window %q: %w", command.WindowID, ErrWindowNotFound)
	}
	if window.Placement == WindowPlacementFloating && !window.Minimized {
		removeWindow(snapshot, command.WindowID)
		if err := insertAtAnchor(
			snapshot,
			command.WindowID,
			window.ReturnAnchor,
			r.focusedWindow,
			r.generate,
		); err != nil {
			return false, err
		}
		placement, placed := findWindowPlacement(snapshot, command.WindowID)
		if !placed || placement.placement == WindowPlacementFloating {
			if placed {
				removeWindow(snapshot, command.WindowID)
			}
			if err := r.tileWindowFallback(snapshot, command.WindowID); err != nil {
				return false, err
			}
		}
		window = snapshot.Windows[command.WindowID]
		window.ReturnAnchor = nil
		snapshot.Windows[command.WindowID] = window
	} else {
		anchor := captureReturnAnchor(snapshot, command.WindowID)
		removeWindow(snapshot, command.WindowID)
		window.Placement = WindowPlacementFloating
		window.Minimized = false
		window.ReturnAnchor = anchor
		if command.FloatingRect != nil {
			window.FloatingRect = clampRect(*command.FloatingRect)
		} else {
			window.FloatingRect = clampRect(window.FloatingRect)
		}
		snapshot.Windows[command.WindowID] = window
		desktopIndex, _ := desktopIndexByID(snapshot, window.DesktopID)
		snapshot.Desktops[desktopIndex].Floating = append(snapshot.Desktops[desktopIndex].Floating, command.WindowID)
	}
	r.changes.window(command.WindowID)
	r.changes.desktop(window.DesktopID)
	return true, nil
}

func (r *reducer) tileWindowFallback(snapshot *Snapshot, windowID WindowID) error {
	window := snapshot.Windows[windowID]
	desktopIndex, exists := desktopIndexByID(snapshot, window.DesktopID)
	if !exists {
		return fmt.Errorf("desktop %q: %w", window.DesktopID, ErrDesktopNotFound)
	}
	desktop := &snapshot.Desktops[desktopIndex]
	if len(desktop.Groups) > 0 {
		members := nodeWindowIDs(desktop.Groups[0].Root)
		if len(members) > 0 {
			return insertRelative(snapshot, members[0], windowID, DropAfter, r.generate)
		}
	}
	leaf, err := newLeaf(windowID, r.generate)
	if err != nil {
		return err
	}
	groupIDRaw, err := r.generate("group")
	if err != nil {
		return fmt.Errorf("generate group ID: %w", err)
	}
	desktop.Groups = append(desktop.Groups, LayoutGroup{ID: GroupID(groupIDRaw), Frame: fullRect(), Root: leaf})
	r.changes.group(GroupID(groupIDRaw))
	r.changes.node(leaf.ID)
	return nil
}
