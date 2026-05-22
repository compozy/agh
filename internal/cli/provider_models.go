package cli

import (
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/spf13/cobra"
)

const (
	providerModelsErrorValue   = "Error"
	providerModelsModelValue   = "Model"
	providerModelsAvailableKey = "available"
)

const providerModelAvailabilityUnknown = "unknown"

func newProviderModelsCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "Inspect and refresh the provider model catalog",
	}
	cmd.AddCommand(newProviderModelsListCommand(deps))
	cmd.AddCommand(newProviderModelsRefreshCommand(deps))
	cmd.AddCommand(newProviderModelsStatusCommand(deps))
	return cmd
}

func newProviderModelsListCommand(deps commandDeps) *cobra.Command {
	var sourceID string
	var refresh bool
	var includeStale bool
	cmd := &cobra.Command{
		Use:   "list [provider]",
		Short: "List provider model catalog entries",
		Args:  optionalProviderArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			providerID := providerArgValue(args)
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			record, err := client.ListProviderModels(cmd.Context(), ProviderModelListQuery{
				ProviderID:   providerID,
				SourceID:     sourceID,
				Refresh:      refresh,
				IncludeStale: includeStale,
			})
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, providerModelListBundle(record))
		},
	}
	cmd.Flags().StringVar(&sourceID, "source", "", "Filter by catalog source id")
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Refresh sources before listing models")
	cmd.Flags().BoolVar(&includeStale, "include-stale", false, "Include stale source rows")
	return cmd
}

func newProviderModelsRefreshCommand(deps commandDeps) *cobra.Command {
	var sourceID string
	var force bool
	var requestID string
	cmd := &cobra.Command{
		Use:   "refresh [provider]",
		Short: "Refresh provider model catalog sources",
		Args:  optionalProviderArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			record, err := client.RefreshProviderModels(
				cmd.Context(),
				providerArgValue(args),
				ProviderModelRefreshRequest{
					SourceID:  sourceID,
					Force:     force,
					RequestID: requestID,
				},
			)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, providerModelRefreshBundle(record))
		},
	}
	cmd.Flags().StringVar(&sourceID, "source", "", "Refresh only one catalog source id")
	cmd.Flags().BoolVar(&force, "force", false, "Force refresh even when cached status is fresh")
	cmd.Flags().StringVar(&requestID, "request-id", "", "Refresh request id for daemon logs")
	return cmd
}

func newProviderModelsStatusCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [provider]",
		Short: "Show provider model catalog source status",
		Args:  optionalProviderArg(),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFromDeps(deps)
			if err != nil {
				return err
			}
			record, err := client.ProviderModelStatus(cmd.Context(), providerArgValue(args))
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, providerModelStatusBundle(record))
		},
	}
	return cmd
}

func providerModelListBundle(record ProviderModelListRecord) outputBundle {
	return listBundle(
		record,
		record.Models,
		"Provider Models",
		[]string{
			agentKernelProviderValue,
			providerModelsModelValue,
			"Available",
			authoredContextStateValue,
			"Stale",
			"Sources",
			"Refreshed",
		},
		"provider_models",
		[]string{
			"provider_id",
			"model_id",
			providerModelsAvailableKey,
			"availability_state",
			"stale",
			"sources",
			"refreshed_at",
		},
		func(model ProviderModelRecord) []string {
			return []string{
				model.ProviderID,
				model.ModelID,
				providerModelNullableBoolString(model.Available),
				model.AvailabilityState,
				providerModelBoolString(model.Stale),
				providerModelSourceSummary(model.Sources),
				model.RefreshedAt,
			}
		},
		func(model ProviderModelRecord) []string {
			return []string{
				model.ProviderID,
				model.ModelID,
				providerModelNullableBoolString(model.Available),
				model.AvailabilityState,
				providerModelBoolString(model.Stale),
				providerModelSourceSummary(model.Sources),
				model.RefreshedAt,
			}
		},
	)
}

func providerModelRefreshBundle(record ProviderModelRefreshRecord) outputBundle {
	return providerModelSourceStatusBundle("Provider Model Refresh", "provider_model_refresh", record, record.Sources)
}

func providerModelStatusBundle(record ProviderModelStatusRecord) outputBundle {
	return providerModelSourceStatusBundle("Provider Model Status", "provider_model_status", record, record.Sources)
}

func providerModelSourceStatusBundle(
	humanTitle string,
	toonName string,
	jsonValue any,
	sources []ProviderModelSourceStatusRecord,
) outputBundle {
	return listBundle(
		jsonValue,
		sources,
		humanTitle,
		[]string{
			agentKernelProviderValue,
			authoredContextSourceValue,
			bridgeKindValue,
			authoredContextStateValue,
			"Rows",
			"Stale",
			providerModelsErrorValue,
		},
		toonName,
		[]string{"provider_id", "source_id", "source_kind", "refresh_state", "row_count", "stale", "last_error"},
		providerModelSourceStatusRow,
		providerModelSourceStatusRow,
	)
}

func providerModelSourceStatusRow(source ProviderModelSourceStatusRecord) []string {
	return []string{
		source.ProviderID,
		source.SourceID,
		source.SourceKind,
		source.RefreshState,
		fmt.Sprintf("%d", source.RowCount),
		providerModelBoolString(source.Stale),
		source.LastError,
	}
}

func optionalProviderArg() cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("accepts at most 1 arg(s), received %d", len(args))
		}
		if len(args) == 1 && strings.TrimSpace(args[0]) == "" {
			return fmt.Errorf("provider id cannot be blank")
		}
		return nil
	}
}

func providerArgValue(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return strings.TrimSpace(args[0])
}

func providerModelSourceSummary(sources []contract.ModelCatalogSourceRefPayload) string {
	if len(sources) == 0 {
		return ""
	}
	values := make([]string, 0, len(sources))
	for _, source := range sources {
		values = append(values, source.SourceID)
	}
	return strings.Join(values, ",")
}

func providerModelNullableBoolString(value *bool) string {
	if value == nil {
		return providerModelAvailabilityUnknown
	}
	return providerModelBoolString(*value)
}

func providerModelBoolString(value bool) string {
	if value {
		return toolBoolTrue
	}
	return toolBoolFalse
}
