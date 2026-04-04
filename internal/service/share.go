package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/ports"
)

// CreateShareInput holds the fields required to create a new Share.
type CreateShareInput struct {
	RecipientID string
	TargetType  domain.ShareTargetType
	TargetID    string
}

// UserSummary holds the minimal user info used in share views.
type UserSummary struct {
	ID    string
	Email string
}

// ShareView wraps a Share record with the recipient's summary for display.
type ShareView struct {
	Share     domain.Share
	Recipient UserSummary
}

// SharedEntity wraps a domain entity with the owner's summary.
type SharedEntity[T any] struct {
	Entity  T
	SharedBy UserSummary
}

// SharedLocation includes a location and all items assigned to it or its descendants.
type SharedLocation struct {
	Location domain.Location
	Items    []domain.Item
	SharedBy UserSummary
}

// SharedWithMeResult is the hydrated result for the shared-with-me query.
type SharedWithMeResult struct {
	Items     []SharedEntity[domain.Item]
	Outfits   []SharedEntity[domain.Outfit]
	Locations []SharedLocation
}

// ShareService manages share CRUD and the hydrated shared-with-me query.
type ShareService struct {
	shares    ports.ShareRepository
	users     ports.UserRepository
	items     ports.ItemRepository
	outfits   ports.OutfitRepository
	locations ports.LocationRepository
}

// NewShareService constructs a ShareService backed by the given repositories.
func NewShareService(
	shares ports.ShareRepository,
	users ports.UserRepository,
	items ports.ItemRepository,
	outfits ports.OutfitRepository,
	locations ports.LocationRepository,
) *ShareService {
	return &ShareService{
		shares:    shares,
		users:     users,
		items:     items,
		outfits:   outfits,
		locations: locations,
	}
}

// Create validates and persists a new share from callerID to a recipient for a target entity.
func (s *ShareService) Create(ctx context.Context, callerID string, input CreateShareInput) (domain.Share, error) {
	if err := ctx.Err(); err != nil {
		return domain.Share{}, err
	}
	if input.RecipientID == callerID {
		return domain.Share{}, domain.ErrSelfShare
	}
	if _, err := s.users.Get(ctx, input.RecipientID); err != nil {
		return domain.Share{}, err
	}
	if err := s.validateTargetOwnership(ctx, callerID, input.TargetType, input.TargetID); err != nil {
		return domain.Share{}, err
	}
	existing, err := s.shares.FindByTarget(ctx, callerID, input.RecipientID, input.TargetType, input.TargetID)
	if err != nil {
		return domain.Share{}, err
	}
	if existing != nil {
		return domain.Share{}, domain.ErrDuplicateShare
	}
	return s.saveNewShare(ctx, callerID, input)
}

func (s *ShareService) saveNewShare(ctx context.Context, ownerID string, input CreateShareInput) (domain.Share, error) {
	var share domain.Share
	share.ID = uuid.NewString()
	share.OwnerID = ownerID
	share.RecipientID = input.RecipientID
	share.TargetType = input.TargetType
	share.TargetID = input.TargetID
	share.CreatedAt = time.Now().UTC()
	if err := s.shares.Save(ctx, share); err != nil {
		return domain.Share{}, err
	}
	return share, nil
}

func (s *ShareService) validateTargetOwnership(ctx context.Context, callerID string, targetType domain.ShareTargetType, targetID string) error {
	switch targetType {
	case domain.ShareTargetItem:
		item, err := s.items.Get(ctx, targetID)
		if err != nil {
			return err
		}
		if item.OwnerID != callerID {
			return domain.ErrForbidden
		}
	case domain.ShareTargetOutfit:
		outfit, err := s.outfits.Get(ctx, targetID)
		if err != nil {
			return err
		}
		if outfit.OwnerID != callerID {
			return domain.ErrForbidden
		}
	case domain.ShareTargetLocation:
		loc, err := s.locations.Get(ctx, targetID)
		if err != nil {
			return err
		}
		if loc.OwnerID != callerID {
			return domain.ErrForbidden
		}
	default:
		return domain.ErrValidation
	}
	return nil
}

// ListOutgoing returns all shares created by callerID, enriched with recipient summaries.
func (s *ShareService) ListOutgoing(ctx context.Context, callerID string) ([]ShareView, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	shares, err := s.shares.ListByOwner(ctx, callerID)
	if err != nil {
		return nil, err
	}
	views := make([]ShareView, 0, len(shares))
	for _, share := range shares {
		recipient, err := s.users.Get(ctx, share.RecipientID)
		if err != nil {
			return nil, err
		}
		views = append(views, ShareView{
			Share:     share,
			Recipient: UserSummary{ID: recipient.GetID(), Email: recipient.Email},
		})
	}
	return views, nil
}

// ListSharedWithMe returns all entities shared with callerID, grouped and hydrated.
func (s *ShareService) ListSharedWithMe(ctx context.Context, callerID string) (SharedWithMeResult, error) {
	if err := ctx.Err(); err != nil {
		return SharedWithMeResult{}, err
	}
	items, err := s.hydrateSharedItems(ctx, callerID)
	if err != nil {
		return SharedWithMeResult{}, err
	}
	outfits, err := s.hydrateSharedOutfits(ctx, callerID)
	if err != nil {
		return SharedWithMeResult{}, err
	}
	locations, err := s.hydrateSharedLocations(ctx, callerID)
	if err != nil {
		return SharedWithMeResult{}, err
	}
	return SharedWithMeResult{Items: items, Outfits: outfits, Locations: locations}, nil
}

func (s *ShareService) hydrateSharedItems(ctx context.Context, callerID string) ([]SharedEntity[domain.Item], error) {
	itemShares, err := s.shares.ListByRecipientAndType(ctx, callerID, domain.ShareTargetItem)
	if err != nil {
		return nil, err
	}
	result := make([]SharedEntity[domain.Item], 0, len(itemShares))
	for _, share := range itemShares {
		item, err := s.items.Get(ctx, share.TargetID)
		if err != nil {
			return nil, err
		}
		owner, err := s.users.Get(ctx, share.OwnerID)
		if err != nil {
			return nil, err
		}
		result = append(result, SharedEntity[domain.Item]{
			Entity:   item,
			SharedBy: UserSummary{ID: owner.GetID(), Email: owner.Email},
		})
	}
	return result, nil
}

func (s *ShareService) hydrateSharedOutfits(ctx context.Context, callerID string) ([]SharedEntity[domain.Outfit], error) {
	outfitShares, err := s.shares.ListByRecipientAndType(ctx, callerID, domain.ShareTargetOutfit)
	if err != nil {
		return nil, err
	}
	result := make([]SharedEntity[domain.Outfit], 0, len(outfitShares))
	for _, share := range outfitShares {
		outfit, err := s.outfits.Get(ctx, share.TargetID)
		if err != nil {
			return nil, err
		}
		owner, err := s.users.Get(ctx, share.OwnerID)
		if err != nil {
			return nil, err
		}
		result = append(result, SharedEntity[domain.Outfit]{
			Entity:   outfit,
			SharedBy: UserSummary{ID: owner.GetID(), Email: owner.Email},
		})
	}
	return result, nil
}

func (s *ShareService) hydrateSharedLocations(ctx context.Context, callerID string) ([]SharedLocation, error) {
	locationShares, err := s.shares.ListByRecipientAndType(ctx, callerID, domain.ShareTargetLocation)
	if err != nil {
		return nil, err
	}
	result := make([]SharedLocation, 0, len(locationShares))
	for _, share := range locationShares {
		loc, err := s.locations.Get(ctx, share.TargetID)
		if err != nil {
			return nil, err
		}
		owner, err := s.users.Get(ctx, share.OwnerID)
		if err != nil {
			return nil, err
		}
		items, err := s.collectLocationTreeItems(ctx, loc.OwnerID, loc.GetID())
		if err != nil {
			return nil, err
		}
		result = append(result, SharedLocation{
			Location: loc,
			Items:    items,
			SharedBy: UserSummary{ID: owner.GetID(), Email: owner.Email},
		})
	}
	return result, nil
}

// collectLocationTreeItems returns all items assigned to locationID or any of its descendants,
// fetching all locations by owner to walk the subtree.
func (s *ShareService) collectLocationTreeItems(ctx context.Context, ownerID, rootLocationID string) ([]domain.Item, error) {
	allLocs, err := s.locations.ListByOwner(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	locationIDs := collectDescendantIDs(rootLocationID, allLocs)

	allItems, err := s.items.ListByOwner(ctx, ownerID, ports.ItemListFilter{Status: ports.ItemStatusAll})
	if err != nil {
		return nil, err
	}
	var result []domain.Item
	for _, item := range allItems {
		if item.LocationID != nil && locationIDs[*item.LocationID] {
			result = append(result, item)
		}
	}
	return result, nil
}

// collectDescendantIDs returns a set of location IDs including rootID and all its descendants.
func collectDescendantIDs(rootID string, allLocs []domain.Location) map[string]bool {
	childrenOf := make(map[string][]string, len(allLocs))
	for _, loc := range allLocs {
		if loc.ParentID != nil {
			childrenOf[*loc.ParentID] = append(childrenOf[*loc.ParentID], loc.GetID())
		}
	}
	ids := map[string]bool{rootID: true}
	queue := []string{rootID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, child := range childrenOf[cur] {
			ids[child] = true
			queue = append(queue, child)
		}
	}
	return ids
}

// Revoke deletes the share identified by shareID if callerID is the share owner.
func (s *ShareService) Revoke(ctx context.Context, callerID, shareID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	share, err := s.shares.Get(ctx, shareID)
	if err != nil {
		return err
	}
	if share.OwnerID != callerID {
		return domain.ErrForbidden
	}
	return s.shares.Delete(ctx, shareID)
}

// HasReadAccess reports whether callerID has read access to the given target.
// For items: direct share or via a shared location ancestor.
// For outfits: direct share only.
// For locations: direct share or via a shared ancestor location.
func (s *ShareService) HasReadAccess(ctx context.Context, callerID string, targetType domain.ShareTargetType, targetID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	switch targetType {
	case domain.ShareTargetOutfit:
		return s.shares.HasDirectAccess(ctx, callerID, targetType, targetID)
	case domain.ShareTargetItem:
		return s.hasItemReadAccess(ctx, callerID, targetID)
	case domain.ShareTargetLocation:
		return s.hasLocationReadAccess(ctx, callerID, targetID)
	}
	return false, nil
}

func (s *ShareService) hasItemReadAccess(ctx context.Context, callerID, itemID string) (bool, error) {
	ok, err := s.shares.HasDirectAccess(ctx, callerID, domain.ShareTargetItem, itemID)
	if err != nil || ok {
		return ok, err
	}
	item, err := s.items.Get(ctx, itemID)
	if err != nil {
		return false, err
	}
	if item.LocationID == nil {
		return false, nil
	}
	return s.hasLocationReadAccess(ctx, callerID, *item.LocationID)
}

func (s *ShareService) hasLocationReadAccess(ctx context.Context, callerID, locationID string) (bool, error) {
	current := locationID
	visited := make(map[string]bool)
	for current != "" {
		if visited[current] {
			break
		}
		visited[current] = true
		ok, err := s.shares.HasDirectAccess(ctx, callerID, domain.ShareTargetLocation, current)
		if err != nil || ok {
			return ok, err
		}
		loc, err := s.locations.Get(ctx, current)
		if err != nil {
			return false, err
		}
		if loc.ParentID == nil {
			break
		}
		current = *loc.ParentID
	}
	return false, nil
}

