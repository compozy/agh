package contract

import "time"

// SettingsMCPAuthBeginResponse returns the verifier-free PKCE handoff.
type SettingsMCPAuthBeginResponse struct {
	AuthorizationURL string    `json:"authorization_url"`
	State            string    `json:"state"`
	ExpiresAt        time.Time `json:"expires_at"`
	CallbackURL      string    `json:"callback_url"`
	ManualSupported  bool      `json:"manual_supported"`
}

// SettingsMCPAuthExchangeRequest completes one login with exactly one input.
type SettingsMCPAuthExchangeRequest struct {
	Code        string `json:"code,omitempty"`
	RedirectURL string `json:"redirect_url,omitempty"`
}
