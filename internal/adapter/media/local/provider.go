package local

import (
	"context"
	"errors"
	"io"

	"github.com/outfitte/outfitte/internal/ports"
)

var _ ports.MediaProvider = (*Provider)(nil)

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
func (p *Provider) Upload(ctx context.Context, key string, r io.Reader) error {
	return errors.New("not implemented")
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
func (p *Provider) GetURL(ctx context.Context, key string) (string, error) {
	return "", errors.New("not implemented")
}
