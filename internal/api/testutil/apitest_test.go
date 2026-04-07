package testutil_test

import (
	"testing"

	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
)

func TestStubSessionManagerSatisfiesInterface(_ *testing.T) {
	var _ core.SessionManager = testutil.StubSessionManager{}
}
