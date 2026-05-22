package extensiontest

import (
	"testing"
	"time"

	bridgepkg "github.com/compozy/agh/internal/bridges"
)

func TestValidateConformanceDeliveryAckTrackingContract(t *testing.T) {
	t.Parallel()

	t.Run("Should keep missing ack when a later normal ack uses the same delivery id", func(t *testing.T) {
		t.Parallel()

		report := validConformanceReport()
		report.Deliveries = []DeliveryRecord{
			{
				Request: testDeliveryRequest("delivery-1", 1, bridgepkg.DeliveryEventTypeStart, false),
			},
			{
				Request: testDeliveryRequest("delivery-1", 2, bridgepkg.DeliveryEventTypeDelta, false),
				Ack:     testDeliveryAck("delivery-1", 2, "telegram:delivery-1:2", "telegram:delivery-1:1"),
			},
		}

		assertConformanceIssue(t, report, ConformanceExpectation{RequireDelivery: true}, "missing_ack")
	})

	t.Run("Should clear missing ack after an explicit resume delivery", func(t *testing.T) {
		t.Parallel()

		report := validConformanceReport()
		report.Deliveries = []DeliveryRecord{
			{
				Request: testDeliveryRequest("delivery-1", 1, bridgepkg.DeliveryEventTypeStart, false),
			},
			testResumeDeliveryRecordContract("delivery-1", 2),
		}

		if err := ValidateConformance(report, ConformanceExpectation{
			RequireDelivery: true,
			RequireResume:   true,
		}); err != nil {
			t.Fatalf("ValidateConformance() error = %v, want nil", err)
		}
	})
}

func testResumeDeliveryRecordContract(deliveryID string, seq int64) DeliveryRecord {
	request := testDeliveryRequest(deliveryID, seq, bridgepkg.DeliveryEventTypeResume, false)
	request.Event.Resume = &bridgepkg.DeliveryResumeState{LatestEventType: bridgepkg.DeliveryEventTypeStart}
	request.Snapshot = &bridgepkg.DeliverySnapshot{
		DeliveryID:       deliveryID,
		SessionID:        "sess-1",
		TurnID:           "turn-1",
		BridgeInstanceID: "brg-telegram-reference",
		RoutingKey:       request.Event.RoutingKey,
		DeliveryTarget:   request.Event.DeliveryTarget,
		LatestSeq:        seq - 1,
		LatestEventType:  bridgepkg.DeliveryEventTypeStart,
		CurrentContent:   bridgepkg.MessageContent{Text: "hello"},
		LastSentSeq:      seq - 1,
		RemoteMessageID:  "telegram:delivery-1:1",
		UpdatedAt:        time.Date(2026, 4, 11, 5, 1, 0, 0, time.UTC),
	}
	return DeliveryRecord{
		Request: request,
		Ack:     testDeliveryAck(deliveryID, seq, "telegram:delivery-1:1", ""),
	}
}
