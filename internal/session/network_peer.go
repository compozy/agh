package session

import (
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func networkPeerCapabilities(catalog *aghconfig.CapabilityCatalog) []NetworkPeerCapability {
	if catalog == nil || len(catalog.Capabilities) == 0 {
		return []NetworkPeerCapability{}
	}

	projected := make([]NetworkPeerCapability, 0, len(catalog.Capabilities))
	for _, capability := range catalog.Capabilities {
		projected = append(projected, NetworkPeerCapability{
			ID:      strings.TrimSpace(capability.ID),
			Summary: strings.TrimSpace(capability.Summary),
		})
	}
	return projected
}

func cloneNetworkPeerCapabilities(capabilities []NetworkPeerCapability) []NetworkPeerCapability {
	if len(capabilities) == 0 {
		return []NetworkPeerCapability{}
	}

	cloned := make([]NetworkPeerCapability, 0, len(capabilities))
	for _, capability := range capabilities {
		cloned = append(cloned, NetworkPeerCapability{
			ID:      strings.TrimSpace(capability.ID),
			Summary: strings.TrimSpace(capability.Summary),
		})
	}
	return cloned
}

func newNetworkPeerJoin(
	sessionID string,
	peerID string,
	channel string,
	capabilities []NetworkPeerCapability,
) NetworkPeerJoin {
	return NetworkPeerJoin{
		SessionID:    strings.TrimSpace(sessionID),
		PeerID:       strings.TrimSpace(peerID),
		Channel:      strings.TrimSpace(channel),
		Capabilities: cloneNetworkPeerCapabilities(capabilities),
	}
}
