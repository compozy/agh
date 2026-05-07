package cli

import (
	"context"
	"strings"
	"testing"
)

func TestProviderModelClientRequiresProviderID(t *testing.T) {
	t.Parallel()

	client := &unixSocketClient{}

	t.Run("Should reject blank provider IDs for refresh", func(t *testing.T) {
		t.Parallel()

		_, err := client.RefreshProviderModels(context.Background(), "  ", ProviderModelRefreshRequest{})
		if err == nil {
			t.Fatal("RefreshProviderModels(blank provider) error = nil, want validation error")
		}
		if !strings.Contains(err.Error(), "provider_id is required") {
			t.Fatalf("RefreshProviderModels(blank provider) error = %v, want provider_id validation", err)
		}
	})

	t.Run("Should reject blank provider IDs for status", func(t *testing.T) {
		t.Parallel()

		_, err := client.ProviderModelStatus(context.Background(), " ")
		if err == nil {
			t.Fatal("ProviderModelStatus(blank provider) error = nil, want validation error")
		}
		if !strings.Contains(err.Error(), "provider_id is required") {
			t.Fatalf("ProviderModelStatus(blank provider) error = %v, want provider_id validation", err)
		}
	})
}
