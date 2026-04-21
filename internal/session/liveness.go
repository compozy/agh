package session

import (
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/procutil"
	"github.com/pedronauck/agh/internal/store"
)

const (
	// DefaultLivenessStallAfter defines how long a session may go without ACP
	// activity before recovery marks it stalled.
	DefaultLivenessStallAfter = 2 * time.Minute

	resumeStopDetailAgentOrphaned = "daemon exited while session subprocess remained alive"
	resumeStopDetailAgentStalled  = "daemon exited while stalled session subprocess remained alive"
)

// ClassifyInactiveMetaForRecovery rewrites non-terminal persisted session
// metadata into a stopped view that captures the best available supervision
// evidence after daemon interruption.
func ClassifyInactiveMetaForRecovery(now time.Time, meta store.SessionMeta) (store.SessionMeta, bool) {
	next := meta

	switch strings.TrimSpace(meta.State) {
	case string(StateActive):
		next.State = string(StateStopped)
		next.StopReason = resumeStopReasonPointer(store.StopAgentCrashed)
		next.StopDetail = classifyInterruptedStopDetail(meta, now, resumeStopDetailAgentCrashed)
		markInterruptedStall(&next, now)
		return next, sessionMetaChanged(meta, next)
	case string(StateStopping):
		next.State = string(StateStopped)
		next.StopReason = resumeStopReasonPointer(store.StopAgentCrashed)
		next.StopDetail = classifyInterruptedStopDetail(meta, now, "stop did not complete")
		markInterruptedStall(&next, now)
		return next, sessionMetaChanged(meta, next)
	case string(StateStarting):
		next.State = string(StateStopped)
		next.StopReason = resumeStopReasonPointer(store.StopError)
		next.StopDetail = classifyInterruptedStopDetail(meta, now, resumeStopDetailStartIncomplete)
		next.ACPSessionID = nil
		markInterruptedStall(&next, now)
		return next, sessionMetaChanged(meta, next)
	case string(StateStopped):
		if strings.TrimSpace(meta.StopDetail) == resumeStopDetailStartIncomplete && meta.ACPSessionID != nil {
			next.ACPSessionID = nil
			return next, sessionMetaChanged(meta, next)
		}
		return next, false
	default:
		return next, false
	}
}

func classifyPreviousStop(meta store.SessionMeta) (store.SessionMeta, bool) {
	return ClassifyInactiveMetaForRecovery(time.Now().UTC(), meta)
}

func classifyInterruptedStopDetail(meta store.SessionMeta, now time.Time, fallback string) string {
	switch {
	case sessionMetaIsStalled(meta, now):
		return resumeStopDetailAgentStalled
	case sessionMetaOwnsLiveSubprocess(meta):
		return resumeStopDetailAgentOrphaned
	default:
		return fallback
	}
}

func markInterruptedStall(meta *store.SessionMeta, now time.Time) {
	if meta == nil {
		return
	}
	if !sessionMetaIsStalled(*meta, now) {
		if meta.Liveness != nil {
			meta.Liveness.StallState = ""
			meta.Liveness.StallReason = ""
		}
		return
	}
	if meta.Liveness == nil {
		meta.Liveness = &store.SessionLivenessMeta{}
	}
	meta.Liveness.StallState = store.SessionStallStateDetected
	if strings.TrimSpace(meta.Liveness.StallReason) == "" {
		meta.Liveness.StallReason = store.SessionStallReasonActivityTimeout
	}
}

func sessionMetaIsStalled(meta store.SessionMeta, now time.Time) bool {
	if !sessionMetaOwnsLiveSubprocess(meta) {
		return false
	}
	if meta.Liveness == nil {
		return false
	}
	if strings.TrimSpace(meta.Liveness.StallState) == store.SessionStallStateDetected {
		return true
	}
	if meta.Liveness.LastUpdateAt == nil || meta.Liveness.LastUpdateAt.IsZero() || now.IsZero() {
		return false
	}
	return now.UTC().Sub(meta.Liveness.LastUpdateAt.UTC()) >= DefaultLivenessStallAfter
}

func sessionMetaOwnsLiveSubprocess(meta store.SessionMeta) bool {
	if meta.Liveness == nil || meta.Liveness.SubprocessPID <= 0 {
		return false
	}
	return procutil.Alive(meta.Liveness.SubprocessPID)
}

func sessionMetaChanged(before store.SessionMeta, after store.SessionMeta) bool {
	return before.State != after.State ||
		before.StopDetail != after.StopDetail ||
		sessionMetaStopReason(before) != sessionMetaStopReason(after) ||
		stringValue(before.ACPSessionID) != stringValue(after.ACPSessionID) ||
		!sessionLivenessEqual(before.Liveness, after.Liveness)
}

func sessionLivenessEqual(left *store.SessionLivenessMeta, right *store.SessionLivenessMeta) bool {
	lhs := store.CloneSessionLivenessMeta(left)
	rhs := store.CloneSessionLivenessMeta(right)
	switch {
	case lhs == nil && rhs == nil:
		return true
	case lhs == nil || rhs == nil:
		return false
	}
	return lhs.SubprocessPID == rhs.SubprocessPID &&
		timesEqual(lhs.SubprocessStartedAt, rhs.SubprocessStartedAt) &&
		timesEqual(lhs.LastUpdateAt, rhs.LastUpdateAt) &&
		lhs.StallState == rhs.StallState &&
		lhs.StallReason == rhs.StallReason
}

func timesEqual(left *time.Time, right *time.Time) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return left.UTC().Equal(right.UTC())
	}
}

// AnnotateUnpersistedRecovery appends persistence failure detail to the
// recovered stopped-state metadata returned to callers when the synthetic crash
// classification could not be durably written.
func AnnotateUnpersistedRecovery(meta store.SessionMeta, err error) store.SessionMeta {
	annotated := meta
	detail := strings.TrimSpace(meta.StopDetail)
	suffix := fmt.Sprintf("classification not persisted: %v", err)
	if detail == "" {
		annotated.StopDetail = suffix
		return annotated
	}
	if strings.Contains(detail, suffix) {
		return annotated
	}
	annotated.StopDetail = detail + " (" + suffix + ")"
	return annotated
}
