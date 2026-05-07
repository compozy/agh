package modelcatalog

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/testutil"
)

func TestMergeRows(t *testing.T) {
	t.Parallel()

	t.Run("Should let higher priority source win conflicting fields", func(t *testing.T) {
		t.Parallel()

		contextWindowConfig := int64(100)
		contextWindowCatalog := int64(200)
		models := MergeRows([]ModelRow{
			testRow(
				"models_dev",
				SourceKindModelsDev,
				PriorityModelsDev,
				"codex",
				"gpt-5.4",
				testTime(0),
				func(row *ModelRow) {
					row.DisplayName = "Catalog GPT"
					row.ContextWindow = &contextWindowCatalog
				},
			),
			testRow("config", SourceKindConfig, PriorityConfig, "codex", "gpt-5.4", testTime(0), func(row *ModelRow) {
				row.DisplayName = "Config GPT"
				row.ContextWindow = &contextWindowConfig
			}),
		})

		model := requireSingleModel(t, models)
		if model.DisplayName != "Config GPT" {
			t.Fatalf("DisplayName = %q, want Config GPT", model.DisplayName)
		}
		if model.ContextWindow == nil || *model.ContextWindow != contextWindowConfig {
			t.Fatalf("ContextWindow = %v, want %d", model.ContextWindow, contextWindowConfig)
		}
	})

	t.Run("Should let provider live priority win over extension priority", func(t *testing.T) {
		t.Parallel()

		liveAvailable := true
		extensionAvailable := false
		models := MergeRows([]ModelRow{
			testRow(
				"extension:alpha",
				SourceKindExtension,
				PriorityExtension,
				"codex",
				"gpt-5.4",
				testTime(0),
				func(row *ModelRow) {
					row.DisplayName = "Extension GPT"
					row.Available = &extensionAvailable
				},
			),
			testRow(
				"provider_live:codex",
				SourceKindProviderLive,
				PriorityProviderLive,
				"codex",
				"gpt-5.4",
				testTime(0),
				func(row *ModelRow) {
					row.DisplayName = "Live GPT"
					row.Available = &liveAvailable
				},
			),
		})

		model := requireSingleModel(t, models)
		if model.DisplayName != "Live GPT" {
			t.Fatalf("DisplayName = %q, want Live GPT", model.DisplayName)
		}
		if model.Available == nil || !*model.Available {
			t.Fatalf("Available = %v, want true", model.Available)
		}
		if model.AvailabilityState != string(AvailabilityStateAvailableLive) {
			t.Fatalf("AvailabilityState = %q, want available_live", model.AvailabilityState)
		}
	})

	t.Run("Should resolve equal priority and freshness by ascending source id", func(t *testing.T) {
		t.Parallel()

		models := MergeRows([]ModelRow{
			testRow(
				"extension:b",
				SourceKindExtension,
				PriorityExtension,
				"codex",
				"gpt-5.4",
				testTime(0),
				func(row *ModelRow) {
					row.DisplayName = "B Source"
				},
			),
			testRow(
				"extension:a",
				SourceKindExtension,
				PriorityExtension,
				"codex",
				"gpt-5.4",
				testTime(0),
				func(row *ModelRow) {
					row.DisplayName = "A Source"
				},
			),
		})

		model := requireSingleModel(t, models)
		if model.DisplayName != "A Source" {
			t.Fatalf("DisplayName = %q, want A Source", model.DisplayName)
		}
		if got, want := model.Sources[0].SourceID, "extension:a"; got != want {
			t.Fatalf("Sources[0].SourceID = %q, want %q", got, want)
		}
	})

	t.Run("Should let lower priority source fill missing metadata", func(t *testing.T) {
		t.Parallel()

		contextWindow := int64(256000)
		costInput := 1.25
		models := MergeRows([]ModelRow{
			testRow("config", SourceKindConfig, PriorityConfig, "codex", "gpt-5.4", testTime(0), nil),
			testRow(
				"models_dev",
				SourceKindModelsDev,
				PriorityModelsDev,
				"codex",
				"gpt-5.4",
				testTime(0),
				func(row *ModelRow) {
					row.DisplayName = "Catalog GPT"
					row.ContextWindow = &contextWindow
					row.CostInputPerMillion = &costInput
				},
			),
		})

		model := requireSingleModel(t, models)
		if model.DisplayName != "Catalog GPT" {
			t.Fatalf("DisplayName = %q, want Catalog GPT", model.DisplayName)
		}
		if model.ContextWindow == nil || *model.ContextWindow != contextWindow {
			t.Fatalf("ContextWindow = %v, want %d", model.ContextWindow, contextWindow)
		}
		if model.CostInputPerMillion == nil || *model.CostInputPerMillion != costInput {
			t.Fatalf("CostInputPerMillion = %v, want %f", model.CostInputPerMillion, costInput)
		}
	})

	t.Run("Should project merged availability states", func(t *testing.T) {
		t.Parallel()

		for _, tc := range []struct {
			name      string
			available bool
			stale     bool
			state     AvailabilityState
		}{
			{name: "Should project stale available live truth", available: true, stale: true, state: AvailabilityStateAvailableStale},
			{name: "Should project fresh available live truth", available: true, stale: false, state: AvailabilityStateAvailableLive},
			{name: "Should project stale unavailable live truth", available: false, stale: true, state: AvailabilityStateUnavailableStale},
			{name: "Should project fresh unavailable live truth", available: false, stale: false, state: AvailabilityStateUnavailableLive},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				models := MergeRows([]ModelRow{
					testRow(
						"provider_live:codex",
						SourceKindProviderLive,
						PriorityProviderLive,
						"codex",
						"gpt-5.4",
						testTime(0),
						func(row *ModelRow) {
							row.Available = &tc.available
							row.Stale = tc.stale
						},
					),
				})
				model := requireSingleModel(t, models)
				if model.Available == nil || *model.Available != tc.available {
					t.Fatalf("Available = %v, want %t", model.Available, tc.available)
				}
				if model.AvailabilityState != string(tc.state) {
					t.Fatalf("AvailabilityState = %q, want %q", model.AvailabilityState, tc.state)
				}
			})
		}
	})

	t.Run("Should keep catalog only models at unknown availability", func(t *testing.T) {
		t.Parallel()

		models := MergeRows([]ModelRow{
			testRow("models_dev", SourceKindModelsDev, PriorityModelsDev, "codex", "gpt-5.4", testTime(0), nil),
		})
		model := requireSingleModel(t, models)
		if model.Available != nil {
			t.Fatalf("Available = %v, want nil", model.Available)
		}
		if model.AvailabilityState != string(AvailabilityStateUnknown) {
			t.Fatalf("AvailabilityState = %q, want unknown", model.AvailabilityState)
		}
	})

	t.Run("Should sort merged projection and source refs deterministically", func(t *testing.T) {
		t.Parallel()

		models := MergeRows([]ModelRow{
			testRow("extension:b", SourceKindExtension, PriorityExtension, "claude", "claude-4", testTime(1), nil),
			testRow(
				"provider_live:codex",
				SourceKindProviderLive,
				PriorityProviderLive,
				"codex",
				"gpt-5.4",
				testTime(0),
				nil,
			),
			testRow("extension:a", SourceKindExtension, PriorityExtension, "codex", "gpt-5.4", testTime(2), nil),
		})
		if got, want := modelKeys(models), []string{"claude/claude-4", "codex/gpt-5.4"}; !slices.Equal(got, want) {
			t.Fatalf("model keys = %#v, want %#v", got, want)
		}
		if got, want := sourceIDs(
			models[1].Sources,
		), []string{
			"provider_live:codex",
			"extension:a",
		}; !slices.Equal(
			got,
			want,
		) {
			t.Fatalf("source ids = %#v, want %#v", got, want)
		}
	})
}

func TestCatalogServiceRefresh(t *testing.T) {
	t.Parallel()

	t.Run("Should respect stale filters when listing merged models", func(t *testing.T) {
		t.Parallel()

		store := newMemoryStore()
		store.rows[sourceProviderKey("models_dev", "codex")] = []ModelRow{
			testRow(
				"models_dev",
				SourceKindModelsDev,
				PriorityModelsDev,
				"codex",
				"gpt-5.4",
				testTime(0),
				func(row *ModelRow) {
					row.Stale = true
				},
			),
		}
		service := newTestService(t, store, nil)

		models, err := service.ListModels(
			testutil.Context(t),
			ListOptions{ProviderID: "codex", Now: testTime(1)},
		)
		if err != nil {
			t.Fatalf("ListModels(exclude stale) error = %v", err)
		}
		if len(models) != 0 {
			t.Fatalf("ListModels(exclude stale) = %#v, want empty projection", models)
		}

		models, err = service.ListModels(
			testutil.Context(t),
			ListOptions{ProviderID: "codex", IncludeStale: true, Now: testTime(1)},
		)
		if err != nil {
			t.Fatalf("ListModels(include stale) error = %v", err)
		}
		model := requireSingleModel(t, models)
		if !model.Stale {
			t.Fatalf("Model.Stale = %t, want true", model.Stale)
		}
	})

	t.Run("Should return partial success and record failed source status", func(t *testing.T) {
		t.Parallel()

		store := newMemoryStore()
		service := newTestService(t, store, []Source{
			&fakeSource{
				id:        "config",
				kind:      SourceKindConfig,
				priority:  PriorityConfig,
				providers: []string{"codex"},
				rows: []ModelRow{
					testRow("config", SourceKindConfig, PriorityConfig, "codex", "gpt-5.4", testTime(0), nil),
				},
			},
			&fakeSource{
				id:        "models_dev",
				kind:      SourceKindModelsDev,
				priority:  PriorityModelsDev,
				providers: []string{"codex"},
				err:       fmt.Errorf("upstream failed with api_key=super-secret"),
			},
		})

		models, err := service.ListModels(
			testutil.Context(t),
			ListOptions{ProviderID: "codex", Refresh: true, Now: testTime(10)},
		)
		if err != nil {
			t.Fatalf("ListModels(refresh) error = %v", err)
		}
		if got, want := modelKeys(models), []string{"codex/gpt-5.4"}; !slices.Equal(got, want) {
			t.Fatalf("model keys = %#v, want %#v", got, want)
		}
		statuses, err := service.ListSourceStatus(testutil.Context(t), "codex")
		if err != nil {
			t.Fatalf("ListSourceStatus() error = %v", err)
		}
		failed := requireStatus(t, statuses, "models_dev")
		if failed.RefreshState != string(RefreshStateFailed) {
			t.Fatalf("RefreshState = %q, want failed", failed.RefreshState)
		}
		if strings.Contains(failed.LastError, "super-secret") || !strings.Contains(failed.LastError, "[REDACTED]") {
			t.Fatalf("LastError = %q, want redacted secret", failed.LastError)
		}
	})

	t.Run("Should fail all source failure when no stale rows exist", func(t *testing.T) {
		t.Parallel()

		store := newMemoryStore()
		service := newTestService(t, store, []Source{
			&fakeSource{
				id:        "models_dev",
				kind:      SourceKindModelsDev,
				priority:  PriorityModelsDev,
				providers: []string{"codex"},
				err:       errors.New("models.dev down"),
			},
		})

		_, err := service.ListModels(
			testutil.Context(t),
			ListOptions{ProviderID: "codex", Refresh: true, Now: testTime(0)},
		)
		if !errors.Is(err, ErrAllSourcesFailed) {
			t.Fatalf("ListModels() error = %v, want ErrAllSourcesFailed", err)
		}
	})

	t.Run("Should return stale rows when refresh fails after prior success", func(t *testing.T) {
		t.Parallel()

		available := true
		source := &fakeSource{
			id:        "provider_live:codex",
			kind:      SourceKindProviderLive,
			priority:  PriorityProviderLive,
			providers: []string{"codex"},
			rows: []ModelRow{
				testRow(
					"provider_live:codex",
					SourceKindProviderLive,
					PriorityProviderLive,
					"codex",
					"gpt-5.4",
					testTime(0),
					func(row *ModelRow) {
						row.Available = &available
					},
				),
			},
		}
		store := newMemoryStore()
		service := newTestService(t, store, []Source{source})
		if _, err := service.ListModels(
			testutil.Context(t),
			ListOptions{ProviderID: "codex", Refresh: true, Now: testTime(1)},
		); err != nil {
			t.Fatalf("ListModels(first refresh) error = %v", err)
		}

		source.rows = nil
		source.err = errors.New("live source unavailable sk-secret-token")
		models, err := service.ListModels(
			testutil.Context(t),
			ListOptions{ProviderID: "codex", Refresh: true, Now: testTime(2)},
		)
		if err != nil {
			t.Fatalf("ListModels(exclude stale refresh) error = %v", err)
		}
		if len(models) != 0 {
			t.Fatalf("ListModels(exclude stale refresh) = %#v, want empty projection", models)
		}

		models, err = service.ListModels(
			testutil.Context(t),
			ListOptions{ProviderID: "codex", Refresh: true, IncludeStale: true, Now: testTime(2)},
		)
		if err != nil {
			t.Fatalf("ListModels(include stale refresh) error = %v", err)
		}
		model := requireSingleModel(t, models)
		if model.AvailabilityState != string(AvailabilityStateAvailableStale) {
			t.Fatalf("AvailabilityState = %q, want available_stale", model.AvailabilityState)
		}
		if !model.Stale {
			t.Fatal("Model.Stale = false, want true")
		}
		if strings.Contains(model.LastError, "sk-secret-token") || !strings.Contains(model.LastError, "[REDACTED]") {
			t.Fatalf("LastError = %q, want redacted stale error", model.LastError)
		}
		statuses, err := service.ListSourceStatus(testutil.Context(t), "codex")
		if err != nil {
			t.Fatalf("ListSourceStatus() error = %v", err)
		}
		status := requireStatus(t, statuses, "provider_live:codex")
		if !status.LastSuccess.Equal(testTime(1)) {
			t.Fatalf("LastSuccess = %s, want first refresh time %s", status.LastSuccess, testTime(1))
		}
	})

	t.Run("Should reject invalid extension source id before persistence", func(t *testing.T) {
		t.Parallel()

		store := newMemoryStore()
		_, err := NewService(store, []Source{
			&fakeSource{id: "extension:BadSlug", kind: SourceKindExtension, priority: PriorityExtension},
		})
		if err == nil {
			t.Fatal("NewService(invalid extension source) error = nil, want validation error")
		}
		if store.replaceCount != 0 {
			t.Fatalf("replaceCount = %d, want 0", store.replaceCount)
		}
	})
}

func TestCatalogServiceRefreshConcurrency(t *testing.T) {
	t.Parallel()

	t.Run("Should coalesce concurrent refreshes for the same provider scope", func(t *testing.T) {
		t.Parallel()

		source := newBlockingRefreshSource(map[string][]ModelRow{
			"codex": {
				testRow(
					"provider_live:codex",
					SourceKindProviderLive,
					PriorityProviderLive,
					"codex",
					"gpt-5.4",
					testTime(30),
					nil,
				),
			},
		})
		store := newMemoryStore()
		service := newTestService(t, store, []Source{source})
		ctx := testutil.Context(t)

		results := make(chan refreshTestResult, 2)
		for range 2 {
			go func() {
				statuses, err := service.Refresh(ctx, RefreshOptions{
					ProviderID: "codex",
					SourceID:   source.ID(),
					Force:      true,
					Now:        testTime(30),
				})
				results <- refreshTestResult{statuses: statuses, err: err}
			}()
		}
		source.waitForCalls(t, 1)
		source.requireCallCountStable(t, 1, 25*time.Millisecond)
		source.release()

		for range 2 {
			result := <-results
			if result.err != nil {
				t.Fatalf("Refresh() error = %v", result.err)
			}
			if got, want := len(result.statuses), 1; got != want {
				t.Fatalf("len(statuses) = %d, want %d: %#v", got, want, result.statuses)
			}
		}
	})

	t.Run("Should let concurrent refreshes across providers replace rows deterministically", func(t *testing.T) {
		t.Parallel()

		source := newBlockingRefreshSource(map[string][]ModelRow{
			"claude": {
				testRow(
					"provider_live:shared",
					SourceKindProviderLive,
					PriorityProviderLive,
					"claude",
					"claude-sonnet-4-6",
					testTime(31),
					nil,
				),
			},
			"codex": {
				testRow(
					"provider_live:shared",
					SourceKindProviderLive,
					PriorityProviderLive,
					"codex",
					"gpt-5.4",
					testTime(31),
					nil,
				),
			},
		})
		store := newMemoryStore()
		service := newTestService(t, store, []Source{source})
		ctx := testutil.Context(t)

		results := make(chan refreshTestResult, 2)
		for _, providerID := range []string{"codex", "claude"} {
			go func(providerID string) {
				statuses, err := service.Refresh(ctx, RefreshOptions{
					ProviderID: providerID,
					SourceID:   source.ID(),
					Force:      true,
					Now:        testTime(31),
				})
				results <- refreshTestResult{statuses: statuses, err: err}
			}(providerID)
		}
		source.waitForCalls(t, 2)
		source.release()

		for range 2 {
			result := <-results
			if result.err != nil {
				t.Fatalf("Refresh() error = %v", result.err)
			}
			if got, want := len(result.statuses), 1; got != want {
				t.Fatalf("len(statuses) = %d, want %d: %#v", got, want, result.statuses)
			}
		}
		models, err := service.ListModels(
			ctx,
			ListOptions{IncludeStale: true, Now: testTime(32)},
		)
		if err != nil {
			t.Fatalf("ListModels() error = %v", err)
		}
		if got, want := modelKeys(models), []string{"claude/claude-sonnet-4-6", "codex/gpt-5.4"}; !slices.Equal(
			got,
			want,
		) {
			t.Fatalf("model keys = %#v, want %#v", got, want)
		}
		if got, want := source.callCount(), 2; got != want {
			t.Fatalf("source calls = %d, want %d cross-provider calls", got, want)
		}
	})

	t.Run("Should cancel waiters blocked on an in-flight refresh", func(t *testing.T) {
		t.Parallel()

		source := newBlockingRefreshSource(map[string][]ModelRow{
			"codex": {
				testRow(
					"provider_live:shared",
					SourceKindProviderLive,
					PriorityProviderLive,
					"codex",
					"gpt-5.4",
					testTime(33),
					nil,
				),
			},
		})
		store := newMemoryStore()
		service := newTestService(t, store, []Source{source})

		results := make(chan refreshTestResult, 2)
		go func() {
			statuses, err := service.Refresh(testutil.Context(t), RefreshOptions{
				ProviderID: "codex",
				SourceID:   source.ID(),
				Force:      true,
				Now:        testTime(33),
			})
			results <- refreshTestResult{statuses: statuses, err: err}
		}()

		source.waitForCalls(t, 1)

		waiterCtx, cancel := context.WithCancel(testutil.Context(t))
		defer cancel()
		go func() {
			statuses, err := service.Refresh(waiterCtx, RefreshOptions{
				ProviderID: "codex",
				SourceID:   source.ID(),
				Force:      true,
				Now:        testTime(33),
			})
			results <- refreshTestResult{statuses: statuses, err: err}
		}()

		source.requireCallCountStable(t, 1, 25*time.Millisecond)
		cancel()

		waiterResult := <-results
		if !errors.Is(waiterResult.err, context.Canceled) {
			t.Fatalf("waiter Refresh() error = %v, want context.Canceled", waiterResult.err)
		}

		source.release()

		ownerResult := <-results
		if ownerResult.err != nil {
			t.Fatalf("owner Refresh() error = %v", ownerResult.err)
		}
		if got, want := len(ownerResult.statuses), 1; got != want {
			t.Fatalf("len(statuses) = %d, want %d: %#v", got, want, ownerResult.statuses)
		}
	})
}

type fakeSource struct {
	id        string
	kind      SourceKind
	priority  int
	providers []string
	rows      []ModelRow
	err       error
	ttl       time.Duration
	calls     int
}

func (s *fakeSource) ID() string {
	return s.id
}

func (s *fakeSource) Kind() SourceKind {
	return s.kind
}

func (s *fakeSource) Priority() int {
	return s.priority
}

func (s *fakeSource) ProviderIDs() []string {
	return append([]string(nil), s.providers...)
}

func (s *fakeSource) TTL() time.Duration {
	return s.ttl
}

func (s *fakeSource) ListModels(_ context.Context, opts ListOptions) ([]ModelRow, error) {
	s.calls++
	rows := make([]ModelRow, 0, len(s.rows))
	for _, row := range s.rows {
		if opts.ProviderID == "" || row.ProviderID == opts.ProviderID {
			rows = append(rows, row)
		}
	}
	return rows, s.err
}

type refreshTestResult struct {
	statuses []SourceStatus
	err      error
}

type blockingRefreshSource struct {
	mu             sync.Mutex
	rowsByProvider map[string][]ModelRow
	calls          int
	callsCh        chan int
	releaseCh      chan struct{}
	releaseOnce    sync.Once
}

func newBlockingRefreshSource(rowsByProvider map[string][]ModelRow) *blockingRefreshSource {
	return &blockingRefreshSource{
		rowsByProvider: rowsByProvider,
		callsCh:        make(chan int, 16),
		releaseCh:      make(chan struct{}),
	}
}

func (s *blockingRefreshSource) ID() string {
	return "provider_live:shared"
}

func (s *blockingRefreshSource) Kind() SourceKind {
	return SourceKindProviderLive
}

func (s *blockingRefreshSource) Priority() int {
	return PriorityProviderLive
}

func (s *blockingRefreshSource) ProviderIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	providers := make([]string, 0, len(s.rowsByProvider))
	for providerID := range s.rowsByProvider {
		providers = append(providers, providerID)
	}
	slices.Sort(providers)
	return providers
}

func (s *blockingRefreshSource) TTL() time.Duration {
	return 0
}

func (s *blockingRefreshSource) ListModels(ctx context.Context, opts ListOptions) ([]ModelRow, error) {
	s.mu.Lock()
	s.calls++
	calls := s.calls
	s.mu.Unlock()
	select {
	case s.callsCh <- calls:
	default:
	}

	select {
	case <-s.releaseCh:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	s.mu.Lock()
	rows := cloneModelRows(s.rowsByProvider[opts.ProviderID])
	s.mu.Unlock()
	return rows, nil
}

func (s *blockingRefreshSource) waitForCalls(t *testing.T, want int) {
	t.Helper()

	deadline := time.After(time.Second)
	for {
		if s.callCount() >= want {
			return
		}
		select {
		case <-s.callsCh:
		case <-deadline:
			t.Fatalf("source calls = %d, want at least %d", s.callCount(), want)
		}
	}
}

func (s *blockingRefreshSource) requireCallCountStable(t *testing.T, want int, duration time.Duration) {
	t.Helper()

	timer := time.NewTimer(duration)
	defer timer.Stop()
	for {
		select {
		case <-s.callsCh:
			if got := s.callCount(); got > want {
				t.Fatalf("source calls = %d while first refresh was blocked, want at most %d", got, want)
			}
		case <-timer.C:
			return
		}
	}
}

func (s *blockingRefreshSource) release() {
	s.releaseOnce.Do(func() {
		close(s.releaseCh)
	})
}

func (s *blockingRefreshSource) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

type memoryStore struct {
	mu           sync.Mutex
	rows         map[string][]ModelRow
	statuses     map[string]SourceStatus
	replaceCount int
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		rows:     make(map[string][]ModelRow),
		statuses: make(map[string]SourceStatus),
	}
}

func (s *memoryStore) ReplaceSourceRows(
	_ context.Context,
	sourceID string,
	providerID string,
	rows []ModelRow,
	status SourceStatus,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.replaceCount++
	key := sourceProviderKey(sourceID, providerID)
	s.rows[key] = cloneModelRows(rows)
	s.statuses[key] = status
	return nil
}

func (s *memoryStore) ListRows(_ context.Context, opts ListOptions) ([]ModelRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := make([]ModelRow, 0)
	for _, group := range s.rows {
		for _, row := range group {
			if opts.ProviderID != "" && row.ProviderID != opts.ProviderID {
				continue
			}
			if opts.SourceID != "" && row.SourceID != opts.SourceID {
				continue
			}
			if row.Stale && !opts.IncludeAll && !opts.IncludeStale {
				continue
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (s *memoryStore) ListSourceStatus(_ context.Context, providerID string) ([]SourceStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	statuses := make([]SourceStatus, 0, len(s.statuses))
	for _, status := range s.statuses {
		if providerID == "" || status.ProviderID == providerID {
			statuses = append(statuses, status)
		}
	}
	return statuses, nil
}

func sourceProviderKey(sourceID string, providerID string) string {
	return sourceID + "\x00" + providerID
}

func newTestService(t *testing.T, store Store, sources []Source) *CatalogService {
	t.Helper()

	service, err := NewService(store, sources)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return service
}

func testRow(
	sourceID string,
	kind SourceKind,
	priority int,
	providerID string,
	modelID string,
	refreshedAt time.Time,
	mutate func(*ModelRow),
) ModelRow {
	row := ModelRow{
		SourceID:    sourceID,
		SourceKind:  kind,
		Priority:    priority,
		ProviderID:  providerID,
		ModelID:     modelID,
		RefreshedAt: refreshedAt,
	}
	if mutate != nil {
		mutate(&row)
	}
	return row
}

func testTime(offset int) time.Time {
	return time.Date(2026, 5, 7, 12, offset, 0, 0, time.UTC)
}

func requireSingleModel(t *testing.T, models []Model) Model {
	t.Helper()

	if len(models) != 1 {
		t.Fatalf("len(models) = %d, want 1: %#v", len(models), models)
	}
	return models[0]
}

func requireStatus(t *testing.T, statuses []SourceStatus, sourceID string) SourceStatus {
	t.Helper()

	for _, status := range statuses {
		if status.SourceID == sourceID {
			return status
		}
	}
	t.Fatalf("statuses = %#v, want source %q", statuses, sourceID)
	return SourceStatus{}
}

func modelKeys(models []Model) []string {
	keys := make([]string, 0, len(models))
	for _, model := range models {
		keys = append(keys, model.ProviderID+"/"+model.ModelID)
	}
	return keys
}

func sourceIDs(sources []SourceRef) []string {
	ids := make([]string, 0, len(sources))
	for _, source := range sources {
		ids = append(ids, source.SourceID)
	}
	return ids
}
