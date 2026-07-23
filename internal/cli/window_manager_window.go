package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/windowmanager"
	"github.com/spf13/cobra"
)

const windowCommandKey = "window"

type windowOpenFlagValues struct {
	windowID    string
	app         string
	instanceKey string
	desktopID   string
	restoreID   string
	rectRaw     string
	pathname    string
	searchJSON  string
	insertTiled bool
}

func newWindowCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{Use: windowCommandKey, Short: "Manage windows in virtual desktops", Args: cobra.NoArgs}
	cmd.AddCommand(newWindowListCommand(deps))
	cmd.AddCommand(newWindowOpenCommand(deps))
	cmd.AddCommand(newWindowNavigateCommand(deps))
	cmd.AddCommand(newWindowCloseCommand(deps))
	cmd.AddCommand(newWindowFocusCommand(deps))
	cmd.AddCommand(newWindowMoveCommand(deps))
	cmd.AddCommand(newWindowSwapCommand(deps))
	cmd.AddCommand(newWindowFloatCommand(deps))
	cmd.AddCommand(newWindowZoomCommand(deps))
	return cmd
}

func newWindowListCommand(deps commandDeps) *cobra.Command {
	var workspace string
	cmd := &cobra.Command{
		Use: agentKernelListKey, Short: "List windows across workspace desktops", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := requiredWindowManagerFlag(workspace, windowManagerWorkspaceFlag)
			if err != nil {
				return err
			}
			client, err := windowManagerClientFromDeps(deps)
			if err != nil {
				return err
			}
			snapshot, err := client.GetWindowManagerSnapshot(cmd.Context(), workspace)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, windowManagerWindowListBundle(snapshot))
		},
	}
	cmd.Flags().StringVar(&workspace, windowManagerWorkspaceFlag, "", "Workspace ID")
	return cmd
}

func newWindowOpenCommand(deps commandDeps) *cobra.Command {
	var flags windowManagerMutationFlags
	var values windowOpenFlagValues
	cmd := &cobra.Command{
		Use:   windowManagerOpenKey,
		Short: "Open a window or restore a minimized window",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			restoreWindowID, err := optionalWindowManagerID[windowmanager.WindowID](
				cmd,
				"restore",
				values.restoreID,
			)
			if err != nil {
				return err
			}
			if restoreWindowID != nil {
				if cmd.Flags().Changed("app") || cmd.Flags().Changed("id") || cmd.Flags().Changed("instance-key") ||
					cmd.Flags().Changed("desktop") || cmd.Flags().Changed(windowManagerRectFlag) ||
					cmd.Flags().Changed(windowManagerPathnameFlag) ||
					cmd.Flags().Changed(windowManagerSearchJSONFlag) || values.insertTiled {
					return errors.New("cli: --restore cannot be combined with new-window flags")
				}
			}
			var spec contract.WindowManagerWindowSpecPayload
			if restoreWindowID == nil {
				values.app, err = requiredWindowManagerFlag(values.app, "app")
				if err != nil {
					return err
				}
				instance, instanceErr := optionalWindowManagerID[string](
					cmd,
					"instance-key",
					values.instanceKey,
				)
				if instanceErr != nil {
					return instanceErr
				}
				rect, rectErr := optionalWindowManagerRect(cmd, values.rectRaw)
				if rectErr != nil {
					return rectErr
				}
				var floatingRect windowmanager.NormalizedRect
				if rect != nil {
					floatingRect = *rect
				}
				route, routeErr := requiredWindowManagerRoute(cmd, values.pathname, values.searchJSON)
				if routeErr != nil {
					return routeErr
				}
				spec = contract.WindowManagerWindowSpecPayload{
					ID:           windowmanager.WindowID(strings.TrimSpace(values.windowID)),
					App:          values.app,
					InstanceKey:  instance,
					Route:        route,
					DesktopID:    windowmanager.DesktopID(strings.TrimSpace(values.desktopID)),
					FloatingRect: floatingRect, InsertTiled: values.insertTiled,
				}
			}
			request, err := flags.request(
				cmd,
				contract.WindowManagerCommandWindowOpen,
				contract.WindowManagerOpenWindowPayload{
					Window: spec, RestoreWindowID: restoreWindowID,
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
	values.addFlags(cmd)
	return cmd
}

func (values *windowOpenFlagValues) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&values.windowID, "id", "", "Stable window ID; generated when omitted")
	cmd.Flags().StringVar(&values.app, "app", "", "Application route identifier")
	cmd.Flags().StringVar(&values.instanceKey, "instance-key", "", "Optional application instance key")
	cmd.Flags().StringVar(
		&values.desktopID,
		"desktop",
		"",
		"Destination desktop; resolved from client focus when omitted",
	)
	cmd.Flags().StringVar(
		&values.restoreID,
		"restore",
		"",
		"Restore this minimized window instead of opening a new one",
	)
	cmd.Flags().StringVar(
		&values.rectRaw,
		windowManagerRectFlag,
		"",
		"Floating rect as x,y,width,height in normalized coordinates",
	)
	cmd.Flags().BoolVar(&values.insertTiled, "tiled", false, "Insert beside client focus when possible")
	addWindowManagerRouteFlags(cmd, &values.pathname, &values.searchJSON)
}

func newWindowCloseCommand(deps commandDeps) *cobra.Command {
	var flags windowManagerMutationFlags
	var windowID string
	var minimize bool
	cmd := &cobra.Command{
		Use: "close", Short: "Close or minimize a window and reflow its group", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			windowID, err := requiredWindowManagerFlag(windowID, "id")
			if err != nil {
				return err
			}
			request, err := flags.request(
				cmd,
				contract.WindowManagerCommandWindowClose,
				contract.WindowManagerCloseWindowPayload{
					WindowID: windowmanager.WindowID(windowID), Minimize: minimize,
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
	cmd.Flags().BoolVar(&minimize, "minimize", false, "Keep the window available for restore")
	return cmd
}

func newWindowFocusCommand(deps commandDeps) *cobra.Command {
	var flags windowManagerMutationFlags
	var windowID, direction string
	cmd := &cobra.Command{
		Use:   windowManagerFocusKey,
		Short: "Focus a window by ID or direction for one connected client",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requiredWindowManagerClient(flags); err != nil {
				return err
			}
			byID := strings.TrimSpace(windowID) != ""
			byDirection := strings.TrimSpace(direction) != ""
			if byID == byDirection {
				return newWindowManagerCLIValidationError(
					windowManagerCLIValidationConflicting,
					"focus_target",
					errors.New("cli: exactly one of --id or --direction is required"),
				)
			}
			var windowIDPtr *windowmanager.WindowID
			if byID {
				value := windowmanager.WindowID(strings.TrimSpace(windowID))
				windowIDPtr = &value
			}
			focusDirection := windowmanager.FocusDirection(strings.TrimSpace(direction))
			if byDirection && !isWindowManagerFocusDirection(focusDirection) {
				return fmt.Errorf("cli: unsupported --direction %q", direction)
			}
			request, err := flags.request(
				cmd,
				contract.WindowManagerCommandWindowFocus,
				contract.WindowManagerFocusWindowPayload{
					WindowID: windowIDPtr, Direction: focusDirection,
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
	cmd.Flags().StringVar(&direction, "direction", "", "Direction: left, right, up, or down")
	return cmd
}

func isWindowManagerFocusDirection(direction windowmanager.FocusDirection) bool {
	switch direction {
	case windowmanager.FocusLeft, windowmanager.FocusRight, windowmanager.FocusUp, windowmanager.FocusDown:
		return true
	default:
		return false
	}
}
