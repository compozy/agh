package automation

import modelpkg "github.com/pedronauck/agh/internal/automation/model"

var (
	// ErrJobNotFound reports that the requested automation job does not exist.
	ErrJobNotFound = modelpkg.ErrJobNotFound
	// ErrTriggerNotFound reports that the requested automation trigger does not exist.
	ErrTriggerNotFound = modelpkg.ErrTriggerNotFound
	// ErrRunNotFound reports that the requested automation run does not exist.
	ErrRunNotFound = modelpkg.ErrRunNotFound
	// ErrRunAlreadyExists reports that an automation run identity has already been claimed.
	ErrRunAlreadyExists = modelpkg.ErrRunAlreadyExists
	// ErrSchedulerStateNotFound reports that no durable scheduler cursor exists for a job.
	ErrSchedulerStateNotFound = modelpkg.ErrSchedulerStateNotFound
	// ErrScheduledFireAlreadyClaimed reports that a scheduled fire identity was already claimed.
	ErrScheduledFireAlreadyClaimed = modelpkg.ErrScheduledFireAlreadyClaimed
	// ErrJobNameTaken reports a duplicate job name within the same automation scope.
	ErrJobNameTaken = modelpkg.ErrJobNameTaken
	// ErrTriggerNameTaken reports a duplicate trigger name within the same automation scope.
	ErrTriggerNameTaken = modelpkg.ErrTriggerNameTaken
	// ErrTriggerWebhookIDTaken reports a duplicate stable webhook identifier.
	ErrTriggerWebhookIDTaken = modelpkg.ErrTriggerWebhookIDTaken
	// ErrOverlayRequiresConfigSource reports that enabled overlays only apply to TOML-backed definitions.
	ErrOverlayRequiresConfigSource = modelpkg.ErrOverlayRequiresConfigSource
	// ErrJobOverlayNotFound reports that a job enabled overlay row does not exist.
	ErrJobOverlayNotFound = modelpkg.ErrJobOverlayNotFound
	// ErrTriggerOverlayNotFound reports that a trigger enabled overlay row does not exist.
	ErrTriggerOverlayNotFound = modelpkg.ErrTriggerOverlayNotFound
)

// JobListQuery filters persisted automation job listings.
type JobListQuery = modelpkg.JobListQuery

// TriggerListQuery filters persisted automation trigger listings.
type TriggerListQuery = modelpkg.TriggerListQuery

// RunQuery filters automation run history and fire-limit window lookups.
type RunQuery = modelpkg.RunQuery

// JobEnabledOverlay stores the runtime enabled override for a config-backed job.
type JobEnabledOverlay = modelpkg.JobEnabledOverlay

// TriggerEnabledOverlay stores the runtime enabled override for a config-backed trigger.
type TriggerEnabledOverlay = modelpkg.TriggerEnabledOverlay
