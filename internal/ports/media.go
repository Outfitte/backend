package ports

import (
	"context"
	"io"
)

// MediaProvider is the interface for storing and retrieving media files.
// Implementations must translate all infrastructure errors into domain errors
// before returning them.
type MediaProvider interface {
	// Upload stores the content from r under the given key.
	// Infrastructure errors are translated into domain errors before being returned.
	Upload(ctx context.Context, key string, r io.Reader) error

	// Delete removes the media file identified by key.
	// Returns a domain not-found error if no file with that key exists.
	Delete(ctx context.Context, key string) error

	// Download returns a reader for the media file identified by key.
	// The caller is responsible for closing the returned ReadCloser.
	// Returns a domain not-found error if no file with that key exists.
	Download(ctx context.Context, key string) (io.ReadCloser, error)

	// GetURL returns the URL for the media file identified by key.
	// Returns a domain not-found error if no file with that key exists.
	GetURL(ctx context.Context, key string) (string, error)
}
