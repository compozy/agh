package sessiondb

import "github.com/compozy/agh/internal/store"

func groupedTurnHistory(events []store.SessionEvent, query store.EventQuery) []store.TurnHistory {
	turns := make([]store.TurnHistory, 0)
	indexByTurnID := make(map[string]int, len(events))
	for _, event := range events {
		if idx, ok := indexByTurnID[event.TurnID]; ok {
			turns[idx].Events = append(turns[idx].Events, event)
			continue
		}

		indexByTurnID[event.TurnID] = len(turns)
		turns = append(turns, store.TurnHistory{
			TurnID: event.TurnID,
			Events: []store.SessionEvent{event},
		})
	}
	return limitTurnHistory(filterTurnHistoryAfterSequence(turns, query.AfterSequence), query.Limit)
}

func filterTurnHistoryAfterSequence(turns []store.TurnHistory, afterSequence int64) []store.TurnHistory {
	if afterSequence <= 0 {
		return turns
	}
	filtered := make([]store.TurnHistory, 0, len(turns))
	for _, turn := range turns {
		minSequence, maxSequence := turnSequenceBounds(turn.Events)
		if maxSequence <= afterSequence {
			continue
		}
		if minSequence <= afterSequence {
			continue
		}
		filtered = append(filtered, turn)
	}
	return filtered
}

func turnSequenceBounds(events []store.SessionEvent) (int64, int64) {
	if len(events) == 0 {
		return 0, 0
	}
	minSequence := events[0].Sequence
	maxSequence := events[0].Sequence
	for _, event := range events[1:] {
		if event.Sequence < minSequence {
			minSequence = event.Sequence
		}
		if event.Sequence > maxSequence {
			maxSequence = event.Sequence
		}
	}
	return minSequence, maxSequence
}
