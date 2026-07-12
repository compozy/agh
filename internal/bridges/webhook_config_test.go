// Suite: Bridge public webhook configuration
// Invariant: provider callbacks use a complete external HTTP(S) URL and never infer one from listener state.
// Boundary IN: persisted bridge provider_config JSON.
// Boundary OUT: network reachability and provider registration, owned by provider control transports.
package bridges

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWebhookPublicURL(t *testing.T) {
	tests := []struct {
		name       string
		config     string
		want       string
		wantErr    bool
		wantErrSub string
	}{
		{
			name:   "Should accept a public HTTPS callback",
			config: `{"webhook":{"public_url":"https://bridge.example.com/slack/support"}}`,
			want:   "https://bridge.example.com/slack/support",
		},
		{
			name:       "Should reject a loopback callback",
			config:     `{"webhook":{"public_url":"https://127.0.0.1:2123/slack/support"}}`,
			wantErr:    true,
			wantErrSub: "publicly routable",
		},
		{
			name:       "Should reject a private callback",
			config:     `{"webhook":{"public_url":"https://10.20.30.40/slack/support"}}`,
			wantErr:    true,
			wantErrSub: "publicly routable",
		},
		{
			name:       "Should reject a link-local metadata callback",
			config:     `{"webhook":{"public_url":"https://169.254.169.254/latest/meta-data"}}`,
			wantErr:    true,
			wantErrSub: "publicly routable",
		},
		{
			name:       "Should reject a multicast callback",
			config:     `{"webhook":{"public_url":"https://224.0.0.1/slack/support"}}`,
			wantErr:    true,
			wantErrSub: "publicly routable",
		},
		{
			name:       "Should reject a NAT64 callback",
			config:     `{"webhook":{"public_url":"https://[64:ff9b::a00:1]/slack/support"}}`,
			wantErr:    true,
			wantErrSub: "publicly routable",
		},
		{
			name:       "Should reject a local-use NAT64 callback",
			config:     `{"webhook":{"public_url":"https://[64:ff9b:1::a00:1]/slack/support"}}`,
			wantErr:    true,
			wantErrSub: "publicly routable",
		},
		{
			name:       "Should reject a 6to4 callback",
			config:     `{"webhook":{"public_url":"https://[2002:a00:1::]/slack/support"}}`,
			wantErr:    true,
			wantErrSub: "publicly routable",
		},
		{
			name:       "Should reject a Teredo callback",
			config:     `{"webhook":{"public_url":"https://[2001::1]/slack/support"}}`,
			wantErr:    true,
			wantErrSub: "publicly routable",
		},
		{
			name:       "Should reject a benchmark callback",
			config:     `{"webhook":{"public_url":"https://198.18.0.1/slack/support"}}`,
			wantErr:    true,
			wantErrSub: "publicly routable",
		},
		{
			name:       "Should reject a documentation callback",
			config:     `{"webhook":{"public_url":"https://192.0.2.1/slack/support"}}`,
			wantErr:    true,
			wantErrSub: "publicly routable",
		},
		{
			name:       "Should reject malformed provider config",
			config:     `{"webhook":`,
			wantErr:    true,
			wantErrSub: "decode provider config",
		},
		{
			name:       "Should reject a missing public URL",
			config:     `{"webhook":{"path":"/slack/support"}}`,
			wantErr:    true,
			wantErrSub: "webhook public_url is required",
		},
		{
			name:       "Should reject insecure non-loopback HTTP",
			config:     `{"webhook":{"public_url":"http://bridge.example.com/slack/support"}}`,
			wantErr:    true,
			wantErrSub: "must use HTTPS",
		},
		{
			name:       "Should reject credentials and fragments",
			config:     `{"webhook":{"public_url":"https://user@bridge.example.com/slack/support#token"}}`,
			wantErr:    true,
			wantErrSub: "must not contain userinfo or fragment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			instance := BridgeInstance{ProviderConfig: json.RawMessage(tt.config)}
			got, err := WebhookPublicURL(instance)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("WebhookPublicURL() = %q, nil; want error containing %q", got, tt.wantErrSub)
				}
				if !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Fatalf("WebhookPublicURL() error = %q, want substring %q", err, tt.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("WebhookPublicURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("WebhookPublicURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
