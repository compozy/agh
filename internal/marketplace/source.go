package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultMaxResponseBytes int64 = 2 << 20

type documentEnvelope struct {
	ManifestVersion *int               `json:"manifest_version"`
	GeneratedAt     string             `json:"generated_at"`
	Entries         *[]json.RawMessage `json:"entries"`
}

// HTTPSource fetches one per-kind document from the configured base URL.
type HTTPSource struct {
	kind             Kind
	endpoint         string
	client           *http.Client
	maxResponseBytes int64
}

var _ Source = (*HTTPSource)(nil)

// HTTPSourceOption customizes bounded feed fetching.
type HTTPSourceOption func(*HTTPSource)

// WithMaxResponseBytes overrides the response cap, primarily for constrained deployments and tests.
func WithMaxResponseBytes(limit int64) HTTPSourceOption {
	return func(source *HTTPSource) {
		if source != nil && limit > 0 {
			source.maxResponseBytes = limit
		}
	}
}

// NewHTTPSource creates one explicit-timeout feed source.
func NewHTTPSource(kind Kind, baseURL string, client *http.Client, options ...HTTPSourceOption) (*HTTPSource, error) {
	filename, err := kindFilename(kind)
	if err != nil {
		return nil, err
	}
	if client == nil || client.Timeout <= 0 {
		return nil, errors.New("marketplace catalog: HTTP client timeout must be positive")
	}
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != protocolHTTP && parsed.Scheme != protocolHTTPS) {
		return nil, errors.New("marketplace catalog: base URL must be an absolute HTTP(S) URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + filename
	parsed.RawQuery = ""
	parsed.Fragment = ""
	source := &HTTPSource{
		kind:             kind,
		endpoint:         parsed.String(),
		client:           client,
		maxResponseBytes: defaultMaxResponseBytes,
	}
	for _, option := range options {
		if option != nil {
			option(source)
		}
	}
	return source, nil
}

func (s *HTTPSource) Kind() Kind {
	if s == nil {
		return ""
	}
	return s.kind
}

// Fetch downloads and validates the source document without mutating projection state.
func (s *HTTPSource) Fetch(ctx context.Context) (document *Document, err error) {
	if ctx == nil {
		return nil, errors.New("marketplace catalog: fetch context is required")
	}
	if s == nil || s.client == nil || s.maxResponseBytes <= 0 {
		return nil, errors.New("marketplace catalog: HTTP source is required")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("marketplace catalog: create %q request: %w", s.kind, err)
	}
	request.Header.Set("Accept", "application/json")
	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("marketplace catalog: fetch %q feed: %w", s.kind, err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("marketplace catalog: close %q response: %w", s.kind, closeErr))
		}
	}()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, &httpStatusError{status: response.StatusCode}
	}
	if response.ContentLength > s.maxResponseBytes {
		return nil, ErrResponseTooLarge
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, s.maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("marketplace catalog: read %q response: %w", s.kind, err)
	}
	if int64(len(body)) > s.maxResponseBytes {
		return nil, ErrResponseTooLarge
	}
	document, err = DecodeDocument(s.kind, body)
	if err != nil {
		return nil, fmt.Errorf("marketplace catalog: validate %q feed: %w", s.kind, err)
	}
	return document, nil
}

// DecodeDocument strictly validates one complete v1 document.
func DecodeDocument(kind Kind, raw []byte) (*Document, error) {
	if _, err := kindFilename(kind); err != nil {
		return nil, err
	}
	var envelope documentEnvelope
	if err := decodeStrict(raw, &envelope); err != nil {
		return nil, fmt.Errorf("marketplace catalog %q document: %w", kind, err)
	}
	if envelope.ManifestVersion == nil || *envelope.ManifestVersion == 0 {
		return nil, fmt.Errorf("marketplace catalog %q manifest_version is required", kind)
	}
	if *envelope.ManifestVersion != ManifestVersion {
		return nil, &UnsupportedManifestVersionError{Kind: kind, Version: *envelope.ManifestVersion}
	}
	if envelope.Entries == nil {
		return nil, fmt.Errorf("marketplace catalog %q entries is required", kind)
	}
	generatedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(envelope.GeneratedAt))
	if err != nil {
		return nil, fmt.Errorf("marketplace catalog %q generated_at must be RFC3339: %w", kind, err)
	}
	entries := make([]Entry, 0, len(*envelope.Entries))
	seen := make(map[string]struct{}, len(*envelope.Entries))
	for index, entryRaw := range *envelope.Entries {
		entry, err := decodeKindEntry(kind, entryRaw)
		if err != nil {
			return nil, fmt.Errorf("marketplace catalog %q entry %d: %w", kind, index, err)
		}
		if _, exists := seen[entry.EntryID]; exists {
			return nil, fmt.Errorf("marketplace catalog %q entry_id %q is duplicated", kind, entry.EntryID)
		}
		seen[entry.EntryID] = struct{}{}
		entries = append(entries, entry)
	}
	return &Document{
		ManifestVersion: *envelope.ManifestVersion,
		GeneratedAt:     generatedAt.UTC(),
		Entries:         entries,
	}, nil
}

func decodeKindEntry(kind Kind, raw []byte) (Entry, error) {
	switch kind {
	case KindMCP:
		return decodeMCPEntry(raw)
	case KindExtension:
		return decodeExtensionEntry(raw)
	case KindSkill:
		return decodeSkillEntry(raw)
	default:
		return Entry{}, fmt.Errorf("marketplace catalog: unsupported kind %q", kind)
	}
}

func decodeStrict(raw []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode JSON: multiple values are not allowed")
		}
		return fmt.Errorf("decode JSON trailing data: %w", err)
	}
	return nil
}

func kindFilename(kind Kind) (string, error) {
	switch kind {
	case KindMCP:
		return "mcp.json", nil
	case KindExtension:
		return "extensions.json", nil
	case KindSkill:
		return "skills.json", nil
	default:
		return "", fmt.Errorf("marketplace catalog: unsupported kind %q", kind)
	}
}
