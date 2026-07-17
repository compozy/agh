package marketplace

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	storepkg "github.com/compozy/agh/internal/store"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

// SQLiteStore maps validated catalog domain values to the global repository.
type SQLiteStore struct {
	repository storepkg.MarketplaceCatalogRepository
}

var _ Store = (*SQLiteStore)(nil)

// NewSQLiteStore binds the projection to the global SQLite repository.
func NewSQLiteStore(repository storepkg.MarketplaceCatalogRepository) (*SQLiteStore, error) {
	if repository == nil {
		return nil, errors.New("marketplace catalog: SQLite repository is required")
	}
	return &SQLiteStore{repository: repository}, nil
}

// ReplaceKind validates and atomically replaces one kind's projection.
func (s *SQLiteStore) ReplaceKind(ctx context.Context, kind Kind, document *Document) error {
	if err := s.checkReady(ctx); err != nil {
		return err
	}
	if err := validateReplacement(kind, document); err != nil {
		return err
	}
	replacement := storepkg.MarketplaceCatalogReplacement{
		Kind:            string(kind),
		ManifestVersion: int64(document.ManifestVersion),
		GeneratedAt:     storepkg.FormatNullableTimestamp(document.GeneratedAt),
		FetchedAt:       storepkg.FormatTimestamp(document.FetchedAt),
		Entries:         make([]storepkg.MarketplaceCatalogEntry, 0, len(document.Entries)),
	}
	for _, entry := range document.Entries {
		replacement.Entries = append(replacement.Entries, marketplaceEntryToRow(entry, document.FetchedAt))
	}
	if err := s.repository.ReplaceMarketplaceCatalog(ctx, replacement); err != nil {
		return fmt.Errorf("marketplace catalog: replace %q projection: %w", kind, err)
	}
	return nil
}

// MarkKindStale records a redacted failure without touching projected entries.
func (s *SQLiteStore) MarkKindStale(
	ctx context.Context,
	kind Kind,
	errorClass string,
	lastError string,
) error {
	if err := s.checkReady(ctx); err != nil {
		return err
	}
	if _, err := kindFilename(kind); err != nil {
		return err
	}
	if err := s.repository.MarkMarketplaceCatalogStale(
		ctx,
		string(kind),
		encodeStoredError(errorClass, lastError),
	); err != nil {
		return fmt.Errorf("marketplace catalog: mark %q stale: %w", kind, err)
	}
	return nil
}

// ListKind returns deterministic projected entries matching name or description.
func (s *SQLiteStore) ListKind(
	ctx context.Context,
	kind Kind,
	query string,
	limit int,
) ([]Entry, error) {
	if err := s.checkReady(ctx); err != nil {
		return nil, err
	}
	if _, err := kindFilename(kind); err != nil {
		return nil, err
	}
	rows, err := s.repository.ListMarketplaceCatalogEntries(
		ctx,
		string(kind),
		strings.ToLower(strings.TrimSpace(query)),
		int64(normalizeListLimit(limit)),
	)
	if err != nil {
		return nil, fmt.Errorf("marketplace catalog: list %q entries: %w", kind, err)
	}
	entries := make([]Entry, 0, len(rows))
	for _, row := range rows {
		entry, mapErr := marketplaceEntryFromRow(row)
		if mapErr != nil {
			return nil, mapErr
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// GetEntry returns one projected entry by immutable feed identity.
func (s *SQLiteStore) GetEntry(ctx context.Context, kind Kind, entryID string) (*Entry, error) {
	if err := s.checkReady(ctx); err != nil {
		return nil, err
	}
	if _, err := kindFilename(kind); err != nil {
		return nil, err
	}
	trimmedID := strings.TrimSpace(entryID)
	if trimmedID == "" {
		return nil, errors.New("marketplace catalog: entry id is required")
	}
	row, err := s.repository.GetMarketplaceCatalogEntry(ctx, string(kind), trimmedID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s/%s", ErrEntryNotFound, kind, trimmedID)
	}
	if err != nil {
		return nil, fmt.Errorf("marketplace catalog: read %q entry %q: %w", kind, trimmedID, err)
	}
	entry, err := marketplaceEntryFromRow(row)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// KindState returns persisted freshness plus the current projected row count.
func (s *SQLiteStore) KindState(ctx context.Context, kind Kind) (*KindState, error) {
	if err := s.checkReady(ctx); err != nil {
		return nil, err
	}
	if _, err := kindFilename(kind); err != nil {
		return nil, err
	}
	row, err := s.repository.GetMarketplaceCatalogState(ctx, string(kind))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", ErrKindStateMissing, kind)
	}
	if err != nil {
		return nil, fmt.Errorf("marketplace catalog: read %q state: %w", kind, err)
	}
	state := KindState{
		Kind:            Kind(row.Kind),
		ManifestVersion: int(row.ManifestVersion),
		Stale:           row.Stale,
		EntryCount:      int(row.EntryCount),
	}
	if strings.TrimSpace(row.GeneratedAt) != "" {
		parsed, err := storepkg.ParseTimestamp(row.GeneratedAt)
		if err != nil {
			return nil, fmt.Errorf("marketplace catalog: parse %q generated_at: %w", kind, err)
		}
		state.GeneratedAt = parsed
	}
	if strings.TrimSpace(row.FetchedAt) != "" {
		parsed, err := storepkg.ParseTimestamp(row.FetchedAt)
		if err != nil {
			return nil, fmt.Errorf("marketplace catalog: parse %q fetched_at: %w", kind, err)
		}
		state.FetchedAt = parsed
	}
	state.ErrorClass, state.LastError = decodeStoredError(row.LastError)
	return &state, nil
}

func (s *SQLiteStore) checkReady(ctx context.Context) error {
	if ctx == nil {
		return errors.New("marketplace catalog: store context is required")
	}
	if s == nil || s.repository == nil {
		return errors.New("marketplace catalog: SQLite store is required")
	}
	return nil
}

func validateReplacement(kind Kind, document *Document) error {
	if _, err := kindFilename(kind); err != nil {
		return err
	}
	if document == nil {
		return fmt.Errorf("marketplace catalog %q document is required", kind)
	}
	if document.ManifestVersion != ManifestVersion {
		return &UnsupportedManifestVersionError{Kind: kind, Version: document.ManifestVersion}
	}
	if document.GeneratedAt.IsZero() || document.FetchedAt.IsZero() {
		return fmt.Errorf("marketplace catalog %q generated_at and fetched_at are required", kind)
	}
	seen := make(map[string]struct{}, len(document.Entries))
	seenInstallSlugs := make(map[string]struct{}, len(document.Entries))
	for index, entry := range document.Entries {
		if entry.Kind != kind {
			return fmt.Errorf("marketplace catalog %q entry %d kind = %q", kind, index, entry.Kind)
		}
		entryID := strings.TrimSpace(entry.EntryID)
		if entryID == "" || strings.TrimSpace(entry.Name) == "" || strings.TrimSpace(entry.Description) == "" {
			return fmt.Errorf("marketplace catalog %q entry %d identity fields are required", kind, index)
		}
		if _, exists := seen[entryID]; exists {
			return fmt.Errorf("marketplace catalog %q entry_id %q is duplicated", kind, entryID)
		}
		seen[entryID] = struct{}{}
		if kind == KindExtension || kind == KindSkill {
			installSlug := strings.TrimSpace(entry.InstallSlug)
			if installSlug == "" {
				return fmt.Errorf("marketplace catalog %q entry %q install_slug is required", kind, entryID)
			}
			if _, exists := seenInstallSlugs[installSlug]; exists {
				return fmt.Errorf("marketplace catalog %q install_slug %q is duplicated", kind, installSlug)
			}
			seenInstallSlugs[installSlug] = struct{}{}
		}
		if len(entry.Payload) == 0 || !jsonValid(entry.Payload) {
			return fmt.Errorf("marketplace catalog %q entry %q payload_json is invalid", kind, entryID)
		}
	}
	return nil
}

func marketplaceEntryToRow(entry Entry, fetchedAt time.Time) storepkg.MarketplaceCatalogEntry {
	return storepkg.MarketplaceCatalogEntry{
		Kind:         string(entry.Kind),
		EntryID:      strings.TrimSpace(entry.EntryID),
		Name:         strings.TrimSpace(entry.Name),
		Description:  strings.TrimSpace(entry.Description),
		Version:      strings.TrimSpace(entry.Version),
		PublishedAt:  formatOptionalTime(entry.PublishedAt),
		UpdatedAt:    formatOptionalTime(entry.UpdatedAt),
		DigestSHA256: strings.TrimSpace(entry.DigestSHA256),
		Tier:         strings.TrimSpace(entry.Tier),
		InstallSlug:  strings.TrimSpace(entry.InstallSlug),
		PayloadJSON:  string(entry.Payload),
		FetchedAt:    storepkg.FormatTimestamp(fetchedAt),
	}
}

func marketplaceEntryFromRow(row storepkg.MarketplaceCatalogEntry) (Entry, error) {
	entry := Entry{
		Kind:         Kind(row.Kind),
		EntryID:      row.EntryID,
		Name:         row.Name,
		Description:  row.Description,
		Version:      row.Version,
		DigestSHA256: row.DigestSHA256,
		Tier:         row.Tier,
		InstallSlug:  row.InstallSlug,
		Payload:      []byte(row.PayloadJSON),
	}
	if strings.TrimSpace(row.PublishedAt) != "" {
		parsed, err := storepkg.ParseTimestamp(row.PublishedAt)
		if err != nil {
			return Entry{}, fmt.Errorf("marketplace catalog: parse entry published_at: %w", err)
		}
		entry.PublishedAt = &parsed
	}
	if strings.TrimSpace(row.UpdatedAt) != "" {
		parsed, err := storepkg.ParseTimestamp(row.UpdatedAt)
		if err != nil {
			return Entry{}, fmt.Errorf("marketplace catalog: parse entry updated_at: %w", err)
		}
		entry.UpdatedAt = &parsed
	}
	parsed, err := storepkg.ParseTimestamp(row.FetchedAt)
	if err != nil {
		return Entry{}, fmt.Errorf("marketplace catalog: parse entry fetched_at: %w", err)
	}
	entry.FetchedAt = parsed
	return entry, nil
}

func formatOptionalTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return storepkg.FormatTimestamp(*value)
}

func normalizeListLimit(limit int) int {
	if limit <= 0 {
		return defaultListLimit
	}
	if limit > maxListLimit {
		return maxListLimit
	}
	return limit
}

func encodeStoredError(errorClass string, lastError string) string {
	trimmedClass := strings.TrimSpace(errorClass)
	trimmedError := strings.TrimSpace(lastError)
	if trimmedClass == "" {
		return trimmedError
	}
	return "[" + trimmedClass + "] " + trimmedError
}

func decodeStoredError(stored string) (string, string) {
	trimmed := strings.TrimSpace(stored)
	if !strings.HasPrefix(trimmed, "[") {
		return "", trimmed
	}
	end := strings.IndexByte(trimmed, ']')
	if end <= 1 {
		return "", trimmed
	}
	return strings.TrimSpace(trimmed[1:end]), strings.TrimSpace(trimmed[end+1:])
}
