package daemon

import (
	"context"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/network"
)

func (n *daemonNativeTools) resolveNativeNetworkPeerSessionID(
	ctx context.Context,
	workspaceID string,
	channel string,
	peerID string,
) (string, error) {
	wanted := strings.TrimSpace(peerID)
	if wanted == "" {
		return "", nil
	}
	if err := network.ValidatePeerID(wanted); err != nil {
		return "", err
	}
	peers, err := n.deps.Network.ListPeers(ctx, workspaceID, channel)
	if err != nil {
		return "", err
	}
	for _, peer := range peers {
		if strings.TrimSpace(peer.PeerID) != wanted || peer.SessionID == nil {
			continue
		}
		if sessionID := strings.TrimSpace(*peer.SessionID); sessionID != "" {
			return sessionID, nil
		}
	}
	return "", fmt.Errorf(
		"%w: peer_id=%q channel=%q has no participating session",
		network.ErrTargetPeerNotFound,
		wanted,
		channel,
	)
}
