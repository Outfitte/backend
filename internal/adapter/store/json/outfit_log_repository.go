package json

import (
	"context"
	"sort"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
)

var _ ports.OutfitLogRepository = (*OutfitLogRepository)(nil)

// OutfitLogRepository is a JSON file-backed implementation of ports.OutfitLogRepository.
type OutfitLogRepository struct {
	provider *Provider[domain.OutfitLog]
}

// NewOutfitLogRepository creates an OutfitLogRepository that stores outfit logs in root/outfit_logs.json.
func NewOutfitLogRepository(root string) *OutfitLogRepository {
	return &OutfitLogRepository{
		provider: NewProvider[domain.OutfitLog](root, "outfit_logs.json"),
	}
}

func (r *OutfitLogRepository) Get(ctx context.Context, id string) (domain.OutfitLog, error) {
	return r.provider.Get(ctx, id)
}

func (r *OutfitLogRepository) Save(ctx context.Context, log domain.OutfitLog) error {
	return r.provider.Save(ctx, log)
}

func (r *OutfitLogRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.provider.Delete(ctx, id)
}

func (r *OutfitLogRepository) ListByOutfit(ctx context.Context, outfitID string) ([]domain.OutfitLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return nil, err
	}
	var result []domain.OutfitLog
	for _, ol := range all {
		if ol.OutfitID == outfitID {
			result = append(result, ol)
		}
	}
	sortOutfitLogsByWornOnDesc(result)
	return result, nil
}

func (r *OutfitLogRepository) ListByOwnerDateRange(ctx context.Context, ownerID string, from, to time.Time) ([]domain.OutfitLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return nil, err
	}
	var result []domain.OutfitLog
	for _, ol := range all {
		if ol.OwnerID == ownerID && !ol.WornOn.Before(from) && !ol.WornOn.After(to) {
			result = append(result, ol)
		}
	}
	sortOutfitLogsByWornOnAsc(result)
	return result, nil
}

func (r *OutfitLogRepository) LinkWearLog(ctx context.Context, outfitLogID, wearLogID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	ol, err := r.provider.Get(ctx, outfitLogID)
	if err != nil {
		return err
	}
	ol.WearLogIDs = append(ol.WearLogIDs, wearLogID)
	return r.provider.Save(ctx, ol)
}

func (r *OutfitLogRepository) LinkedWearLogIDs(ctx context.Context, outfitLogID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ol, err := r.provider.Get(ctx, outfitLogID)
	if err != nil {
		return nil, err
	}
	return ol.WearLogIDs, nil
}

func sortOutfitLogsByWornOnDesc(logs []domain.OutfitLog) {
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].WornOn.After(logs[j].WornOn)
	})
}

func sortOutfitLogsByWornOnAsc(logs []domain.OutfitLog) {
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].WornOn.Before(logs[j].WornOn)
	})
}
