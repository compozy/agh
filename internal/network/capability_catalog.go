package network

import (
	"encoding/json"
	"fmt"
	"strings"

	sessionpkg "github.com/pedronauck/agh/internal/session"
)

const (
	whoisIncludeExtKey                = "agh.include"
	whoisCapabilityIDsExtKey          = "agh.capability_ids"
	whoisCapabilityCatalogExtKey      = "agh.capability_catalog"
	whoisCapabilityCatalogIncludeItem = "capability_catalog"
	maxProtocolEnvelopeBytes          = 1 << 20
)

type whoisCapabilityDiscoveryRequest struct {
	includeCapabilityCatalog bool
	capabilityIDs            []string
}

type whoisCapabilityCatalogPayload struct {
	Capabilities []whoisCapabilityCatalogEntry `json:"capabilities"`
}

type whoisCapabilityCatalogEntry struct {
	ID                string   `json:"id"`
	Summary           string   `json:"summary"`
	Outcome           string   `json:"outcome"`
	Version           string   `json:"version,omitempty"`
	Digest            string   `json:"digest,omitempty"`
	ContextNeeded     []string `json:"context_needed,omitempty"`
	ArtifactsExpected []string `json:"artifacts_expected,omitempty"`
	ExecutionOutline  []string `json:"execution_outline,omitempty"`
	Constraints       []string `json:"constraints,omitempty"`
	Examples          []string `json:"examples,omitempty"`
	Requirements      []string `json:"requirements,omitempty"`
}

func parseWhoisCapabilityDiscoveryRequest(ext ExtensionMap) whoisCapabilityDiscoveryRequest {
	includeValues := decodeExtensionStringList(ext, whoisIncludeExtKey)
	request := whoisCapabilityDiscoveryRequest{
		includeCapabilityCatalog: containsString(includeValues, whoisCapabilityCatalogIncludeItem),
	}
	if !request.includeCapabilityCatalog {
		return request
	}

	if _, ok := ext[whoisCapabilityIDsExtKey]; !ok {
		return request
	}

	request.capabilityIDs = decodeExtensionStringList(ext, whoisCapabilityIDsExtKey)
	if request.capabilityIDs == nil {
		request.capabilityIDs = []string{}
	}
	return request
}

func buildWhoisCapabilityCatalogResponseExt(
	request whoisCapabilityDiscoveryRequest,
	capabilityCatalog []sessionpkg.NetworkPeerCapability,
) (ExtensionMap, error) {
	if !request.includeCapabilityCatalog {
		return nil, nil
	}

	payload := whoisCapabilityCatalogPayload{
		Capabilities: projectWhoisCapabilityCatalog(capabilityCatalog, request.capabilityIDs),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("network: marshal whois capability catalog: %w", err)
	}

	return ExtensionMap{
		whoisCapabilityCatalogExtKey: raw,
	}, nil
}

func projectWhoisCapabilityCatalog(
	capabilityCatalog []sessionpkg.NetworkPeerCapability,
	capabilityIDs []string,
) []whoisCapabilityCatalogEntry {
	if len(capabilityCatalog) == 0 {
		return []whoisCapabilityCatalogEntry{}
	}
	if capabilityIDs != nil && len(capabilityIDs) == 0 {
		return []whoisCapabilityCatalogEntry{}
	}

	filter := make(map[string]struct{}, len(capabilityIDs))
	for _, capabilityID := range capabilityIDs {
		trimmed := strings.TrimSpace(capabilityID)
		if trimmed == "" {
			continue
		}
		filter[trimmed] = struct{}{}
	}

	entries := make([]whoisCapabilityCatalogEntry, 0, len(capabilityCatalog))
	for _, capability := range capabilityCatalog {
		id := strings.TrimSpace(capability.ID)
		if id == "" {
			continue
		}
		if len(filter) > 0 {
			if _, ok := filter[id]; !ok {
				continue
			}
		}

		entries = append(entries, whoisCapabilityCatalogEntry{
			ID:                id,
			Summary:           strings.TrimSpace(capability.Summary),
			Outcome:           strings.TrimSpace(capability.Outcome),
			Version:           strings.TrimSpace(capability.Version),
			Digest:            strings.TrimSpace(capability.Digest),
			ContextNeeded:     cloneStringList(capability.ContextNeeded),
			ArtifactsExpected: cloneStringList(capability.ArtifactsExpected),
			ExecutionOutline:  cloneStringList(capability.ExecutionOutline),
			Constraints:       cloneStringList(capability.Constraints),
			Examples:          cloneStringList(capability.Examples),
			Requirements:      cloneStringList(capability.Requirements),
		})
	}
	if len(entries) == 0 {
		return []whoisCapabilityCatalogEntry{}
	}

	return entries
}

func cloneNetworkPeerCapabilityCatalog(
	capabilityCatalog []sessionpkg.NetworkPeerCapability,
) []sessionpkg.NetworkPeerCapability {
	if len(capabilityCatalog) == 0 {
		return []sessionpkg.NetworkPeerCapability{}
	}

	cloned := make([]sessionpkg.NetworkPeerCapability, 0, len(capabilityCatalog))
	for _, capability := range capabilityCatalog {
		cloned = append(cloned, sessionpkg.NetworkPeerCapability{
			ID:                strings.TrimSpace(capability.ID),
			Summary:           strings.TrimSpace(capability.Summary),
			Outcome:           strings.TrimSpace(capability.Outcome),
			Version:           strings.TrimSpace(capability.Version),
			Digest:            strings.TrimSpace(capability.Digest),
			ContextNeeded:     cloneStringList(capability.ContextNeeded),
			ArtifactsExpected: cloneStringList(capability.ArtifactsExpected),
			ExecutionOutline:  cloneStringList(capability.ExecutionOutline),
			Constraints:       cloneStringList(capability.Constraints),
			Examples:          cloneStringList(capability.Examples),
			Requirements:      cloneStringList(capability.Requirements),
		})
	}
	return cloned
}

func decodeWhoisCapabilityCatalogResponseExt(
	ext ExtensionMap,
) ([]sessionpkg.NetworkPeerCapability, bool) {
	if len(ext) == 0 {
		return nil, false
	}

	raw, ok := ext[whoisCapabilityCatalogExtKey]
	if !ok || len(raw) == 0 {
		return nil, false
	}

	var payload whoisCapabilityCatalogPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, false
	}

	catalog := make([]sessionpkg.NetworkPeerCapability, 0, len(payload.Capabilities))
	for _, capability := range payload.Capabilities {
		id := strings.TrimSpace(capability.ID)
		if id == "" {
			continue
		}

		catalog = append(catalog, sessionpkg.NetworkPeerCapability{
			ID:                id,
			Summary:           strings.TrimSpace(capability.Summary),
			Outcome:           strings.TrimSpace(capability.Outcome),
			Version:           strings.TrimSpace(capability.Version),
			Digest:            strings.TrimSpace(capability.Digest),
			ContextNeeded:     cloneStringList(capability.ContextNeeded),
			ArtifactsExpected: cloneStringList(capability.ArtifactsExpected),
			ExecutionOutline:  cloneStringList(capability.ExecutionOutline),
			Constraints:       cloneStringList(capability.Constraints),
			Examples:          cloneStringList(capability.Examples),
			Requirements:      cloneStringList(capability.Requirements),
		})
	}

	return catalog, true
}

func capabilityCatalogAlignsWithCapabilityIDs(
	capabilityIDs []string,
	capabilityCatalog []sessionpkg.NetworkPeerCapability,
) bool {
	normalizedIDs := normalizeCapabilityIDList(capabilityIDs)
	if len(normalizedIDs) != len(capabilityCatalog) {
		return false
	}

	for idx, capability := range capabilityCatalog {
		if normalizedIDs[idx] != strings.TrimSpace(capability.ID) {
			return false
		}
	}
	return true
}

func normalizeCapabilityIDList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}

	return normalized
}

func decodeExtensionStringList(ext ExtensionMap, key string) []string {
	if len(ext) == 0 {
		return nil
	}

	raw, ok := ext[key]
	if !ok || len(raw) == 0 {
		return nil
	}

	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	return normalizeStringList(values)
}

func ensureEnvelopeSizeLimit(envelope Envelope) error {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("network: marshal envelope for size check: %w", err)
	}
	if len(payload) > maxProtocolEnvelopeBytes {
		return fmt.Errorf(
			"%w: envelope size %d exceeds protocol limit %d",
			ErrEnvelopeTooLarge,
			len(payload),
			maxProtocolEnvelopeBytes,
		)
	}
	return nil
}
