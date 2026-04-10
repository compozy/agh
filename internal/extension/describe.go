package extension

import (
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
)

// DescribeExtension projects one extension snapshot into the shared CLI/API payload.
func DescribeExtension(ext *Extension, daemonRunning bool, now time.Time) contract.ExtensionPayload {
	if ext == nil {
		return contract.ExtensionPayload{}
	}

	uptimeSeconds := int64(0)
	if ext.Status.Active && !ext.Status.LastStartedAt.IsZero() {
		uptimeSeconds = int64(now.Sub(ext.Status.LastStartedAt).Seconds())
		if uptimeSeconds < 0 {
			uptimeSeconds = 0
		}
	}

	return contract.ExtensionPayload{
		Name:          ext.Info.Name,
		Version:       ext.Info.Version,
		Type:          extensionType(ext.Manifest, ext.Info),
		Source:        ext.Info.Source.String(),
		Enabled:       ext.Info.Enabled,
		State:         extensionState(ext.Info, ext.Status, daemonRunning),
		Capabilities:  append([]string(nil), ext.Info.Capabilities.Provides...),
		Actions:       append([]string(nil), ext.Info.Actions.Requires...),
		PID:           ext.Status.PID,
		UptimeSeconds: uptimeSeconds,
		Health:        extensionHealth(ext.Manifest, ext.Info, ext.Status, daemonRunning),
		HealthMessage: ext.Status.HealthMessage,
		LastError:     ext.Status.LastError,
		DaemonRunning: daemonRunning,
	}
}

func extensionType(manifest *Manifest, info ExtensionInfo) string {
	if requiresSubprocess(manifest) || len(info.Capabilities.Provides) > 0 || len(info.Actions.Requires) > 0 {
		return "subprocess"
	}
	return "resource"
}

func extensionState(info ExtensionInfo, status ExtensionStatus, daemonRunning bool) string {
	if !info.Enabled {
		return "disabled"
	}
	if !daemonRunning {
		return "enabled"
	}
	if status.Active {
		return "active"
	}
	if status.LastError != "" {
		return "error"
	}
	if status.Registered {
		return "registered"
	}
	return "enabled"
}

func extensionHealth(manifest *Manifest, info ExtensionInfo, status ExtensionStatus, daemonRunning bool) string {
	if !daemonRunning {
		return "unknown"
	}
	if status.Active {
		if status.Healthy || !requiresSubprocess(manifest) && len(info.Capabilities.Provides) == 0 && len(info.Actions.Requires) == 0 {
			return "healthy"
		}
		return "unhealthy"
	}
	if status.LastError != "" {
		return "unhealthy"
	}
	if !requiresSubprocess(manifest) && len(info.Capabilities.Provides) == 0 && len(info.Actions.Requires) == 0 && status.Registered {
		return "healthy"
	}
	return "unknown"
}
