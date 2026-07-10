package daemon

import (
	"context"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/diagnosticcontract"
	"github.com/compozy/agh/internal/diagnostics"
	"github.com/compozy/agh/internal/providers"
	settingspkg "github.com/compozy/agh/internal/settings"
)

type daemonSettingsRuntimeApplier struct {
	daemon *Daemon
	state  *bootState
}

func (a daemonSettingsRuntimeApplier) ApplyActiveConfig(
	ctx context.Context,
	snap *aghconfig.Config,
) []settingspkg.ApplyFailure {
	if a.daemon == nil || a.state == nil || snap == nil {
		return nil
	}
	next := *snap

	a.daemon.mu.Lock()
	previous := a.state.cfg
	a.state.cfg = next
	a.daemon.config = next
	a.daemon.mu.Unlock()

	failures := a.applyRuntimeDependencies(ctx, &next)
	if len(failures) > 0 {
		a.daemon.mu.Lock()
		a.state.cfg = previous
		a.daemon.config = previous
		a.daemon.mu.Unlock()
		if a.state.modelCatalog != nil {
			if err := a.state.modelCatalog.ReconcileConfig(ctx, &previous); err != nil {
				failures = append(failures, configApplyFailure(
					"model_catalog_rollback",
					diagnosticcontract.CategoryConfig,
					"Model catalog rollback failed",
					err,
				))
			}
		}
		if a.state.toolMCPResources != nil {
			if err := a.state.toolMCPResources.Sync(ctx); err != nil {
				failures = append(failures, configApplyFailure(
					"mcp_rollback",
					diagnosticcontract.CategoryMCP,
					"MCP runtime rollback failed",
					err,
				))
			}
		}
		return failures
	}

	providers.InvalidatePreStartCache()
	return nil
}

func (a daemonSettingsRuntimeApplier) applyRuntimeDependencies(
	ctx context.Context,
	next *aghconfig.Config,
) []settingspkg.ApplyFailure {
	var failures []settingspkg.ApplyFailure
	if a.state.modelCatalog != nil {
		if err := a.state.modelCatalog.ReconcileConfig(ctx, next); err != nil {
			failures = append(failures, configApplyFailure(
				"model_catalog",
				diagnosticcontract.CategoryConfig,
				"Model catalog sync failed",
				err,
			))
		}
	}
	if a.state.toolMCPResources != nil {
		if err := a.state.toolMCPResources.Sync(ctx); err != nil {
			failures = append(failures, configApplyFailure(
				"mcp",
				diagnosticcontract.CategoryMCP,
				"MCP runtime sync failed",
				err,
			))
		}
	}
	return failures
}

func configApplyFailure(
	subsystem string,
	category string,
	summary string,
	err error,
) settingspkg.ApplyFailure {
	return settingspkg.ApplyFailure{
		Subsystem: subsystem,
		Diagnostic: diagnostics.NewItem(
			"config.apply."+subsystem+"_sync_failed",
			diagnosticcontract.CodeConfigPartialFailure,
			category,
			summary,
			diagnostics.RedactAndBound(err.Error(), 1024),
			diagnosticcontract.SeverityError,
			diagnosticcontract.FreshnessLive,
			diagnostics.WithSuggestedCommand("agh config reload"),
		),
	}
}
