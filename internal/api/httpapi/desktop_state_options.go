package httpapi

import "github.com/compozy/agh/internal/api/core"

type httpExtendedServices struct {
	resources    core.ResourceService
	extensions   ExtensionService
	desktopState core.DesktopStateService
}

// WithDesktopStateService injects the daemon-owned desktop-state engine.
func WithDesktopStateService(service core.DesktopStateService) Option {
	return func(server *Server) {
		server.desktopState = service
	}
}
