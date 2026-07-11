package spec

func networkConversationListQueryParams(label string, includePeer bool) []ParameterSpec {
	parameters := []ParameterSpec{
		queryParam("query", "Filter "+label+" by title, peer, or preview text", false),
		boolQueryParam("has_work", "Filter by presence of open work"),
		queryParam("sort", "Order by recent_activity, created, or alphabetical", false),
		queryParam("after", "Return rows after the specified stable id cursor", false),
		intQueryParam("limit", "Maximum number of "+label+" to return"),
	}
	if !includePeer {
		return parameters
	}
	return append(
		[]ParameterSpec{queryParam("peer_id", "Filter by participant peer id", false)},
		parameters...,
	)
}

func networkConversationMessageQueryParams() []ParameterSpec {
	return []ParameterSpec{
		queryParam("before", "Return messages before the specified message id", false),
		queryParam("after", "Return messages after the specified message id", false),
		queryParam("kind", "Filter messages by network kind", false),
		queryParam("work_id", "Filter messages by work id", false),
		intQueryParam("limit", "Maximum number of messages to return"),
	}
}
