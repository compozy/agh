package bridges

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	redactpkg "github.com/compozy/agh/internal/redact"
)

const sessionStoppedDeliveryMessage = "session stopped before delivery completed"

// ReconcileDelivery fails one persisted unfinished delivery open with a
// universal terminal post so append-only adapters cannot acknowledge silently.
func (b *Broker) ReconcileDelivery(
	ctx context.Context,
	record DeliveryLedgerRecord,
	extensionName string,
) error {
	if b == nil {
		return errors.New("bridges: delivery broker is required")
	}
	if ctx == nil {
		return errors.New("bridges: delivery reconciliation context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if b.ledgerStore == nil {
		return errors.New("bridges: delivery ledger store is required")
	}
	normalized, err := record.Canonicalize()
	if err != nil {
		return err
	}
	if normalized.State != DeliveryLedgerStateActive {
		return fmt.Errorf(
			"bridges: reconcile delivery %q state %q: %w",
			normalized.DeliveryID,
			normalized.State,
			ErrDeliveryLedgerConflict,
		)
	}
	extensionName = strings.TrimSpace(extensionName)
	if extensionName == "" {
		return errors.New("bridges: delivery reconciliation extension name is required")
	}

	request, checkpoint, metrics, err := b.failOpenReconciliation(normalized)
	if err != nil {
		return err
	}
	transport := b.currentTransport()
	if transport == nil {
		return ErrDeliveryTransportUnavailable
	}
	callCtx, cancel := context.WithTimeout(ctx, b.requestTimeout)
	ack, err := transport.DeliverBridge(callCtx, extensionName, request)
	cancel()
	if err != nil {
		return fmt.Errorf("bridges: reconcile delivery %q: %w", normalized.DeliveryID, err)
	}
	if err := ack.ValidateFor(request.Event); err != nil {
		return fmt.Errorf("bridges: validate reconciled delivery %q acknowledgement: %w", normalized.DeliveryID, err)
	}
	normalizedAck := ack.normalize()
	if normalizedAck.RemoteMessageID != "" {
		checkpoint.RemoteMessageID = normalizedAck.RemoteMessageID
	}
	checkpoint.Metrics = &metrics
	if err := b.persistDeliveryCheckpoint(ctx, checkpoint); err != nil {
		return err
	}

	b.mu.Lock()
	b.applyDeliveryMetricRecordLocked(metrics)
	b.mu.Unlock()
	return nil
}

func (b *Broker) failOpenReconciliation(
	record DeliveryLedgerRecord,
) (DeliveryRequest, DeliveryLedgerCheckpoint, DeliveryMetricRecord, error) {
	target := DeliveryTarget{
		BridgeInstanceID: record.BridgeInstanceID,
		PeerID:           record.RoutingKey.PeerID,
		ThreadID:         record.RoutingKey.ThreadID,
		GroupID:          record.RoutingKey.GroupID,
		Mode:             DeliveryModeReply,
	}
	if err := target.Validate(); err != nil {
		return DeliveryRequest{}, DeliveryLedgerCheckpoint{}, DeliveryMetricRecord{}, err
	}
	seq := max(record.LastSentSeq, record.LastAckedSeq) + 1
	updatedAt := b.now()
	content := MessageContent{Text: sessionStoppedDeliveryMessage}
	event := DeliveryEvent{
		DeliveryID:       record.DeliveryID,
		BridgeInstanceID: record.BridgeInstanceID,
		RoutingKey:       record.RoutingKey,
		DeliveryTarget:   target,
		Seq:              seq,
		EventType:        DeliveryEventTypeError,
		Content:          content,
		Final:            true,
		Operation:        DeliveryOperationPost,
		Error:            &DeliveryErrorDetail{Message: sessionStoppedDeliveryMessage},
	}
	request := DeliveryRequest{Event: event}
	if err := request.Validate(); err != nil {
		return DeliveryRequest{}, DeliveryLedgerCheckpoint{}, DeliveryMetricRecord{}, err
	}

	metrics := b.reconciliationMetricRecord(record, updatedAt)
	checkpoint := DeliveryLedgerCheckpoint{
		DeliveryID:      record.DeliveryID,
		State:           DeliveryLedgerStateTerminalError,
		LastSentSeq:     seq,
		LastAckedSeq:    seq,
		RemoteMessageID: record.RemoteMessageID,
		TerminalError:   sessionStoppedDeliveryMessage,
		UpdatedAt:       updatedAt,
	}
	return request, checkpoint, metrics, nil
}

func (b *Broker) reconciliationMetricRecord(
	record DeliveryLedgerRecord,
	updatedAt time.Time,
) DeliveryMetricRecord {
	b.mu.Lock()
	defer b.mu.Unlock()
	metrics := b.metricsLocked(record.BridgeInstanceID)
	droppedReasons := cloneDeliveryDropReasons(metrics.droppedByReason)
	droppedTotal := 0
	for _, count := range droppedReasons {
		droppedTotal += count
	}
	return DeliveryMetricRecord{
		BridgeDeliveryMetrics: BridgeDeliveryMetrics{
			BridgeInstanceID:        record.BridgeInstanceID,
			DeliveryDroppedTotal:    droppedTotal,
			DeliveryDroppedByReason: droppedReasons,
			DeliveryFailuresTotal:   metrics.deliveryFailuresTotal + 1,
			LastError:               sessionStoppedDeliveryMessage,
			LastErrorAt:             updatedAt,
			LastSuccessAt:           metrics.lastSuccessAt,
		},
		Scope:       record.Scope,
		WorkspaceID: record.WorkspaceID,
		UpdatedAt:   updatedAt,
	}
}

func (b *Broker) applyDeliveryMetricRecordLocked(record DeliveryMetricRecord) {
	metrics := b.metricsLocked(record.BridgeInstanceID)
	if metrics == nil {
		return
	}
	metrics.scope = record.Scope
	metrics.workspaceID = record.WorkspaceID
	metrics.droppedByReason = cloneDeliveryDropReasons(record.DeliveryDroppedByReason)
	metrics.deliveryFailuresTotal = record.DeliveryFailuresTotal
	metrics.lastError = record.LastError
	metrics.lastErrorAt = record.LastErrorAt
	metrics.lastSuccessAt = record.LastSuccessAt
	metrics.updatedAt = record.UpdatedAt
	metrics.persistedRevision = metrics.revision
}

func terminalFailureContent(current MessageContent, reason string) MessageContent {
	safeReason := redactpkg.String(strings.TrimSpace(reason))
	if safeReason == "" {
		safeReason = sessionStoppedDeliveryMessage
	}
	currentText := strings.TrimSpace(current.Text)
	if currentText == "" {
		return MessageContent{Text: safeReason}
	}
	return MessageContent{Text: currentText + "\n\n" + safeReason}
}
