package network

import (
	"encoding/json"
	"reflect"
	"testing"

	sessionpkg "github.com/pedronauck/agh/internal/session"
)

func TestParseWhoisCapabilityDiscoveryRequestCapabilityFilterPresence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		ext               ExtensionMap
		wantCapabilityIDs []string
		wantNil           bool
	}{
		{
			name: "filter absent",
			ext: ExtensionMap{
				whoisIncludeExtKey: mustRawJSON(t, []string{whoisCapabilityCatalogIncludeItem}),
			},
			wantNil: true,
		},
		{
			name: "filter malformed json",
			ext: ExtensionMap{
				whoisIncludeExtKey:       mustRawJSON(t, []string{whoisCapabilityCatalogIncludeItem}),
				whoisCapabilityIDsExtKey: json.RawMessage(`{"id":"review-pr"}`),
			},
			wantCapabilityIDs: []string{},
		},
		{
			name: "filter empty list",
			ext: ExtensionMap{
				whoisIncludeExtKey:       mustRawJSON(t, []string{whoisCapabilityCatalogIncludeItem}),
				whoisCapabilityIDsExtKey: mustRawJSON(t, []string{}),
			},
			wantCapabilityIDs: []string{},
		},
		{
			name: "filter blank values",
			ext: ExtensionMap{
				whoisIncludeExtKey:       mustRawJSON(t, []string{whoisCapabilityCatalogIncludeItem}),
				whoisCapabilityIDsExtKey: mustRawJSON(t, []string{" ", "\n"}),
			},
			wantCapabilityIDs: []string{},
		},
		{
			name: "filter normalized values",
			ext: ExtensionMap{
				whoisIncludeExtKey:       mustRawJSON(t, []string{whoisCapabilityCatalogIncludeItem}),
				whoisCapabilityIDsExtKey: mustRawJSON(t, []string{" review-pr ", "draft-spec"}),
			},
			wantCapabilityIDs: []string{"review-pr", "draft-spec"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			request := parseWhoisCapabilityDiscoveryRequest(tc.ext)
			if !request.includeCapabilityCatalog {
				t.Fatal("includeCapabilityCatalog = false, want true")
			}
			if tc.wantNil {
				if request.capabilityIDs != nil {
					t.Fatalf("capabilityIDs = %#v, want nil", request.capabilityIDs)
				}
				return
			}
			if request.capabilityIDs == nil {
				t.Fatal("capabilityIDs = nil, want explicit slice")
			}
			if !reflect.DeepEqual(request.capabilityIDs, tc.wantCapabilityIDs) {
				t.Fatalf("capabilityIDs = %#v, want %#v", request.capabilityIDs, tc.wantCapabilityIDs)
			}
		})
	}
}

func TestProjectWhoisCapabilityCatalogDistinguishesAbsentAndExplicitEmptyFilters(t *testing.T) {
	t.Parallel()

	catalog := []sessionpkg.NetworkPeerCapability{
		{ID: "review-pr", Summary: "Review pull requests", Outcome: "Actionable review findings"},
		{ID: "draft-spec", Summary: "Draft technical specifications", Outcome: "Reviewed implementation plan"},
	}

	tests := []struct {
		name          string
		capabilityIDs []string
		want          []whoisCapabilityCatalogEntry
	}{
		{
			name:          "filter absent returns full catalog",
			capabilityIDs: nil,
			want: []whoisCapabilityCatalogEntry{
				{ID: "review-pr", Summary: "Review pull requests", Outcome: "Actionable review findings"},
				{ID: "draft-spec", Summary: "Draft technical specifications", Outcome: "Reviewed implementation plan"},
			},
		},
		{
			name:          "explicit empty filter returns empty projection",
			capabilityIDs: []string{},
			want:          []whoisCapabilityCatalogEntry{},
		},
		{
			name:          "filter values returns matching entries in catalog order",
			capabilityIDs: []string{"draft-spec"},
			want: []whoisCapabilityCatalogEntry{
				{ID: "draft-spec", Summary: "Draft technical specifications", Outcome: "Reviewed implementation plan"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := projectWhoisCapabilityCatalog(catalog, tc.capabilityIDs)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("projectWhoisCapabilityCatalog() = %#v, want %#v", got, tc.want)
			}
		})
	}
}
