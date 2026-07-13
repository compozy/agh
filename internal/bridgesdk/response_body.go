package bridgesdk

import (
	"errors"
	"fmt"
	"io"
)

// HTTPResponseCommitPolicy defines which observation proves an HTTP operation committed remotely.
type HTTPResponseCommitPolicy uint8

const (
	// HTTPResponseNoCommit keeps response cleanup failures on the returned operation error.
	HTTPResponseNoCommit HTTPResponseCommitPolicy = iota
	// HTTPResponseCommitOnSuccessStatus treats a successful status as remote commit evidence.
	HTTPResponseCommitOnSuccessStatus
	// HTTPResponseCommitOnMaterializedResult requires a decoded remote result before commit.
	HTTPResponseCommitOnMaterializedResult
)

// HTTPResponseCommitEvidence records why retrying a successful HTTP operation is unsafe.
type HTTPResponseCommitEvidence uint8

const (
	// HTTPResponseCommitUnconfirmed means no remote mutation has been proven.
	HTTPResponseCommitUnconfirmed HTTPResponseCommitEvidence = iota
	// HTTPResponseCommittedBySuccessStatus means a successful status confirms the mutation.
	HTTPResponseCommittedBySuccessStatus
	// HTTPResponseCommittedByMaterializedResult means decoding produced the required remote identity.
	HTTPResponseCommittedByMaterializedResult
)

// Evidence derives observed commit evidence after a successful status and optional decoded result.
func (p HTTPResponseCommitPolicy) Evidence(materializedResult bool) HTTPResponseCommitEvidence {
	switch {
	case p == HTTPResponseCommitOnSuccessStatus:
		return HTTPResponseCommittedBySuccessStatus
	case p == HTTPResponseCommitOnMaterializedResult && materializedResult:
		return HTTPResponseCommittedByMaterializedResult
	default:
		return HTTPResponseCommitUnconfirmed
	}
}

// DrainAndCloseHTTPResponseBody drains and closes a provider response, joining both failures.
func DrainAndCloseHTTPResponseBody(body io.ReadCloser) error {
	if body == nil {
		return nil
	}

	var drainErr error
	if _, err := io.Copy(io.Discard, body); err != nil {
		drainErr = fmt.Errorf("bridgesdk: drain HTTP response body: %w", err)
	}
	var closeErr error
	if err := body.Close(); err != nil {
		closeErr = fmt.Errorf("bridgesdk: close HTTP response body: %w", err)
	}
	return errors.Join(drainErr, closeErr)
}

// FinalizeHTTPResponseBody preserves cleanup failures until remote commit evidence exists. Once
// committed, cleanup is reported without invalidating the operation. A missing reporter keeps the
// cleanup failure on the returned operation error rather than discarding it.
func FinalizeHTTPResponseBody(
	body io.ReadCloser,
	operationErr error,
	commitEvidence HTTPResponseCommitEvidence,
	reportCleanup func(error),
) error {
	cleanupErr := DrainAndCloseHTTPResponseBody(body)
	if cleanupErr == nil {
		return operationErr
	}
	if operationErr != nil || !commitEvidence.confirmed() || reportCleanup == nil {
		return errors.Join(operationErr, cleanupErr)
	}
	reportCleanup(cleanupErr)
	return nil
}

func (e HTTPResponseCommitEvidence) confirmed() bool {
	return e == HTTPResponseCommittedBySuccessStatus ||
		e == HTTPResponseCommittedByMaterializedResult
}
