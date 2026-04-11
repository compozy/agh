package network

import (
	"sort"
	"sync"
)

// KindMetric is the runtime per-kind network activity snapshot surfaced by
// status APIs.
type KindMetric struct {
	Kind      Kind
	Sent      int64
	Received  int64
	Rejected  int64
	Delivered int64
}

type statsSnapshot struct {
	MessagesSent         int64
	MessagesReceived     int64
	MessagesRejected     int64
	MessagesDelivered    int64
	WorkflowTaggedEvents int64
	HandoffTaggedEvents  int64
	KindMetrics          []KindMetric
}

type runtimeStats struct {
	mu sync.Mutex

	messagesSent         int64
	messagesReceived     int64
	messagesRejected     int64
	messagesDelivered    int64
	workflowTaggedEvents int64
	handoffTaggedEvents  int64
	kindMetrics          map[Kind]*KindMetric
}

func newRuntimeStats() *runtimeStats {
	return &runtimeStats{
		kindMetrics: make(map[Kind]*KindMetric),
	}
}

func (s *runtimeStats) recordSent(envelope Envelope) {
	s.record(envelope, func(metric *KindMetric) {
		metric.Sent++
		s.messagesSent++
	})
}

func (s *runtimeStats) recordReceived(envelope Envelope) {
	s.record(envelope, func(metric *KindMetric) {
		metric.Received++
		s.messagesReceived++
	})
}

func (s *runtimeStats) recordRejected(envelope Envelope) {
	s.record(envelope, func(metric *KindMetric) {
		metric.Rejected++
		s.messagesRejected++
	})
}

func (s *runtimeStats) recordDelivered(envelope Envelope) {
	s.record(envelope, func(metric *KindMetric) {
		metric.Delivered++
		s.messagesDelivered++
	})
}

func (s *runtimeStats) snapshot() statsSnapshot {
	if s == nil {
		return statsSnapshot{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	metrics := make([]KindMetric, 0, len(s.kindMetrics))
	for _, metric := range s.kindMetrics {
		if metric == nil {
			continue
		}
		if metric.Sent == 0 && metric.Received == 0 && metric.Rejected == 0 && metric.Delivered == 0 {
			continue
		}
		metrics = append(metrics, *metric)
	}
	sort.Slice(metrics, func(i, j int) bool {
		return string(metrics[i].Kind) < string(metrics[j].Kind)
	})

	return statsSnapshot{
		MessagesSent:         s.messagesSent,
		MessagesReceived:     s.messagesReceived,
		MessagesRejected:     s.messagesRejected,
		MessagesDelivered:    s.messagesDelivered,
		WorkflowTaggedEvents: s.workflowTaggedEvents,
		HandoffTaggedEvents:  s.handoffTaggedEvents,
		KindMetrics:          metrics,
	}
}

func (s *runtimeStats) record(envelope Envelope, update func(*KindMetric)) {
	if s == nil || update == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	metric := s.kindMetricLocked(envelope.Kind)
	update(metric)

	if hasWorkflowID(envelope.Ext) {
		s.workflowTaggedEvents++
	}
	if hasHandoffVersion(envelope.Ext) {
		s.handoffTaggedEvents++
	}
}

func (s *runtimeStats) kindMetricLocked(kind Kind) *KindMetric {
	metric, ok := s.kindMetrics[kind]
	if ok && metric != nil {
		return metric
	}
	metric = &KindMetric{Kind: kind}
	s.kindMetrics[kind] = metric
	return metric
}
