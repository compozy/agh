package apitest_test

import (
	"testing"

	"github.com/pedronauck/agh/internal/apicore"
	"github.com/pedronauck/agh/internal/apitest"
)

func TestStubSessionManagerSatisfiesInterface(_ *testing.T) {
	var _ apicore.SessionManager = apitest.StubSessionManager{}
}
