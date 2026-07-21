package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/spf13/cobra"
)

const (
	desktopStateCommandKey = "desktop-state"
	desktopStateWorkspace  = "workspace"
	desktopStateIfRev      = "if-rev"
)

func newDesktopStateCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{Use: desktopStateCommandKey, Short: "Manage workspace desktop state"}
	cmd.AddCommand(newDesktopStateListCommand(deps))
	cmd.AddCommand(newDesktopStateGetCommand(deps))
	cmd.AddCommand(newDesktopStateSetCommand(deps))
	cmd.AddCommand(newDesktopStateDeleteCommand(deps))
	cmd.AddCommand(newDesktopStateWatchCommand(deps))
	return cmd
}

func newDesktopStateListCommand(deps commandDeps) *cobra.Command {
	var workspace string
	cmd := &cobra.Command{
		Use:   agentKernelListKey,
		Short: "List the saved desktop state for a workspace",
		Example: `  agh desktop-state list --workspace ws_1234
  agh desktop-state list --workspace ws_1234 -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := requiredDesktopStateFlag(workspace, desktopStateWorkspace)
			if err != nil {
				return err
			}
			client, err := desktopStateClientFromDeps(deps)
			if err != nil {
				return err
			}
			response, err := client.ListDesktopState(cmd.Context(), workspace)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, desktopStateListBundle(response.Entries))
		},
	}
	cmd.Flags().StringVar(&workspace, desktopStateWorkspace, "", "Workspace id")
	return cmd
}

func newDesktopStateGetCommand(deps commandDeps) *cobra.Command {
	var workspace, key string
	cmd := &cobra.Command{
		Use:   cliGetKey,
		Short: "Read one saved desktop-state value",
		Example: `  agh desktop-state get --workspace ws_1234 --key desktop
  agh desktop-state get --workspace ws_1234 --key 'win:app:tasks' -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, key, err := requiredDesktopStateIdentity(workspace, key)
			if err != nil {
				return err
			}
			client, err := desktopStateClientFromDeps(deps)
			if err != nil {
				return err
			}
			entry, err := client.GetDesktopState(cmd.Context(), workspace, key)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, desktopStateEntryBundle(entry))
		},
	}
	addDesktopStateIdentityFlags(cmd, &workspace, &key)
	return cmd
}

func newDesktopStateSetCommand(deps commandDeps) *cobra.Command {
	var workspace, key, value, file string
	var ifRev uint64
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Create or replace one desktop-state value",
		Example: `  agh desktop-state set --workspace ws_1234 --key desktop --value '{"v":1}'
  agh desktop-state set --workspace ws_1234 --key 'win:app:tasks' --file window.json --if-rev 3 -o json`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, key, err := requiredDesktopStateIdentity(workspace, key)
			if err != nil {
				return err
			}
			payload, err := desktopStateValueInput(cmd, value, file)
			if err != nil {
				return err
			}
			client, err := desktopStateClientFromDeps(deps)
			if err != nil {
				return err
			}
			entry, err := client.PutDesktopState(cmd.Context(), workspace, key, contract.DesktopStatePutRequest{
				Value: payload,
				IfRev: optionalDesktopStateRevision(cmd, ifRev),
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, desktopStateEntryBundle(entry))
		},
	}
	addDesktopStateIdentityFlags(cmd, &workspace, &key)
	cmd.Flags().StringVar(&value, "value", "", "Desktop-state value as a JSON object")
	cmd.Flags().StringVar(&file, "file", "", "Read the desktop-state JSON object from a file")
	cmd.Flags().Uint64Var(&ifRev, desktopStateIfRev, 0, "Require this current revision")
	return cmd
}

func newDesktopStateDeleteCommand(deps commandDeps) *cobra.Command {
	var workspace, key string
	var ifRev uint64
	cmd := &cobra.Command{
		Use:   loopDeleteKey,
		Short: "Delete one desktop-state value",
		Example: `  agh desktop-state delete --workspace ws_1234 --key 'win:app:tasks'
  agh desktop-state delete --workspace ws_1234 --key desktop --if-rev 4`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, key, err := requiredDesktopStateIdentity(workspace, key)
			if err != nil {
				return err
			}
			client, err := desktopStateClientFromDeps(deps)
			if err != nil {
				return err
			}
			if err := client.DeleteDesktopState(
				cmd.Context(), workspace, key, optionalDesktopStateRevision(cmd, ifRev),
			); err != nil {
				return err
			}
			return writeCommandOutput(cmd, desktopStateDeleteBundle(key))
		},
	}
	addDesktopStateIdentityFlags(cmd, &workspace, &key)
	cmd.Flags().Uint64Var(&ifRev, desktopStateIfRev, 0, "Require this current revision")
	return cmd
}

func newDesktopStateWatchCommand(deps commandDeps) *cobra.Command {
	var workspace string
	cmd := &cobra.Command{
		Use:     "watch",
		Short:   "Watch committed desktop-state changes",
		Example: `  agh desktop-state watch --workspace ws_1234 -o jsonl`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspace, err := requiredDesktopStateFlag(workspace, desktopStateWorkspace)
			if err != nil {
				return err
			}
			format, err := resolveOutputFormat(cmd)
			if err != nil {
				return err
			}
			if format != OutputHuman && format != OutputJSONL {
				return errors.New("cli: desktop-state watch supports human or jsonl output")
			}
			client, err := desktopStateClientFromDeps(deps)
			if err != nil {
				return err
			}
			return client.WatchDesktopState(
				cmd.Context(),
				workspace,
				func(event contract.DesktopStateEventFrame) error {
					if format == OutputJSONL {
						return writeJSONLine(cmd, event)
					}
					_, err := fmt.Fprintf(
						cmd.OutOrStdout(),
						"%d\t%s\trev=%d\tdeleted=%t\n",
						event.Entry.Seq,
						event.Entry.Key,
						event.Entry.Rev,
						event.Entry.Deleted,
					)
					return err
				},
			)
		},
	}
	cmd.Flags().StringVar(&workspace, desktopStateWorkspace, "", "Workspace id")
	return cmd
}

func addDesktopStateIdentityFlags(cmd *cobra.Command, workspace *string, key *string) {
	cmd.Flags().StringVar(workspace, desktopStateWorkspace, "", "Workspace id")
	cmd.Flags().StringVar(key, cliKeyKey, "", "Desktop-state key")
}

func desktopStateClientFromDeps(deps commandDeps) (DesktopStateClient, error) {
	client, err := clientFromDeps(deps)
	if err != nil {
		return nil, err
	}
	desktopStateClient, ok := client.(DesktopStateClient)
	if !ok {
		return nil, errors.New("cli: desktop-state client is unavailable")
	}
	return desktopStateClient, nil
}

func requiredDesktopStateIdentity(workspace string, key string) (string, string, error) {
	workspace, err := requiredDesktopStateFlag(workspace, desktopStateWorkspace)
	if err != nil {
		return "", "", err
	}
	key, err = requiredDesktopStateFlag(key, cliKeyKey)
	if err != nil {
		return "", "", err
	}
	return workspace, key, nil
}

func requiredDesktopStateFlag(value string, name string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("cli: --%s is required", name)
	}
	return trimmed, nil
}

func optionalDesktopStateRevision(
	cmd *cobra.Command,
	value uint64,
) *contract.DesktopStateSafeNumber {
	if !cmd.Flags().Changed(desktopStateIfRev) {
		return nil
	}
	revision := contract.DesktopStateSafeNumber(value)
	return &revision
}

func desktopStateValueInput(cmd *cobra.Command, value string, file string) (map[string]any, error) {
	hasValue := cmd.Flags().Changed("value")
	hasFile := cmd.Flags().Changed("file")
	if hasValue == hasFile {
		return nil, errors.New("cli: exactly one of --value or --file is required")
	}
	data := []byte(value)
	if hasFile {
		loaded, err := os.ReadFile(strings.TrimSpace(file))
		if err != nil {
			return nil, fmt.Errorf("cli: read desktop-state file: %w", err)
		}
		data = loaded
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var payload map[string]any
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("cli: decode desktop-state value: %w", err)
	}
	if payload == nil {
		return nil, errors.New("cli: desktop-state value must be a JSON object")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("cli: desktop-state value must contain one JSON object")
		}
		return nil, fmt.Errorf("cli: decode desktop-state value: %w", err)
	}
	return payload, nil
}

func desktopStateEntryBundle(entry contract.DesktopStateEntry) outputBundle {
	return outputBundle{
		jsonValue: entry,
		human: func() (string, error) {
			value, err := json.Marshal(entry.Value)
			if err != nil {
				return "", fmt.Errorf("cli: encode desktop-state value: %w", err)
			}
			return renderHumanSection("Desktop state", []keyValue{
				{Label: cliKeyValue, Value: entry.Key},
				{Label: cliRevisionValue, Value: strconv.FormatUint(uint64(entry.Rev), 10)},
				{Label: "Sequence", Value: strconv.FormatUint(uint64(entry.Seq), 10)},
				{Label: "Updated", Value: entry.UpdatedAt.Format(time.RFC3339)},
				{Label: "Value", Value: string(value)},
			}), nil
		},
		toon: func() (string, error) {
			value, err := json.Marshal(entry.Value)
			if err != nil {
				return "", fmt.Errorf("cli: encode desktop-state value: %w", err)
			}
			return renderToonObject(
				desktopStateCommandKey,
				[]string{cliKeyKey, "rev", "seq", bridgeDeletedKey, bridgeUpdatedAtKey, "value"},
				[]string{
					entry.Key,
					strconv.FormatUint(uint64(entry.Rev), 10),
					strconv.FormatUint(uint64(entry.Seq), 10),
					strconv.FormatBool(entry.Deleted), entry.UpdatedAt.Format(time.RFC3339), string(value),
				},
			), nil
		},
	}
}

func desktopStateListBundle(entries []contract.DesktopStateEntry) outputBundle {
	return listBundle(
		entries,
		entries,
		"Desktop state",
		[]string{"KEY", "REV", "SEQ", "UPDATED"},
		desktopStateCommandKey,
		[]string{cliKeyKey, "rev", "seq", bridgeUpdatedAtKey},
		func(entry contract.DesktopStateEntry) []string {
			return []string{
				entry.Key,
				strconv.FormatUint(uint64(entry.Rev), 10),
				strconv.FormatUint(uint64(entry.Seq), 10),
				entry.UpdatedAt.Format(time.RFC3339),
			}
		},
		func(entry contract.DesktopStateEntry) []string {
			return []string{
				entry.Key,
				strconv.FormatUint(uint64(entry.Rev), 10),
				strconv.FormatUint(uint64(entry.Seq), 10),
				entry.UpdatedAt.Format(time.RFC3339),
			}
		},
	)
}

func desktopStateDeleteBundle(key string) outputBundle {
	payload := struct {
		Deleted bool   `json:"deleted"`
		Key     string `json:"key"`
	}{Deleted: true, Key: key}
	return outputBundle{
		jsonValue: payload,
		human: func() (string, error) {
			return fmt.Sprintf("Deleted desktop state %s", key), nil
		},
		toon: func() (string, error) {
			return renderToonObject(
				desktopStateCommandKey,
				[]string{cliKeyKey, bridgeDeletedKey},
				[]string{key, toolBoolTrue},
			), nil
		},
	}
}
