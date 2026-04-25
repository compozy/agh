package memory

import (
	"context"
	"os"
	"strings"
	"testing"
)

type testContextRefResolver struct{}

func (testContextRefResolver) Resolve(
	context.Context,
	[]ContextRef,
	TokenBudget,
) (ResolvedContext, error) {
	return ResolvedContext{}, nil
}

type testProviderHookRunner struct{}

func (testProviderHookRunner) RunMemoryHook(
	context.Context,
	ProviderHookRequest,
) (ProviderHookResult, error) {
	return ProviderHookResult{}, nil
}

var (
	_ ContextRefResolver = testContextRefResolver{}
	_ ProviderHookRunner = testProviderHookRunner{}
)

func TestFutureInterfacesRemainOutOfRuntimePromptAssembly(t *testing.T) {
	t.Parallel()

	for _, filename := range []string{"assembler.go", "recall.go"} {
		content, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) error = %v", filename, err)
		}
		source := string(content)
		for _, forbidden := range []string{
			"ContextRefResolver",
			"ProviderHookRunner",
			"RunMemoryHook",
			"Resolve(ctx context.Context, refs []ContextRef",
		} {
			if strings.Contains(source, forbidden) {
				t.Fatalf(
					"%s references future interface %q; Task 07 must not wire prompt integration",
					filename,
					forbidden,
				)
			}
		}
	}
}
