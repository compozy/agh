package transcript

import (
	"strings"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/store"
)

const (
	sessionStoppedEventType = "session_stopped"

	interruptMarker = "*[interrupted]*"
	timeoutMarker   = "*[timeout]*"
	unhealthyMarker = "*[unhealthy]*"
)

func transcriptMarkerText(parsed event) string {
	switch parsed.Type {
	case acp.EventTypeRuntimeWarning:
		return runtimeWarningMarkerText(parsed)
	case sessionStoppedEventType:
		return sessionStoppedMarkerText(parsed)
	default:
		return ""
	}
}

func runtimeWarningMarkerText(parsed event) string {
	detail := firstNonEmpty(
		parsed.Text,
		runtimeActivityDetail(parsed.Runtime),
	)
	combined := strings.ToLower(strings.Join([]string{
		parsed.Text,
		runtimeActivityKind(parsed.Runtime),
		runtimeActivityDetail(parsed.Runtime),
	}, " "))
	switch {
	case strings.Contains(combined, "timeout") ||
		strings.Contains(combined, "timed out") ||
		strings.Contains(combined, "deadline exceeded"):
		return markerWithDetail(timeoutMarker, "Runtime activity timed out.", detail)
	case strings.Contains(combined, "unhealthy") ||
		strings.Contains(combined, string(store.SessionStallReasonProcessUnhealthy)) ||
		strings.Contains(combined, "health check failed"):
		return markerWithDetail(unhealthyMarker, "Runtime health check failed.", detail)
	default:
		return ""
	}
}

func sessionStoppedMarkerText(parsed event) string {
	failure := parsed.Failure
	reason := strings.TrimSpace(parsed.StopReason)
	summary := firstNonEmpty(parsed.Error, failureSummary(failure))
	switch {
	case reason == string(store.StopUserCanceled) || failureKind(failure) == store.FailureCanceled:
		return markerWithDetail(interruptMarker, "Session interrupted by operator.", summary)
	case reason == string(store.StopTimeout) || failureKind(failure) == store.FailureTimeout:
		return markerWithDetail(timeoutMarker, "Session timed out.", summary)
	default:
		return ""
	}
}

func markerWithDetail(marker string, fallback string, detail string) string {
	trimmed := strings.TrimSpace(detail)
	if trimmed == "" {
		trimmed = fallback
	}
	return marker + " " + trimmed
}

func runtimeActivityKind(activity *acp.RuntimeActivity) string {
	if activity == nil {
		return ""
	}
	return activity.LastActivityKind
}

func runtimeActivityDetail(activity *acp.RuntimeActivity) string {
	if activity == nil {
		return ""
	}
	return activity.LastActivityDetail
}

func failureKind(failure *store.SessionFailure) store.FailureKind {
	if failure == nil {
		return ""
	}
	return failure.Normalize().Kind
}

func failureSummary(failure *store.SessionFailure) string {
	if failure == nil {
		return ""
	}
	return failure.Normalize().Summary
}
