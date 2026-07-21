package udsapi

import (
	"github.com/compozy/agh/internal/api/core"
	mcppkg "github.com/compozy/agh/internal/mcp"
)

type udsExtendedServices struct {
	resources    core.ResourceService
	extensions   ExtensionService
	hostedMCP    *mcppkg.HostedService
	mcpHostAPI   mcppkg.HostAPIInvoker
	desktopState core.DesktopStateService
}

// WithDesktopStateService injects the daemon-owned desktop-state engine.
func WithDesktopStateService(service core.DesktopStateService) Option {
	return func(server *Server) {
		server.desktopState = service
	}
}
