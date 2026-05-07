package httpapi

import (
	"context"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	apispec "github.com/pedronauck/agh/internal/api/spec"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestHTTPAgentKernelRoutesMatchDocumentedSpecOperations(t *testing.T) {
	t.Run("Should register every HTTP agent operation in the spec", func(t *testing.T) {
		t.Parallel()

		engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, newTestHomePaths(t)))

		got := registeredHTTPAgentKernelRoutes(engine.Routes())
		want := documentedHTTPAgentKernelRoutes()
		if !slices.Equal(got, want) {
			t.Fatalf("HTTP agent routes = %v, want documented agent routes %v", got, want)
		}
	})
}

func TestServerHandlerConfigIncludesCoordinatorConfig(t *testing.T) {
	t.Run("Should carry coordinator resolver into handlers", func(t *testing.T) {
		t.Parallel()

		resolver := httpapiCoordinatorConfigResolverFunc(
			func(_ context.Context, workspaceID string) (aghconfig.CoordinatorConfig, error) {
				if workspaceID != "ws-1" {
					t.Fatalf("ResolveCoordinatorConfig() workspaceID = %q, want ws-1", workspaceID)
				}
				return aghconfig.CoordinatorConfig{AgentName: "coordinator"}, nil
			},
		)

		server := &Server{}
		WithCoordinatorConfig(resolver)(server)
		handlers := newHandlers(server.handlerConfig(nil))
		if handlers.CoordinatorConfig == nil {
			t.Fatal("handlers.CoordinatorConfig is nil, want configured resolver")
		}

		cfg, err := handlers.CoordinatorConfig.ResolveCoordinatorConfig(context.Background(), "ws-1")
		if err != nil {
			t.Fatalf("ResolveCoordinatorConfig() error = %v", err)
		}
		if got, want := cfg.AgentName, "coordinator"; got != want {
			t.Fatalf("CoordinatorConfig.AgentName = %q, want %q", got, want)
		}
	})
}

func registeredHTTPAgentKernelRoutes(routes gin.RoutesInfo) []string {
	filtered := make([]string, 0)
	for _, route := range routes {
		if isHTTPAgentKernelRoute(route.Path) {
			filtered = append(filtered, route.Method+" "+route.Path)
		}
	}
	sort.Strings(filtered)
	return filtered
}

func documentedHTTPAgentKernelRoutes() []string {
	routes := make([]string, 0)
	for _, operation := range apispec.Operations() {
		if !slices.Contains(operation.Transports, apispec.TransportHTTP) {
			continue
		}
		if !isHTTPAgentKernelRoute(operation.Path) {
			continue
		}
		routes = append(routes, operation.Method+" "+normalizeHTTPAgentSpecRoutePath(operation.Path))
	}
	sort.Strings(routes)
	return routes
}

func isHTTPAgentKernelRoute(routePath string) bool {
	return routePath == "/api/agent" || strings.HasPrefix(routePath, "/api/agent/")
}

func normalizeHTTPAgentSpecRoutePath(routePath string) string {
	parts := strings.Split(routePath, "/")
	for index, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") && len(part) > 2 {
			parts[index] = ":" + part[1:len(part)-1]
		}
	}
	return strings.Join(parts, "/")
}

type httpapiCoordinatorConfigResolverFunc func(context.Context, string) (aghconfig.CoordinatorConfig, error)

func (f httpapiCoordinatorConfigResolverFunc) ResolveCoordinatorConfig(
	ctx context.Context,
	workspaceID string,
) (aghconfig.CoordinatorConfig, error) {
	return f(ctx, workspaceID)
}
