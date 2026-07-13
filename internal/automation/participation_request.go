package automation

import "github.com/compozy/agh/internal/network/participation"

func cloneParticipationRequest(request *participation.Request) *participation.Request {
	return participation.CloneRequest(request)
}
