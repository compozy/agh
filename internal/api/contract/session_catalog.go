package contract

// SessionCatalogResponse wraps one bounded public session catalog page.
type SessionCatalogResponse struct {
	Sessions []SessionPayload         `json:"sessions"`
	Page     CountedCursorPagePayload `json:"page"`
}
