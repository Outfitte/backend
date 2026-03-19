package json

import (
	"context"
	"sort"

	"github.com/outfitte/outfitte/internal/domain"
)

// WearLogRepository is a JSON file-backed implementation of ports.WearLogRepository.
type WearLogRepository struct {
	provider *Provider[domain.WearLog]
}

// NewWearLogRepository creates a WearLogRepository that stores wear logs in root/wear_logs.json.
func NewWearLogRepository(root string) *WearLogRepository {
	return &WearLogRepository{
		provider: NewProvider[domain.WearLog](root, "wear_logs.json"),
	}
}

func (r *WearLogRepository) Get(ctx context.Context, id string) (domain.WearLog, error) {
	return r.provider.Get(ctx, id)
}

func (r *WearLogRepository) Save(ctx context.Context, log domain.WearLog) error {
	return r.provider.Save(ctx, log)
}

func (r *WearLogRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.provider.Delete(ctx, id)
}

func (r *WearLogRepository) ListByItem(ctx context.Context, itemID string) ([]domain.WearLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return nil, err
	}
	result := filterWearLogsByItem(all, itemID)
	sortWearLogsByWornOnDesc(result)
	return result, nil
}

func (r *WearLogRepository) LatestByItem(ctx context.Context, itemID string) (*domain.WearLog, error) {
	logs, err := r.ListByItem(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if len(logs) == 0 {
		return nil, nil
	}
	return &logs[0], nil
}

func (r *WearLogRepository) CountByItem(ctx context.Context, itemID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	all, err := r.provider.List(ctx)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, wl := range all {
		if wl.ItemID == itemID {
			count++
		}
	}
	return count, nil
}

func filterWearLogsByItem(logs []domain.WearLog, itemID string) []domain.WearLog {
	var result []domain.WearLog
	for _, wl := range logs {
		if wl.ItemID == itemID {
			result = append(result, wl)
		}
	}
	return result
}

func sortWearLogsByWornOnDesc(logs []domain.WearLog) {
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].WornOn.After(logs[j].WornOn)
	})
}
