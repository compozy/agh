package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/windowmanager"
	"github.com/spf13/cobra"
)

func newWindowMoveCommand(deps commandDeps) *cobra.Command {
	var flags windowManagerMutationFlags
	var windowID, destination, targetID, placement, rectRaw string
	var moveGroup bool
	cmd := &cobra.Command{
		Use:   windowManagerMoveKey,
		Short: "Move a window or tiled group to an explicit destination",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			windowID, err := requiredWindowManagerFlag(windowID, "id")
			if err != nil {
				return err
			}
			destination, err := requiredWindowManagerFlag(destination, "desktop")
			if err != nil {
				return err
			}
			dropPlacement := windowmanager.DropPlacement(strings.TrimSpace(placement))
			if !isWindowManagerDropPlacement(dropPlacement) {
				return fmt.Errorf("cli: unsupported --placement %q", placement)
			}
			target, err := optionalWindowManagerID[windowmanager.WindowID](cmd, "target", targetID)
			if err != nil {
				return err
			}
			if !moveGroup && dropPlacement != windowmanager.DropFloating && target == nil {
				return newWindowManagerCLIValidationError(
					windowManagerCLIValidationRequired,
					"target",
					errors.New("cli: --target is required for structural placement"),
				)
			}
			rect, err := optionalWindowManagerRect(cmd, rectRaw)
			if err != nil {
				return err
			}
			if rect != nil && dropPlacement != windowmanager.DropFloating {
				return errors.New("cli: --rect is only valid with --placement=floating")
			}
			request, err := flags.request(
				cmd,
				contract.WindowManagerCommandWindowMove,
				contract.WindowManagerMoveWindowPayload{
					WindowID:             windowmanager.WindowID(windowID),
					DestinationDesktopID: windowmanager.DesktopID(destination),
					TargetWindowID:       target, Placement: dropPlacement,
					FloatingRect: rect, MoveGroup: moveGroup,
				},
			)
			if err != nil {
				return err
			}
			result, err := executeWindowManagerCommand(cmd, deps, request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, windowManagerResultBundle(result))
		},
	}
	flags.add(cmd)
	cmd.Flags().StringVar(&windowID, "id", "", "Window ID")
	cmd.Flags().StringVar(&destination, "desktop", "", "Destination desktop ID")
	cmd.Flags().StringVar(&targetID, "target", "", "Target window for structural placement")
	cmd.Flags().StringVar(
		&placement,
		"placement",
		string(windowmanager.DropFloating),
		"Placement: floating, before, after, left, right, top, bottom, or center",
	)
	cmd.Flags().StringVar(&rectRaw, windowManagerRectFlag, "", "Floating rect as x,y,width,height")
	cmd.Flags().BoolVar(&moveGroup, "group", false, "Move the complete tiled group")
	return cmd
}

func newWindowSwapCommand(deps commandDeps) *cobra.Command {
	var flags windowManagerMutationFlags
	var firstID, secondID string
	cmd := &cobra.Command{
		Use: "swap", Short: "Swap two window placements", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			firstID, err := requiredWindowManagerFlag(firstID, "first")
			if err != nil {
				return err
			}
			secondID, err := requiredWindowManagerFlag(secondID, "second")
			if err != nil {
				return err
			}
			request, err := flags.request(
				cmd,
				contract.WindowManagerCommandWindowSwap,
				contract.WindowManagerSwapWindowsPayload{
					FirstWindowID: windowmanager.WindowID(firstID), SecondWindowID: windowmanager.WindowID(secondID),
				},
			)
			if err != nil {
				return err
			}
			result, err := executeWindowManagerCommand(cmd, deps, request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, windowManagerResultBundle(result))
		},
	}
	flags.add(cmd)
	cmd.Flags().StringVar(&firstID, "first", "", "First window ID")
	cmd.Flags().StringVar(&secondID, "second", "", "Second window ID")
	return cmd
}

func newWindowFloatCommand(deps commandDeps) *cobra.Command {
	var flags windowManagerMutationFlags
	var windowID, rectRaw string
	cmd := &cobra.Command{
		Use: "float", Short: "Toggle a window between floating and tiled placement", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			windowID, err := requiredWindowManagerFlag(windowID, "id")
			if err != nil {
				return err
			}
			rect, err := optionalWindowManagerRect(cmd, rectRaw)
			if err != nil {
				return err
			}
			request, err := flags.request(
				cmd,
				contract.WindowManagerCommandWindowToggleFloating,
				contract.WindowManagerToggleFloatingPayload{
					WindowID: windowmanager.WindowID(windowID), FloatingRect: rect,
				},
			)
			if err != nil {
				return err
			}
			result, err := executeWindowManagerCommand(cmd, deps, request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, windowManagerResultBundle(result))
		},
	}
	flags.add(cmd)
	cmd.Flags().StringVar(&windowID, "id", "", "Window ID")
	cmd.Flags().StringVar(&rectRaw, windowManagerRectFlag, "", "Floating rect as x,y,width,height")
	return cmd
}

func newWindowZoomCommand(deps commandDeps) *cobra.Command {
	var flags windowManagerMutationFlags
	var windowID string
	cmd := &cobra.Command{
		Use: "zoom", Short: "Toggle a window's client-specific focus desktop", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requiredWindowManagerClient(flags); err != nil {
				return err
			}
			windowID, err := requiredWindowManagerFlag(windowID, "id")
			if err != nil {
				return err
			}
			request, err := flags.request(
				cmd,
				contract.WindowManagerCommandWindowZoom,
				contract.WindowManagerZoomWindowPayload{
					WindowID: windowmanager.WindowID(windowID),
				},
			)
			if err != nil {
				return err
			}
			result, err := executeWindowManagerCommand(cmd, deps, request)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, windowManagerResultBundle(result))
		},
	}
	flags.add(cmd)
	cmd.Flags().StringVar(&windowID, "id", "", "Window ID")
	return cmd
}

func isWindowManagerDropPlacement(placement windowmanager.DropPlacement) bool {
	switch placement {
	case windowmanager.DropFloating, windowmanager.DropBefore, windowmanager.DropAfter,
		windowmanager.DropLeft, windowmanager.DropRight, windowmanager.DropTop,
		windowmanager.DropBottom, windowmanager.DropCenter:
		return true
	default:
		return false
	}
}
