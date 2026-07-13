package main

import (
	"context"

	bridgepkg "github.com/compozy/agh/internal/bridges/contract"
	"github.com/compozy/agh/internal/bridgesdk"
)

func (p *discordProvider) recordDeliveryFailure(
	ctx context.Context,
	session *bridgesdk.Session,
	instanceID string,
	request bridgepkg.DeliveryRequest,
	state deliveryState,
	marker bridgesdk.DeliveryMarker,
	err error,
) {
	if state.Chunks.Active() {
		p.storeDeliveryState(instanceID, request.Event.DeliveryID, state)
	}
	marker.Error = err.Error()
	p.markers.RecordDelivery(marker)
	p.reportDiscordDeliveryError(ctx, session, instanceID, err)
}
