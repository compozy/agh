package contract

import "slices"

var slackRequiredScopesByEvent = map[string][]string{
	"app_mention":      {"app_mentions:read"},
	"message.channels": {"channels:history"},
	"message.groups":   {"groups:history"},
	"message.im":       {"im:history"},
	"message.mpim":     {"mpim:history"},
	"reaction_added":   {"reactions:read"},
	"reaction_removed": {"reactions:read"},
}

var slackAdditionalBotScopes = []string{
	"assistant:write",
	"chat:write",
	"commands",
	"reactions:write",
}

// SlackBotScopes returns the canonical required bot scopes.
func SlackBotScopes() []string {
	unique := make(map[string]struct{}, len(slackAdditionalBotScopes)+len(slackRequiredScopesByEvent))
	for _, scope := range slackAdditionalBotScopes {
		unique[scope] = struct{}{}
	}
	for _, scopes := range slackRequiredScopesByEvent {
		for _, scope := range scopes {
			unique[scope] = struct{}{}
		}
	}
	result := make([]string, 0, len(unique))
	for scope := range unique {
		result = append(result, scope)
	}
	slices.Sort(result)
	return result
}
