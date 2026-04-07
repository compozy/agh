package apitest_test

import (
	"testing"

	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/apitest"
)

func TestStubSessionManagerSatisfiesInterface(_ *testing.T) {
	var _ core.SessionManager = apitest.StubSessionManager{}
}
