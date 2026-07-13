package daemon

import "github.com/compozy/agh/internal/store"

func networkAvailabilityStoreDependency(registry Registry) store.NetworkAvailabilityStore {
	availability, ok := registry.(store.NetworkAvailabilityStore)
	if !ok {
		return nil
	}
	return availability
}
