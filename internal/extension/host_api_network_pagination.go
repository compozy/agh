package extensionpkg

import (
	"context"
	"strings"

	apicontract "github.com/compozy/agh/internal/api/contract"
	extensioncontract "github.com/compozy/agh/internal/extension/contract"
	"github.com/compozy/agh/internal/store"
)

func hostAPINetworkThreadQuery(params extensioncontract.NetworkThreadsParams) (store.NetworkThreadQuery, error) {
	query := store.NetworkThreadQuery{
		Search:  strings.TrimSpace(params.Query),
		PeerID:  strings.TrimSpace(params.PeerID),
		Sort:    strings.TrimSpace(params.Sort),
		HasWork: params.HasWork,
		Limit:   params.Limit,
		After:   strings.TrimSpace(params.After),
	}
	if err := query.Validate(); err != nil {
		return store.NetworkThreadQuery{}, invalidParamsRPCError(err)
	}
	return query, nil
}

func hostAPINetworkDirectRoomQuery(
	params extensioncontract.NetworkDirectsParams,
) (store.NetworkDirectRoomQuery, error) {
	query := store.NetworkDirectRoomQuery{
		Search:  strings.TrimSpace(params.Query),
		PeerID:  strings.TrimSpace(params.PeerID),
		Sort:    strings.TrimSpace(params.Sort),
		HasWork: params.HasWork,
		Limit:   params.Limit,
		After:   strings.TrimSpace(params.After),
	}
	if err := query.Validate(); err != nil {
		return store.NetworkDirectRoomQuery{}, invalidParamsRPCError(err)
	}
	return query, nil
}

func hostAPINetworkConversationMessageQuery(
	limit int,
	before string,
	after string,
	kind string,
	workID string,
) (store.NetworkConversationMessageQuery, error) {
	query := store.NetworkConversationMessageQuery{
		BeforeMessageID: strings.TrimSpace(before),
		AfterMessageID:  strings.TrimSpace(after),
		Kind:            strings.TrimSpace(kind),
		WorkID:          strings.TrimSpace(workID),
		Limit:           limit,
	}
	if err := query.Validate(); err != nil {
		return store.NetworkConversationMessageQuery{}, invalidParamsRPCError(err)
	}
	query.Limit = store.NormalizeNetworkMessageLimit(query.Limit) + 1
	return query, nil
}

func (h *HostAPIHandler) hostAPINetworkConversationMessages(
	ctx context.Context,
	ref store.NetworkConversationRef,
	query store.NetworkConversationMessageQuery,
) ([]apicontract.NetworkConversationMessagePayload, apicontract.CursorPagePayload, error) {
	networkStore, err := h.requireHostAPINetworkStore()
	if err != nil {
		return nil, apicontract.CursorPagePayload{}, err
	}
	ref.WorkspaceID = strings.TrimSpace(ref.WorkspaceID)
	ref.Channel = strings.TrimSpace(ref.Channel)
	ref.ThreadID = strings.TrimSpace(ref.ThreadID)
	ref.DirectID = strings.TrimSpace(ref.DirectID)
	if err := ref.Validate(); err != nil {
		return nil, apicontract.CursorPagePayload{}, invalidParamsRPCError(err)
	}
	messages, err := networkStore.ListConversationMessages(ctx, ref, query)
	if err != nil {
		return nil, apicontract.CursorPagePayload{}, mapHostAPINetworkRPCError(err)
	}
	payload := hostAPINetworkConversationMessagePayloads(messages)
	limit := query.Limit - 1
	payload, page := trimHostAPINetworkMessagePage(
		payload,
		query.AfterMessageID,
		limit,
	)
	return payload, page, nil
}

func trimHostAPINetworkMessagePage(
	messages []apicontract.NetworkConversationMessagePayload,
	after string,
	limit int,
) ([]apicontract.NetworkConversationMessagePayload, apicontract.CursorPagePayload) {
	page := apicontract.CursorPagePayload{Limit: limit}
	if len(messages) <= limit {
		return messages, page
	}
	page.HasMore = true
	if strings.TrimSpace(after) != "" {
		messages = messages[:limit]
		page.NextCursor = strings.TrimSpace(messages[len(messages)-1].MessageID)
		return messages, page
	}
	messages = messages[len(messages)-limit:]
	page.NextCursor = strings.TrimSpace(messages[0].MessageID)
	return messages, page
}
