package bridges

import "errors"

// DeliveryMetrics returns a point-in-time snapshot of all broker telemetry.
func (b *Broker) DeliveryMetrics() map[string]BridgeDeliveryMetrics {
	if b == nil {
		return nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	ids := make([]string, 0, len(b.metrics)+len(b.bridgeRoutes))
	seen := make(map[string]struct{}, len(b.metrics)+len(b.bridgeRoutes))
	for bridgeInstanceID := range b.metrics {
		seen[bridgeInstanceID] = struct{}{}
		ids = append(ids, bridgeInstanceID)
	}
	for bridgeInstanceID := range b.bridgeRoutes {
		if _, exists := seen[bridgeInstanceID]; exists {
			continue
		}
		ids = append(ids, bridgeInstanceID)
	}
	return b.deliveryMetricsForIDsLocked(ids)
}

// DeliveryMetricsFor returns telemetry for one bounded set of bridge IDs.
func (b *Broker) DeliveryMetricsFor(
	bridgeInstanceIDs []string,
) (map[string]BridgeDeliveryMetrics, error) {
	if b == nil {
		return nil, errors.New("bridges: delivery broker is required")
	}
	ids, err := normalizeBoundedBridgeInstanceIDs(bridgeInstanceIDs)
	if err != nil {
		return nil, err
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	return b.deliveryMetricsForIDsLocked(ids), nil
}

func (b *Broker) deliveryMetricsForIDsLocked(ids []string) map[string]BridgeDeliveryMetrics {
	snapshot := make(map[string]BridgeDeliveryMetrics, len(ids))
	for _, bridgeInstanceID := range ids {
		metrics := b.metrics[bridgeInstanceID]
		if metrics != nil {
			clonedReasons := make(map[string]int, len(metrics.droppedByReason))
			totalDropped := 0
			for reason, count := range metrics.droppedByReason {
				clonedReasons[reason] = count
				totalDropped += count
			}
			snapshot[bridgeInstanceID] = BridgeDeliveryMetrics{
				BridgeInstanceID:        bridgeInstanceID,
				DeliveryDroppedTotal:    totalDropped,
				DeliveryDroppedByReason: clonedReasons,
				DeliveryFailuresTotal:   metrics.deliveryFailuresTotal,
				LastError:               metrics.lastError,
				LastErrorAt:             metrics.lastErrorAt,
				LastSuccessAt:           metrics.lastSuccessAt,
			}
		}
		for _, route := range b.bridgeRoutes[bridgeInstanceID] {
			if route == nil {
				continue
			}
			entry := snapshot[bridgeInstanceID]
			entry.BridgeInstanceID = bridgeInstanceID
			entry.DeliveryBacklog += len(route.queue)
			snapshot[bridgeInstanceID] = entry
		}
	}
	return snapshot
}
