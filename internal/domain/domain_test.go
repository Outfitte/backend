package domain_test

import (
	"testing"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/stretchr/testify/assert"
)

func TestEntitiesImplementPortsEntity(t *testing.T) {
	iface := (*ports.Entity)(nil)
	assert.Implements(t, iface, domain.User{})
	assert.Implements(t, iface, domain.Item{})
	assert.Implements(t, iface, domain.Location{})
	assert.Implements(t, iface, domain.Category{})
	assert.Implements(t, iface, domain.Session{})
}
