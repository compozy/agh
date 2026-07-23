package windowmanager

import (
	"fmt"
	"math"
)

func (r *reducer) arrange(snapshot *Snapshot, command ArrangeLayoutCommand) (bool, error) {
	desktopIndex, exists := desktopIndexByID(snapshot, command.DesktopID)
	if !exists {
		return false, fmt.Errorf("desktop %q: %w", command.DesktopID, ErrDesktopNotFound)
	}
	if command.ResourceID != "" {
		return false, fmt.Errorf("unresolved layout resource %q: %w", command.ResourceID, ErrInvalidCommand)
	}
	if len(command.WindowIDs) == 0 {
		return false, fmt.Errorf("arrange participants are required: %w", ErrInvalidCommand)
	}
	seen := make(map[WindowID]struct{}, len(command.WindowIDs))
	for _, windowID := range command.WindowIDs {
		window, found := snapshot.Windows[windowID]
		if !found {
			return false, fmt.Errorf("window %q: %w", windowID, ErrWindowNotFound)
		}
		if window.DesktopID != command.DesktopID {
			return false, fmt.Errorf("window %q belongs to another desktop: %w", windowID, ErrInvalidCommand)
		}
		if _, duplicate := seen[windowID]; duplicate {
			return false, fmt.Errorf("window %q is duplicated: %w", windowID, ErrInvalidCommand)
		}
		seen[windowID] = struct{}{}
	}
	for _, windowID := range command.WindowIDs {
		removeWindow(snapshot, windowID)
		window := snapshot.Windows[windowID]
		window.Minimized = false
		window.ReturnAnchor = nil
		snapshot.Windows[windowID] = window
		r.changes.window(windowID)
	}
	root, err := r.buildArrangement(command.WindowIDs, command.Arrangement)
	if err != nil {
		return false, err
	}
	groupID := command.GroupID
	if groupID == "" {
		generated, generateErr := r.generate("group")
		if generateErr != nil {
			return false, fmt.Errorf("generate group ID: %w", generateErr)
		}
		groupID = GroupID(generated)
	}
	frame := command.Frame
	if frame.Width == 0 && frame.Height == 0 {
		frame = fullRect()
	}
	snapshot.Desktops[desktopIndex].Groups = append(
		snapshot.Desktops[desktopIndex].Groups,
		LayoutGroup{ID: groupID, Frame: frame, Root: root},
	)
	r.changes.desktop(command.DesktopID)
	r.changes.group(groupID)
	return true, nil
}

func (r *reducer) buildArrangement(windowIDs []WindowID, arrangement Arrangement) (LayoutNode, error) {
	if len(windowIDs) == 1 {
		return newLeaf(windowIDs[0], r.generate)
	}
	switch arrangement {
	case ArrangementHorizontal:
		return r.buildSplit(windowIDs, AxisHorizontal)
	case ArrangementVertical:
		return r.buildSplit(windowIDs, AxisVertical)
	case ArrangementStack:
		id, err := r.generate("node")
		if err != nil {
			return LayoutNode{}, fmt.Errorf("generate stack ID: %w", err)
		}
		active := windowIDs[0]
		return LayoutNode{
			ID:        NodeID(id),
			Kind:      NodeKindStack,
			WindowIDs: append([]WindowID(nil), windowIDs...),
			ActiveID:  &active,
		}, nil
	case ArrangementGrid:
		columns := int(math.Ceil(math.Sqrt(float64(len(windowIDs)))))
		rows := make([]LayoutNode, 0, (len(windowIDs)+columns-1)/columns)
		for start := 0; start < len(windowIDs); start += columns {
			end := min(start+columns, len(windowIDs))
			row, err := r.buildArrangement(windowIDs[start:end], ArrangementHorizontal)
			if err != nil {
				return LayoutNode{}, err
			}
			rows = append(rows, row)
		}
		if len(rows) == 1 {
			return rows[0], nil
		}
		id, err := r.generate("node")
		if err != nil {
			return LayoutNode{}, fmt.Errorf("generate grid root ID: %w", err)
		}
		weights := equalWeights(len(rows))
		axis := AxisVertical
		return LayoutNode{ID: NodeID(id), Kind: NodeKindSplit, Axis: &axis, Children: rows, Weights: weights}, nil
	default:
		return LayoutNode{}, fmt.Errorf("arrangement %q: %w", arrangement, ErrInvalidCommand)
	}
}

func (r *reducer) buildSplit(windowIDs []WindowID, axis Axis) (LayoutNode, error) {
	children := make([]LayoutNode, len(windowIDs))
	for index, windowID := range windowIDs {
		leaf, err := newLeaf(windowID, r.generate)
		if err != nil {
			return LayoutNode{}, err
		}
		children[index] = leaf
		r.changes.node(leaf.ID)
	}
	id, err := r.generate("node")
	if err != nil {
		return LayoutNode{}, fmt.Errorf("generate split ID: %w", err)
	}
	r.changes.node(NodeID(id))
	return LayoutNode{
		ID:       NodeID(id),
		Kind:     NodeKindSplit,
		Axis:     &axis,
		Children: children,
		Weights:  equalWeights(len(children)),
	}, nil
}

func (r *reducer) resize(snapshot *Snapshot, command ResizeLayoutCommand) (bool, error) {
	if !finite(command.Delta) {
		return false, fmt.Errorf("resize delta is not finite: %w", ErrInvalidCommand)
	}
	node, exists := findNodeInSnapshot(snapshot, command.SplitID)
	if !exists || node.Kind != NodeKindSplit {
		return false, fmt.Errorf("split %q: %w", command.SplitID, ErrInvalidCommand)
	}
	if command.BoundaryIndex < 0 || command.BoundaryIndex >= len(node.Children)-1 {
		return false, fmt.Errorf("boundary %d: %w", command.BoundaryIndex, ErrInvalidCommand)
	}
	left := command.BoundaryIndex
	right := left + 1
	leftWeight := node.Weights[left] + command.Delta
	rightWeight := node.Weights[right] - command.Delta
	if leftWeight < 0.01 || rightWeight < 0.01 {
		return false, fmt.Errorf("resize exceeds boundary: %w", ErrInvalidCommand)
	}
	if math.Abs(command.Delta) <= weightTolerance {
		return false, nil
	}
	node.Weights[left] = leftWeight
	node.Weights[right] = rightWeight
	normalizeWeights(node.Weights)
	r.changes.node(command.SplitID)
	return true, nil
}

func (r *reducer) balance(snapshot *Snapshot, command BalanceLayoutCommand) (bool, error) {
	if command.SplitID == nil && command.GroupID == nil {
		return false, fmt.Errorf("balance target is required: %w", ErrInvalidCommand)
	}
	if command.SplitID != nil {
		node, exists := findNodeInSnapshot(snapshot, *command.SplitID)
		if !exists || node.Kind != NodeKindSplit {
			return false, fmt.Errorf("split %q: %w", *command.SplitID, ErrInvalidCommand)
		}
		if weightsEqual(node.Weights) {
			return false, nil
		}
		node.Weights = equalWeights(len(node.Children))
		r.changes.node(node.ID)
		return true, nil
	}
	for desktopIndex := range snapshot.Desktops {
		for groupIndex := range snapshot.Desktops[desktopIndex].Groups {
			group := &snapshot.Desktops[desktopIndex].Groups[groupIndex]
			if group.ID != *command.GroupID {
				continue
			}
			changed := balanceNode(&group.Root, &r.changes)
			if changed {
				r.changes.group(group.ID)
			}
			return changed, nil
		}
	}
	return false, fmt.Errorf("group %q: %w", *command.GroupID, ErrInvalidCommand)
}

func balanceNode(node *LayoutNode, changes *changeBuilder) bool {
	changed := false
	if node.Kind == NodeKindSplit {
		if !weightsEqual(node.Weights) {
			node.Weights = equalWeights(len(node.Children))
			changes.node(node.ID)
			changed = true
		}
		for index := range node.Children {
			if balanceNode(&node.Children[index], changes) {
				changed = true
			}
		}
	}
	return changed
}

func equalWeights(count int) []float64 {
	values := make([]float64, count)
	for index := range values {
		values[index] = 1 / float64(count)
	}
	return values
}
func weightsEqual(weights []float64) bool {
	if len(weights) == 0 {
		return true
	}
	expected := 1 / float64(len(weights))
	for _, weight := range weights {
		if math.Abs(weight-expected) > weightTolerance {
			return false
		}
	}
	return true
}
