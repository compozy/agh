package registry

import (
	"context"
	"errors"
)

// ErrNotSupported reports that a registry source does not implement an
// operation.
var ErrNotSupported = errors.New("registry: operation not supported")

// RegistrySource abstracts one registry backend.
type RegistrySource interface {
	Name() string
	Capabilities() SourceCaps
	Search(ctx context.Context, query string, opts SearchOpts) ([]Listing, error)
	Info(ctx context.Context, slug string) (*Detail, error)
	Download(ctx context.Context, slug string, opts DownloadOpts) (*DownloadResult, error)
	Close() error
}

// Downloader is the minimal download contract required by the installer.
type Downloader interface {
	Download(ctx context.Context, slug string, opts DownloadOpts) (*DownloadResult, error)
}
