package session

// StopCause records why a session stop was initiated.
type StopCause int

const (
	CauseNone StopCause = iota
	CauseCompleted
	CauseUserRequested
	CauseShutdown
	CauseHookDenied
	CauseProcessExited
)
