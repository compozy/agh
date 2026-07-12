package bridges

import "encoding/json"

func cloneDeliveryEvent(event DeliveryEvent) DeliveryEvent {
	cloned := event.normalize()
	cloned.ProviderMetadata = cloneRawJSON(cloned.ProviderMetadata)
	if cloned.Reference != nil {
		reference := cloned.Reference.normalize()
		cloned.Reference = &reference
	}
	if cloned.Error != nil {
		errorDetail := cloned.Error.normalize()
		cloned.Error = &errorDetail
	}
	if cloned.Resume != nil {
		resume := cloned.Resume.normalize()
		cloned.Resume = &resume
	}
	cloned.Progress = cloneToolProgress(cloned.Progress)
	return cloned
}

func cloneDeliverySnapshot(snapshot DeliverySnapshot) DeliverySnapshot {
	return snapshot.normalize()
}

func cloneDeliveryRequest(req DeliveryRequest) DeliveryRequest {
	cloned := DeliveryRequest{Event: cloneDeliveryEvent(req.Event)}
	if req.Snapshot != nil {
		snapshot := cloneDeliverySnapshot(*req.Snapshot)
		cloned.Snapshot = &snapshot
	}
	return cloned
}

func cloneRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}
