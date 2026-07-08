package daemon

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/compozy/agh/internal/api/contract"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/dsl"
	watchpkg "github.com/compozy/agh/internal/loop/watch"
)

func TestLoopWatchEventKindContractParity(t *testing.T) {
	t.Run("Should mirror the supported watch-events family registry exactly", func(t *testing.T) {
		t.Parallel()
		want := make([]string, 0)
		for kind := range looppkg.SupportedWatchEvents() {
			want = append(want, string(kind))
		}
		slices.Sort(want)
		got := slices.Clone(contract.LoopWatchEventKindValues())
		slices.Sort(got)
		require.Equal(t, want, got,
			"contract watch-event kinds must stay in lockstep with SupportedWatchEvents()")
	})
}

func watchEventsDefinition(nodeID, kind string) *contract.LoopDefinitionDocument {
	return &contract.LoopDefinitionDocument{
		Graph: contract.LoopGraph{
			Nodes: []contract.LoopGraphNode{
				{ID: nodeID, Class: contract.LoopNodeClassSource, Kind: kind},
			},
		},
	}
}

func TestLoopWatchEventsReadModel(t *testing.T) {
	t.Run("Should project subscriptions, cursors, and last_wake_at when parked", func(t *testing.T) {
		t.Parallel()
		ref, err := watchpkg.EventsPendingOutputRef(watchpkg.EventsPendingState{
			Subscriptions: []watchpkg.EventSubscriptionRef{
				{Kind: string(contract.LoopWatchEventTaskStatusChanged), Filter: "event.payload.to_status == 'completed'"},
			},
			Cursors: map[string]int64{"task_events": 42},
		})
		require.NoError(t, err)
		wokeAt := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
		run := looppkg.Run{Generation: 1, LastProgressAt: wokeAt}
		generations := []contract.LoopGenerationPayload{{
			Generation: 1,
			Outputs:    []contract.LoopGenerationOutput{{NodeID: "watch", Status: "running", OutputRef: ref}},
		}}
		state, err := loopWatchEventsReadModel(run, watchEventsDefinition("watch", string(dsl.SourceWatchEvents)), generations)
		require.NoError(t, err)
		require.NotNil(t, state)
		require.Equal(t, []contract.LoopWatchEventSubscription{
			{Kind: contract.LoopWatchEventTaskStatusChanged, Filter: "event.payload.to_status == 'completed'"},
		}, state.Subscriptions)
		require.Equal(t, map[string]int64{"task_events": 42}, state.Cursors)
		require.NotNil(t, state.LastWakeAt)
		require.Equal(t, wokeAt, *state.LastWakeAt)
	})
	t.Run("Should return nil when the watch-events node is not parked", func(t *testing.T) {
		t.Parallel()
		ref, err := watchpkg.EventsConfirmedOutputRef([]map[string]any{{"kind": "task.status_changed"}},
			map[string]int64{"task_events": 43})
		require.NoError(t, err)
		run := looppkg.Run{Generation: 1, LastProgressAt: time.Now()}
		generations := []contract.LoopGenerationPayload{{
			Generation: 1,
			Outputs:    []contract.LoopGenerationOutput{{NodeID: "watch", Status: "succeeded", OutputRef: ref}},
		}}
		state, err := loopWatchEventsReadModel(run, watchEventsDefinition("watch", string(dsl.SourceWatchEvents)), generations)
		require.NoError(t, err)
		require.Nil(t, state)
	})
	t.Run("Should return nil when the definition has no watch-events node", func(t *testing.T) {
		t.Parallel()
		run := looppkg.Run{Generation: 1, LastProgressAt: time.Now()}
		generations := []contract.LoopGenerationPayload{{
			Generation: 1,
			Outputs:    []contract.LoopGenerationOutput{{NodeID: "load", Status: "succeeded"}},
		}}
		state, err := loopWatchEventsReadModel(run, watchEventsDefinition("load", string(dsl.SourceFileImport)), generations)
		require.NoError(t, err)
		require.Nil(t, state)
	})
	t.Run("Should omit last_wake_at when the run never progressed", func(t *testing.T) {
		t.Parallel()
		ref, err := watchpkg.EventsPendingOutputRef(watchpkg.EventsPendingState{
			Subscriptions: []watchpkg.EventSubscriptionRef{{Kind: string(contract.LoopWatchEventLoopTerminal)}},
			Cursors:       map[string]int64{"loop_run_events": 0},
		})
		require.NoError(t, err)
		run := looppkg.Run{Generation: 1}
		generations := []contract.LoopGenerationPayload{{
			Generation: 1,
			Outputs:    []contract.LoopGenerationOutput{{NodeID: "watch", Status: "running", OutputRef: ref}},
		}}
		state, err := loopWatchEventsReadModel(run, watchEventsDefinition("watch", string(dsl.SourceWatchEvents)), generations)
		require.NoError(t, err)
		require.NotNil(t, state)
		require.Nil(t, state.LastWakeAt)
	})
	t.Run("Should surface a decode error for a corrupt watch-events park ref", func(t *testing.T) {
		t.Parallel()
		run := looppkg.Run{Generation: 1}
		generations := []contract.LoopGenerationPayload{{
			Generation: 1,
			Outputs: []contract.LoopGenerationOutput{
				{NodeID: "watch", Status: "running", OutputRef: "{not json"},
			},
		}}
		state, err := loopWatchEventsReadModel(run, watchEventsDefinition("watch", string(dsl.SourceWatchEvents)), generations)
		require.Error(t, err)
		require.Nil(t, state)
	})
}
