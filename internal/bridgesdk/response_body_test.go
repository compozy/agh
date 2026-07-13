// Suite: provider HTTP response cleanup
// Invariant: every response body is drained and closed, preserving simultaneous cleanup failures.
// Boundary IN: provider-owned HTTP response bodies after transport completion.
// Boundary OUT: provider request construction, status classification, and retry policy.
package bridgesdk

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

func TestDrainAndCloseHTTPResponseBodyPreservesCleanupFailures(t *testing.T) {
	t.Parallel()

	t.Run("Should drain unread bytes before closing the response body", func(t *testing.T) {
		t.Parallel()

		body := &responseBodyProbe{reader: strings.NewReader("provider response")}
		if err := DrainAndCloseHTTPResponseBody(body); err != nil {
			t.Fatalf("DrainAndCloseHTTPResponseBody() error = %v", err)
		}
		if got, want := body.bytesRead, len("provider response"); got != want {
			t.Fatalf("bytes read = %d, want %d", got, want)
		}
		if !body.closed {
			t.Fatal("response body closed = false, want true")
		}
	})

	t.Run("Should join drain and close failures", func(t *testing.T) {
		t.Parallel()

		drainErr := errors.New("drain failed")
		closeErr := errors.New("close failed")
		body := &responseBodyProbe{
			reader:   iotest.ErrReader(drainErr),
			closeErr: closeErr,
		}
		err := DrainAndCloseHTTPResponseBody(body)
		if !errors.Is(err, drainErr) {
			t.Fatalf("DrainAndCloseHTTPResponseBody() error = %v, want drain failure", err)
		}
		if !errors.Is(err, closeErr) {
			t.Fatalf("DrainAndCloseHTTPResponseBody() error = %v, want close failure", err)
		}
		if !body.closed {
			t.Fatal("response body closed = false after drain failure, want true")
		}
	})

	t.Run("Should accept an absent response body", func(t *testing.T) {
		t.Parallel()

		if err := DrainAndCloseHTTPResponseBody(nil); err != nil {
			t.Fatalf("DrainAndCloseHTTPResponseBody(nil) error = %v", err)
		}
	})

	t.Run("Should preserve operation and cleanup failures before commit", func(t *testing.T) {
		t.Parallel()

		operationErr := errors.New("decode failed")
		closeErr := errors.New("close failed")
		reports := 0
		err := FinalizeHTTPResponseBody(
			&responseBodyProbe{reader: strings.NewReader(""), closeErr: closeErr},
			operationErr,
			HTTPResponseCommitUnconfirmed,
			func(error) { reports++ },
		)
		if !errors.Is(err, operationErr) || !errors.Is(err, closeErr) {
			t.Fatalf("FinalizeHTTPResponseBody() error = %v, want operation and cleanup failures", err)
		}
		if reports != 0 {
			t.Fatalf("cleanup reports = %d, want 0", reports)
		}
	})

	t.Run("Should preserve cleanup without commit evidence", func(t *testing.T) {
		t.Parallel()

		closeErr := errors.New("close failed")
		reports := 0
		err := FinalizeHTTPResponseBody(
			&responseBodyProbe{reader: strings.NewReader(""), closeErr: closeErr},
			nil,
			HTTPResponseCommitUnconfirmed,
			func(error) { reports++ },
		)
		if !errors.Is(err, closeErr) {
			t.Fatalf("FinalizeHTTPResponseBody() error = %v, want cleanup failure", err)
		}
		if reports != 0 {
			t.Fatalf("cleanup reports = %d, want 0", reports)
		}
	})

	t.Run("Should retain committed cleanup failure when no reporter is available", func(t *testing.T) {
		t.Parallel()

		closeErr := errors.New("close failed")
		err := FinalizeHTTPResponseBody(
			&responseBodyProbe{reader: strings.NewReader(""), closeErr: closeErr},
			nil,
			HTTPResponseCommittedByMaterializedResult,
			nil,
		)
		if !errors.Is(err, closeErr) {
			t.Fatalf("FinalizeHTTPResponseBody() error = %v, want cleanup failure", err)
		}
	})
}

type responseBodyProbe struct {
	reader    io.Reader
	closeErr  error
	bytesRead int
	closed    bool
}

func (b *responseBodyProbe) Read(buffer []byte) (int, error) {
	read, err := b.reader.Read(buffer)
	b.bytesRead += read
	return read, err
}

func (b *responseBodyProbe) Close() error {
	b.closed = true
	return b.closeErr
}
