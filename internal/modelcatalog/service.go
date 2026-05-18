package modelcatalog

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type sourceProviderLister interface {
	ProviderIDs() []string
}

type sourceTTLProvider interface {
	TTL() time.Duration
}

// CatalogService refreshes sources and projects stored model catalog rows.
type CatalogService struct {
	store          Store
	sources        []Source
	sourceByID     map[string]Source
	lockMu         sync.Mutex
	refreshFlights map[string]*refreshFlight
	onFlightWait   func(providerID string)
}

type refreshFlight struct {
	scopeKey string
	done     chan struct{}
	statuses []SourceStatus
	err      error
}

var _ Service = (*CatalogService)(nil)

// NewService creates a model catalog service from a store and source list.
func NewService(store Store, sources []Source) (*CatalogService, error) {
	if store == nil {
		return nil, fmt.Errorf("model catalog store is required")
	}
	normalizedSources := make([]Source, 0, len(sources))
	sourceByID := make(map[string]Source, len(sources))
	for _, source := range sources {
		if source == nil {
			return nil, fmt.Errorf("model catalog source is required")
		}
		if err := ValidateSourceIdentity(source.ID(), source.Kind()); err != nil {
			return nil, err
		}
		if source.Priority() <= 0 {
			return nil, fmt.Errorf("model catalog source %q priority must be positive", source.ID())
		}
		if _, exists := sourceByID[source.ID()]; exists {
			return nil, fmt.Errorf("model catalog source %q is registered more than once", source.ID())
		}
		normalizedSources = append(normalizedSources, source)
		sourceByID[source.ID()] = source
	}
	sort.SliceStable(normalizedSources, func(i, j int) bool {
		if normalizedSources[i].Priority() != normalizedSources[j].Priority() {
			return normalizedSources[i].Priority() > normalizedSources[j].Priority()
		}
		return normalizedSources[i].ID() < normalizedSources[j].ID()
	})
	return &CatalogService{
		store:          store,
		sources:        normalizedSources,
		sourceByID:     sourceByID,
		refreshFlights: make(map[string]*refreshFlight),
	}, nil
}

// ListModels returns the merged catalog projection.
func (s *CatalogService) ListModels(ctx context.Context, opts ListOptions) ([]Model, error) {
	if ctx == nil {
		return nil, fmt.Errorf("model catalog list context is required")
	}
	now := defaultNow(opts.Now)
	listOpts := opts
	listOpts.Now = now
	rows, err := s.store.ListRows(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("model catalog: list stored rows: %w", err)
	}

	var refreshStatuses []SourceStatus
	var refreshErr error
	if opts.Refresh || (len(rows) == 0 && len(s.sources) > 0) {
		refreshStatuses, refreshErr = s.Refresh(ctx, RefreshOptions{
			ProviderID: opts.ProviderID,
			SourceID:   opts.SourceID,
			Force:      opts.Refresh,
			Now:        now,
		})
		rows, err = s.store.ListRows(ctx, listOpts)
		if err != nil {
			return nil, fmt.Errorf("model catalog: list stored rows after refresh: %w", err)
		}
	}
	if len(rows) == 0 && refreshErr != nil && !hasStaleFailureStatus(refreshStatuses) {
		return nil, refreshErr
	}
	return MergeRows(rows), nil
}

// Refresh updates registered sources and returns their latest statuses.
func (s *CatalogService) Refresh(ctx context.Context, opts RefreshOptions) ([]SourceStatus, error) {
	if ctx == nil {
		return nil, fmt.Errorf("model catalog refresh context is required")
	}
	now := defaultNow(opts.Now)
	sources, err := s.selectSources(opts.SourceID)
	if err != nil {
		return nil, err
	}
	providerKey := strings.TrimSpace(opts.ProviderID)
	if providerKey == "" {
		return s.refreshAllProviders(ctx, sources, opts, now)
	}
	scopeKey := refreshFlightScopeKey(providerKey, opts)

	return s.withRefreshFlight(ctx, providerKey, scopeKey, func() ([]SourceStatus, error) {
		return s.refreshSources(ctx, sources, opts, now)
	})
}

// ListSourceStatus returns provider-scoped source health rows.
func (s *CatalogService) ListSourceStatus(ctx context.Context, providerID string) ([]SourceStatus, error) {
	if ctx == nil {
		return nil, fmt.Errorf("model catalog status context is required")
	}
	statuses, err := s.store.ListSourceStatus(ctx, strings.TrimSpace(providerID))
	if err != nil {
		return nil, fmt.Errorf("model catalog: list source status: %w", err)
	}
	for index := range statuses {
		statuses[index].LastError = RedactString(statuses[index].LastError)
	}
	return statuses, nil
}

func (s *CatalogService) refreshSources(
	ctx context.Context,
	sources []Source,
	opts RefreshOptions,
	now time.Time,
) ([]SourceStatus, error) {
	statuses := make([]SourceStatus, 0, len(sources))
	var degradedErrs []error
	successes := 0
	failures := 0
	staleFallbacks := 0
	for _, source := range sources {
		sourceStatuses, outcome, err := s.refreshSource(ctx, source, opts, now)
		statuses = append(statuses, sourceStatuses...)
		if err != nil {
			if outcome == refreshOutcomeStale {
				degradedErrs = append(degradedErrs, err)
			} else if !errors.Is(err, ErrSourceDisabled) {
				failures++
			}
		}
		switch outcome {
		case refreshOutcomeSuccess:
			successes++
		case refreshOutcomeStale:
			staleFallbacks++
		}
	}
	if len(degradedErrs) > 0 {
		return statuses, errors.Join(degradedErrs...)
	}
	if successes == 0 && staleFallbacks == 0 && failures > 0 {
		return statuses, fmt.Errorf("%w (%d failed)", ErrAllSourcesFailed, failures)
	}
	return statuses, nil
}

func (s *CatalogService) refreshAllProviders(
	ctx context.Context,
	sources []Source,
	opts RefreshOptions,
	now time.Time,
) ([]SourceStatus, error) {
	statuses := make([]SourceStatus, 0)
	var firstErr error
	var degradedErrs []error
	successes := 0

	for _, source := range sources {
		providers := sourceProviders(source)
		if len(providers) == 0 {
			sourceStatuses, err := s.refreshSources(ctx, []Source{source}, opts, now)
			statuses = append(statuses, sourceStatuses...)
			if err != nil {
				if hasStaleFailureStatus(sourceStatuses) {
					degradedErrs = append(degradedErrs, err)
				} else if firstErr == nil {
					firstErr = err
				}
			} else {
				successes++
			}
			continue
		}

		for _, providerID := range providers {
			providerOpts := opts
			providerOpts.ProviderID = providerID
			providerOpts.SourceID = source.ID()
			scopeKey := refreshFlightScopeKey(providerID, providerOpts)
			sourceStatuses, err := s.withRefreshFlight(ctx, providerID, scopeKey, func() ([]SourceStatus, error) {
				return s.refreshSources(ctx, []Source{source}, providerOpts, now)
			})
			statuses = append(statuses, sourceStatuses...)
			if err != nil {
				if hasStaleFailureStatus(sourceStatuses) {
					degradedErrs = append(degradedErrs, err)
				} else if firstErr == nil {
					firstErr = err
				}
				continue
			}
			successes++
		}
	}

	if len(degradedErrs) > 0 {
		return statuses, errors.Join(degradedErrs...)
	}
	if successes == 0 && firstErr != nil {
		return statuses, firstErr
	}
	return statuses, nil
}

type refreshOutcome int

const (
	refreshOutcomeEmpty refreshOutcome = iota
	refreshOutcomeSuccess
	refreshOutcomeStale
)

func (s *CatalogService) refreshSource(
	ctx context.Context,
	source Source,
	opts RefreshOptions,
	now time.Time,
) ([]SourceStatus, refreshOutcome, error) {
	if !opts.Force &&
		strings.TrimSpace(opts.ProviderID) != "" &&
		sourceHasFreshStatus(ctx, s.store, source, opts.ProviderID, now) {
		statuses, err := s.store.ListSourceStatus(ctx, opts.ProviderID)
		if err != nil {
			return nil, refreshOutcomeEmpty, fmt.Errorf("model catalog: load fresh source status: %w", err)
		}
		return filterStatusesBySource(statuses, source.ID()), refreshOutcomeSuccess, nil
	}

	rows, err := source.ListModels(ctx, ListOptions{
		ProviderID:   opts.ProviderID,
		SourceID:     source.ID(),
		Refresh:      true,
		IncludeAll:   true,
		IncludeStale: true,
		Now:          now,
	})
	if err != nil {
		return s.recordSourceFailure(ctx, source, opts.ProviderID, rows, now, err)
	}
	statuses, err := s.persistSourceRows(ctx, source, opts.ProviderID, rows, now, false, "")
	if err != nil {
		return nil, refreshOutcomeEmpty, err
	}
	if len(rows) > 0 {
		return statuses, refreshOutcomeSuccess, nil
	}
	return statuses, refreshOutcomeEmpty, nil
}

func (s *CatalogService) recordSourceFailure(
	ctx context.Context,
	source Source,
	providerID string,
	rows []ModelRow,
	now time.Time,
	sourceErr error,
) ([]SourceStatus, refreshOutcome, error) {
	if errors.Is(sourceErr, ErrSourceDisabled) {
		statuses, err := s.persistDisabledSource(ctx, source, providerID, now)
		return statuses, refreshOutcomeEmpty, err
	}
	redacted := sourceErrorText(sourceErr)
	if len(rows) > 0 {
		staleRows := markRowsStale(rows, redacted)
		statuses, err := s.persistSourceRows(ctx, source, providerID, staleRows, now, true, redacted)
		if err != nil {
			return nil, refreshOutcomeEmpty, err
		}
		return statuses, refreshOutcomeStale, sourceErr
	}

	providers, err := s.providersForSource(ctx, source, providerID, now)
	if err != nil {
		return nil, refreshOutcomeEmpty, err
	}
	statuses := make([]SourceStatus, 0, len(providers))
	staleCount := 0
	for _, provider := range providers {
		previous, err := s.store.ListRows(ctx, ListOptions{
			ProviderID:   provider,
			SourceID:     source.ID(),
			IncludeAll:   true,
			IncludeStale: true,
			Now:          now,
		})
		if err != nil {
			return nil, refreshOutcomeEmpty, fmt.Errorf("model catalog: load stale rows for %q: %w", source.ID(), err)
		}
		staleRows := markRowsStale(previous, redacted)
		status := sourceStatus(source, provider, now, len(staleRows), true, redacted, RefreshStateFailed)
		s.preserveLastSuccess(ctx, provider, &status)
		if err := s.store.ReplaceSourceRows(ctx, source.ID(), provider, staleRows, status); err != nil {
			return nil, refreshOutcomeEmpty, fmt.Errorf("model catalog: persist failed source status: %w", err)
		}
		if len(staleRows) > 0 {
			staleCount += len(staleRows)
		}
		statuses = append(statuses, status)
	}
	if staleCount > 0 {
		return statuses, refreshOutcomeStale, sourceErr
	}
	return statuses, refreshOutcomeEmpty, sourceErr
}

func (s *CatalogService) persistSourceRows(
	ctx context.Context,
	source Source,
	providerID string,
	rows []ModelRow,
	now time.Time,
	stale bool,
	lastError string,
) ([]SourceStatus, error) {
	grouped := groupRowsByProvider(source, rows)
	providers := providerKeys(grouped)
	if strings.TrimSpace(providerID) != "" && len(providers) == 0 {
		providers = []string{strings.TrimSpace(providerID)}
	}
	if len(providers) == 0 {
		var err error
		providers, err = s.providersForSource(ctx, source, providerID, now)
		if err != nil {
			return nil, err
		}
	}
	statuses := make([]SourceStatus, 0, len(providers))
	state := RefreshStateSucceeded
	if stale {
		state = RefreshStateFailed
	}
	for _, provider := range providers {
		providerRows := grouped[provider]
		for index := range providerRows {
			providerRows[index] = normalizeSourceRow(source, providerRows[index], now, stale, lastError)
		}
		status := sourceStatus(source, provider, now, len(providerRows), stale, lastError, state)
		if stale {
			s.preserveLastSuccess(ctx, provider, &status)
		}
		if err := s.store.ReplaceSourceRows(ctx, source.ID(), provider, providerRows, status); err != nil {
			return nil, fmt.Errorf("model catalog: persist source rows for %q/%q: %w", source.ID(), provider, err)
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (s *CatalogService) persistDisabledSource(
	ctx context.Context,
	source Source,
	providerID string,
	now time.Time,
) ([]SourceStatus, error) {
	providers, err := s.providersForSource(ctx, source, providerID, now)
	if err != nil {
		return nil, err
	}
	statuses := make([]SourceStatus, 0, len(providers))
	for _, provider := range providers {
		status := sourceStatus(source, provider, now, 0, false, "", RefreshStateDisabled)
		if err := s.store.ReplaceSourceRows(ctx, source.ID(), provider, nil, status); err != nil {
			return nil, fmt.Errorf("model catalog: persist disabled source status: %w", err)
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func (s *CatalogService) providersForSource(
	ctx context.Context,
	source Source,
	providerID string,
	now time.Time,
) ([]string, error) {
	if trimmed := strings.TrimSpace(providerID); trimmed != "" {
		return []string{trimmed}, nil
	}
	if lister, ok := source.(sourceProviderLister); ok {
		providers := normalizedProviderIDs(lister.ProviderIDs())
		if len(providers) > 0 {
			return providers, nil
		}
	}
	providers, err := s.storedProvidersForSource(ctx, source, now)
	if err != nil {
		return nil, err
	}
	return providers, nil
}

func (s *CatalogService) storedProvidersForSource(ctx context.Context, source Source, now time.Time) ([]string, error) {
	providerSet := make(map[string]struct{})
	statuses, err := s.store.ListSourceStatus(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("model catalog: list stored source providers for %q: %w", source.ID(), err)
	}
	for _, status := range statuses {
		if status.SourceID != source.ID() {
			continue
		}
		if providerID := strings.TrimSpace(status.ProviderID); providerID != "" {
			providerSet[providerID] = struct{}{}
		}
	}
	rows, err := s.store.ListRows(ctx, ListOptions{
		SourceID:     source.ID(),
		IncludeAll:   true,
		IncludeStale: true,
		Now:          now,
	})
	if err != nil {
		return nil, fmt.Errorf("model catalog: list stored source rows for %q: %w", source.ID(), err)
	}
	for _, row := range rows {
		if providerID := strings.TrimSpace(row.ProviderID); providerID != "" {
			providerSet[providerID] = struct{}{}
		}
	}
	providers := make([]string, 0, len(providerSet))
	for providerID := range providerSet {
		providers = append(providers, providerID)
	}
	sort.Strings(providers)
	return providers, nil
}

func (s *CatalogService) selectSources(sourceID string) ([]Source, error) {
	trimmed := strings.TrimSpace(sourceID)
	if trimmed == "" {
		return s.sources, nil
	}
	source, ok := s.sourceByID[trimmed]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrSourceNotRegistered, trimmed)
	}
	return []Source{source}, nil
}

func (s *CatalogService) withRefreshFlight(
	ctx context.Context,
	providerID string,
	scopeKey string,
	fn func() ([]SourceStatus, error),
) ([]SourceStatus, error) {
	for {
		s.lockMu.Lock()
		flight := s.refreshFlights[providerID]
		if flight == nil {
			flight = &refreshFlight{
				scopeKey: scopeKey,
				done:     make(chan struct{}),
			}
			s.refreshFlights[providerID] = flight
			s.lockMu.Unlock()

			flight.statuses, flight.err = fn()
			s.lockMu.Lock()
			close(flight.done)
			delete(s.refreshFlights, providerID)
			s.lockMu.Unlock()
			return cloneSourceStatuses(flight.statuses), flight.err
		}
		s.lockMu.Unlock()
		if hook := s.onFlightWait; hook != nil {
			hook(providerID)
		}
		select {
		case <-flight.done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if flight.scopeKey == scopeKey {
			return cloneSourceStatuses(flight.statuses), flight.err
		}
	}
}

func refreshFlightScopeKey(providerKey string, opts RefreshOptions) string {
	return fmt.Sprintf("%s\x00%s\x00%t", providerKey, strings.TrimSpace(opts.SourceID), opts.Force)
}

func sourceHasFreshStatus(ctx context.Context, store Store, source Source, providerID string, now time.Time) bool {
	if ttlProvider, ok := source.(sourceTTLProvider); !ok || ttlProvider.TTL() <= 0 {
		return false
	}
	statuses, err := store.ListSourceStatus(ctx, providerID)
	if err != nil {
		return false
	}
	for _, status := range statuses {
		if status.SourceID != source.ID() {
			continue
		}
		return status.RefreshState == RefreshStateSucceeded &&
			!status.NextRefresh.IsZero() &&
			status.NextRefresh.After(now)
	}
	return false
}

func filterStatusesBySource(statuses []SourceStatus, sourceID string) []SourceStatus {
	filtered := make([]SourceStatus, 0, len(statuses))
	for _, status := range statuses {
		if status.SourceID == sourceID {
			filtered = append(filtered, status)
		}
	}
	return filtered
}

func normalizeSourceRow(source Source, row ModelRow, now time.Time, stale bool, lastError string) ModelRow {
	normalized := row
	if strings.TrimSpace(normalized.SourceID) == "" {
		normalized.SourceID = source.ID()
	}
	if normalized.SourceKind == "" {
		normalized.SourceKind = source.Kind()
	}
	if normalized.Priority == 0 {
		normalized.Priority = source.Priority()
	}
	if normalized.RefreshedAt.IsZero() {
		normalized.RefreshedAt = now
	}
	normalized.Stale = stale || normalized.Stale
	if lastError != "" {
		normalized.LastError = RedactString(lastError)
	} else {
		normalized.LastError = RedactString(normalized.LastError)
	}
	return normalized
}

func sourceStatus(
	source Source,
	providerID string,
	now time.Time,
	rowCount int,
	stale bool,
	lastError string,
	state RefreshState,
) SourceStatus {
	status := SourceStatus{
		SourceID:     source.ID(),
		SourceKind:   source.Kind(),
		ProviderID:   providerID,
		Priority:     source.Priority(),
		LastRefresh:  now,
		RefreshState: state,
		RowCount:     rowCount,
		Stale:        stale,
		LastError:    RedactString(lastError),
	}
	if state == RefreshStateSucceeded {
		status.LastSuccess = now
	}
	if ttlProvider, ok := source.(sourceTTLProvider); ok && ttlProvider.TTL() > 0 {
		status.NextRefresh = now.Add(ttlProvider.TTL())
	}
	return status
}

func (s *CatalogService) preserveLastSuccess(ctx context.Context, providerID string, status *SourceStatus) {
	statuses, err := s.store.ListSourceStatus(ctx, providerID)
	if err != nil {
		return
	}
	for _, previous := range statuses {
		if previous.SourceID == status.SourceID {
			status.LastSuccess = previous.LastSuccess
			return
		}
	}
}

func groupRowsByProvider(source Source, rows []ModelRow) map[string][]ModelRow {
	grouped := make(map[string][]ModelRow)
	for _, row := range rows {
		providerID := strings.TrimSpace(row.ProviderID)
		if providerID == "" {
			continue
		}
		normalized := row
		normalized.SourceID = source.ID()
		normalized.SourceKind = source.Kind()
		normalized.Priority = source.Priority()
		grouped[providerID] = append(grouped[providerID], normalized)
	}
	return grouped
}

func providerKeys(grouped map[string][]ModelRow) []string {
	providers := make([]string, 0, len(grouped))
	for providerID := range grouped {
		providers = append(providers, providerID)
	}
	sort.Strings(providers)
	return providers
}

func markRowsStale(rows []ModelRow, lastError string) []ModelRow {
	staleRows := make([]ModelRow, 0, len(rows))
	for _, row := range rows {
		stale := row
		stale.Stale = true
		stale.LastError = RedactString(lastError)
		staleRows = append(staleRows, stale)
	}
	return staleRows
}

func cloneSourceStatuses(statuses []SourceStatus) []SourceStatus {
	return append([]SourceStatus(nil), statuses...)
}

func hasStaleFailureStatus(statuses []SourceStatus) bool {
	for _, status := range statuses {
		if status.Stale && status.RefreshState == RefreshStateFailed && status.RowCount > 0 {
			return true
		}
	}
	return false
}

func sourceProviders(source Source) []string {
	lister, ok := source.(sourceProviderLister)
	if !ok {
		return nil
	}
	return normalizedProviderIDs(lister.ProviderIDs())
}

func normalizedProviderIDs(providerIDs []string) []string {
	providerSet := make(map[string]struct{}, len(providerIDs))
	for _, providerID := range providerIDs {
		trimmed := strings.TrimSpace(providerID)
		if trimmed != "" {
			providerSet[trimmed] = struct{}{}
		}
	}
	providers := make([]string, 0, len(providerSet))
	for providerID := range providerSet {
		providers = append(providers, providerID)
	}
	sort.Strings(providers)
	return providers
}
