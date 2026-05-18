package main

import (
	"context"
	"errors"
	"io"
	"testing"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
)

func TestTelegramClawpatchRegressions(t *testing.T) {
	t.Run("Should exclude invalid webhook path configs from routing", func(t *testing.T) {
		t.Parallel()

		provider := &telegramProvider{
			routes: make(map[string]resolvedInstanceConfig),
		}
		provider.swapTelegramRoutes([]resolvedInstanceConfig{
			{
				instanceID:  "brg-1",
				webhookPath: "/telegram/shared",
				configError: errors.New("duplicate webhook path"),
			},
		}, "127.0.0.1:21230")

		if cfg, ok := provider.configForPath("/telegram/shared"); ok {
			t.Fatalf("configForPath() = (%#v, true), want invalid config excluded", cfg)
		}
	})

	t.Run("Should evict terminal delivery state after final acknowledgement", func(t *testing.T) {
		t.Parallel()

		provider := &telegramProvider{
			stderr: io.Discard,
			routes: map[string]resolvedInstanceConfig{
				"brg-1": {instanceID: "brg-1"},
			},
			deliveries: make(map[string]deliveryState),
			reportedStatus: map[string]bridgepkg.BridgeStatus{
				"brg-1": bridgepkg.BridgeStatusReady,
			},
			stopCh: make(chan struct{}),
		}
		provider.apiFactory = func(resolvedInstanceConfig) telegramAPI {
			return &fakeTelegramAPI{nextMessageID: 900}
		}

		startReq := testDeliveryRequest(
			"brg-1",
			"delivery-1",
			1,
			bridgepkg.DeliveryEventTypeStart,
			false,
		)
		startAck, err := provider.handleBridgesDeliver(context.Background(), nil, startReq)
		if err != nil {
			t.Fatalf("handleBridgesDeliver(start) error = %v", err)
		}
		if got, want := provider.deliveryState(
			"brg-1",
			"delivery-1",
		).RemoteMessageID, startAck.RemoteMessageID; got != want {
			t.Fatalf("deliveryState(start).RemoteMessageID = %q, want %q", got, want)
		}

		finalReq := testDeliveryRequest(
			"brg-1",
			"delivery-1",
			2,
			bridgepkg.DeliveryEventTypeFinal,
			true,
		)
		finalReq.Event.Content.Text = "final message"
		if _, err := provider.handleBridgesDeliver(context.Background(), nil, finalReq); err != nil {
			t.Fatalf("handleBridgesDeliver(final) error = %v", err)
		}
		if got := provider.deliveryState("brg-1", "delivery-1"); got != (deliveryState{}) {
			t.Fatalf("deliveryState(final) = %#v, want empty after terminal ack", got)
		}
	})
}
