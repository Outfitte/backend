package json

import (
	"context"

	"github.com/outfitte/backend/internal/domain"
)

// SessionRepository is a JSON file-backed implementation of ports.SessionRepository.
type SessionRepository struct {
	provider *Provider[domain.Session]
}

// NewSessionRepository creates a SessionRepository that stores sessions in root/sessions.json.
func NewSessionRepository(root string) *SessionRepository {
	return &SessionRepository{
		provider: NewProvider[domain.Session](root, "sessions.json"),
	}
}

func (r *SessionRepository) Get(ctx context.Context, id string) (domain.Session, error) {
	return r.provider.Get(ctx, id)
}

func (r *SessionRepository) Save(ctx context.Context, session domain.Session) error {
	return r.provider.Save(ctx, session)
}

func (r *SessionRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.provider.Delete(ctx, id)
}

func (r *SessionRepository) FindByTokenHash(ctx context.Context, hash string) (domain.Session, error) {
	if err := ctx.Err(); err != nil {
		return domain.Session{}, err
	}
	sessions, err := r.provider.List(ctx)
	if err != nil {
		return domain.Session{}, err
	}
	for _, s := range sessions {
		if s.TokenHash == hash {
			return s, nil
		}
	}
	return domain.Session{}, domain.ErrNotFound
}

func (r *SessionRepository) CountByUser(ctx context.Context, userID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	sessions, err := r.provider.List(ctx)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, s := range sessions {
		if s.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (r *SessionRepository) DeleteOldestByUser(ctx context.Context, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	sessions, err := r.provider.List(ctx)
	if err != nil {
		return err
	}
	userSessions := filterSessionsByUser(sessions, userID)
	if len(userSessions) == 0 {
		return domain.ErrNotFound
	}
	return r.provider.Delete(ctx, findOldestSession(userSessions).GetID())
}

func filterSessionsByUser(sessions []domain.Session, userID string) []domain.Session {
	var result []domain.Session
	for _, s := range sessions {
		if s.UserID == userID {
			result = append(result, s)
		}
	}
	return result
}

func findOldestSession(sessions []domain.Session) domain.Session {
	oldest := sessions[0]
	for _, s := range sessions[1:] {
		if s.CreatedAt.Before(oldest.CreatedAt) {
			oldest = s
		}
	}
	return oldest
}
