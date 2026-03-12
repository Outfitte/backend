package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/outfitte/outfitte/internal/domain"
)

// Provider is a local filesystem-backed implementation of ports.MediaProvider.
// Media files are stored under root, keyed by their path segment.
type Provider struct {
	root string
}

// NewProvider creates a Provider that stores media files under root.
func NewProvider(root string) *Provider {
	return &Provider{root: root}
}

// Upload stores the content from r under the given key.
// Creates any necessary parent directories. Translates all os errors into domain errors.
func (p *Provider) Upload(ctx context.Context, key string, r io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	dest := filepath.Join(p.root, key)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("%w: %w", domain.ErrIO, err)
	}

	return nil
}

// Delete removes the media file identified by key.
func (p *Provider) Delete(ctx context.Context, key string) error {
	return errors.New("not implemented")
}

// Download returns a reader for the media file identified by key.
func (p *Provider) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

// GetURL returns the URL for the media file identified by key.
// Returns a relative path of the form /media/<key>; no filesystem I/O is performed.
// Note: unlike Download and Delete, this implementation does not check whether the
// file exists — URL construction is purely deterministic. Existence validation is
// left to the caller or to the Download/Delete methods.
func (p *Provider) GetURL(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return "/media/" + key, nil
}
