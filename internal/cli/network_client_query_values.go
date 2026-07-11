package cli

import (
	"net/url"
	"strconv"
	"strings"
)

func networkThreadsValues(query NetworkThreadsQuery) url.Values {
	return networkConversationListValues(
		query.Limit,
		query.After,
		query.Query,
		query.PeerID,
		query.Sort,
		query.HasWork,
	)
}

func networkDirectsValues(query NetworkDirectsQuery) url.Values {
	return networkConversationListValues(
		query.Limit,
		query.After,
		query.Query,
		query.PeerID,
		query.Sort,
		query.HasWork,
	)
}

func networkConversationListValues(
	limit int,
	after string,
	search string,
	peerID string,
	sortOrder string,
	hasWork *bool,
) url.Values {
	values := networkListValues(limit, after)
	if trimmed := strings.TrimSpace(search); trimmed != "" {
		values.Set("query", trimmed)
	}
	if trimmed := strings.TrimSpace(peerID); trimmed != "" {
		values.Set("peer_id", trimmed)
	}
	if trimmed := strings.TrimSpace(sortOrder); trimmed != "" {
		values.Set("sort", trimmed)
	}
	if hasWork != nil {
		values.Set("has_work", strconv.FormatBool(*hasWork))
	}
	return values
}
