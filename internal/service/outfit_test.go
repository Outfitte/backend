package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/outfitte/outfitte/internal/domain"
	"github.com/stretchr/testify/require"
)

// mockOutfitRepo is an in-memory ports.OutfitRepository for tests.
type mockOutfitRepo struct {
	outfits        []domain.Outfit
	getErr         error
	saveErr        error
	deleteErr      error
	listByOwnerErr error

	saveItemErr   error
	deleteItemErr error
	listItemIDErr error
	itemIDs       []string

	savePhotoErr   error
	deletePhotoErr error

	// tracking
	savedItemOutfitID   string
	savedItemItemID     string
	savedItemPosition   int
	deletedItemOutfitID string
	deletedItemItemID   string

	savedPhotoOutfitID   string
	savedPhotoPhotoID    string
	savedPhotoKey        string
	savedPhotoPosition   int
	deletedPhotoOutfitID string
	deletedPhotoKey      string
}

func (m *mockOutfitRepo) Get(_ context.Context, id string) (domain.Outfit, error) {
	if m.getErr != nil {
		return domain.Outfit{}, m.getErr
	}
	for _, o := range m.outfits {
		if o.GetID() == id {
			return o, nil
		}
	}
	return domain.Outfit{}, domain.ErrNotFound
}

func (m *mockOutfitRepo) Save(_ context.Context, outfit domain.Outfit) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	for i, existing := range m.outfits {
		if existing.GetID() == outfit.GetID() {
			m.outfits[i] = outfit
			return nil
		}
	}
	m.outfits = append(m.outfits, outfit)
	return nil
}

func (m *mockOutfitRepo) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, o := range m.outfits {
		if o.GetID() == id {
			m.outfits = append(m.outfits[:i], m.outfits[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (m *mockOutfitRepo) ListByOwner(_ context.Context, ownerID string) ([]domain.Outfit, error) {
	if m.listByOwnerErr != nil {
		return nil, m.listByOwnerErr
	}
	var result []domain.Outfit
	for _, o := range m.outfits {
		if o.OwnerID == ownerID {
			result = append(result, o)
		}
	}
	return result, nil
}

func (m *mockOutfitRepo) SaveItem(_ context.Context, outfitID, itemID string, position int) error {
	if m.saveItemErr != nil {
		return m.saveItemErr
	}
	m.savedItemOutfitID = outfitID
	m.savedItemItemID = itemID
	m.savedItemPosition = position
	return nil
}

func (m *mockOutfitRepo) DeleteItem(_ context.Context, outfitID, itemID string) error {
	if m.deleteItemErr != nil {
		return m.deleteItemErr
	}
	m.deletedItemOutfitID = outfitID
	m.deletedItemItemID = itemID
	return nil
}

func (m *mockOutfitRepo) ListItemIDs(_ context.Context, outfitID string) ([]string, error) {
	if m.listItemIDErr != nil {
		return nil, m.listItemIDErr
	}
	_ = outfitID
	return m.itemIDs, nil
}

func (m *mockOutfitRepo) SavePhoto(_ context.Context, outfitID, photoID, mediaKey string, position int) error {
	if m.savePhotoErr != nil {
		return m.savePhotoErr
	}
	m.savedPhotoOutfitID = outfitID
	m.savedPhotoPhotoID = photoID
	m.savedPhotoKey = mediaKey
	m.savedPhotoPosition = position
	return nil
}

func (m *mockOutfitRepo) DeletePhoto(_ context.Context, outfitID, mediaKey string) error {
	if m.deletePhotoErr != nil {
		return m.deletePhotoErr
	}
	m.deletedPhotoOutfitID = outfitID
	m.deletedPhotoKey = mediaKey
	return nil
}

// helpers

func newOutfitSvc(outfits *mockOutfitRepo, items *mockItemRepo, media *mockMediaProvider) *OutfitService {
	return NewOutfitService(outfits, items, media, &mockOutfitLogRepo{})
}

func outfitWithOwner(id, ownerID string) domain.Outfit {
	var o domain.Outfit
	o.ID = id
	o.OwnerID = ownerID
	o.CreatedAt = time.Now().UTC()
	return o
}

func itemWithOwner(id, ownerID string) domain.Item {
	var it domain.Item
	it.ID = id
	it.OwnerID = ownerID
	it.Name = "shirt"
	return it
}

// ---- Create ----

func TestCreateShouldReturnContextErrorWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	_, err := svc.Create(ctx, "user1", CreateOutfitInput{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestCreateShouldReturnRepoErrorWhenSaveFails(t *testing.T) {
	repo := &mockOutfitRepo{saveErr: domain.ErrIO}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	_, err := svc.Create(t.Context(), "user1", CreateOutfitInput{})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestCreateShouldReturnOutfitWhenSuccessful(t *testing.T) {
	repo := &mockOutfitRepo{}
	name := "Summer Look"
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	got, err := svc.Create(t.Context(), "user1", CreateOutfitInput{Name: &name})
	require.NoError(t, err)
	require.NotEmpty(t, got.GetID())
	require.Equal(t, "user1", got.OwnerID)
	require.Equal(t, &name, got.Name)
	require.False(t, got.CreatedAt.IsZero())
}

// ---- GetByID ----

func TestGetByIDShouldReturnContextErrorWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	_, err := svc.GetByID(ctx, "user1", "o1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestGetByIDShouldReturnNotFoundWhenOutfitMissing(t *testing.T) {
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	_, err := svc.GetByID(t.Context(), "user1", "o1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetByIDShouldReturnForbiddenWhenCallerIsNotOwner(t *testing.T) {
	outfit := outfitWithOwner("o1", "owner")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	_, err := svc.GetByID(t.Context(), "other", "o1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestGetByIDShouldReturnOutfitWhenCallerIsOwner(t *testing.T) {
	outfit := outfitWithOwner("o1", "user1")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	got, err := svc.GetByID(t.Context(), "user1", "o1")
	require.NoError(t, err)
	require.Equal(t, "o1", got.GetID())
}

// ---- ListByOwner ----

func TestListByOwnerShouldReturnContextErrorWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	_, err := svc.ListByOwner(ctx, "user1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestListByOwnerShouldReturnRepoErrorWhenListFails(t *testing.T) {
	repo := &mockOutfitRepo{listByOwnerErr: domain.ErrIO}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	_, err := svc.ListByOwner(t.Context(), "user1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListByOwnerShouldReturnOutfitsWhenSuccessful(t *testing.T) {
	outfits := []domain.Outfit{outfitWithOwner("o1", "user1"), outfitWithOwner("o2", "user1")}
	repo := &mockOutfitRepo{outfits: outfits}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	got, err := svc.ListByOwner(t.Context(), "user1")
	require.NoError(t, err)
	require.Len(t, got, 2)
}

// ---- Update ----

func TestUpdateShouldReturnContextErrorWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	_, err := svc.Update(ctx, "user1", "o1", UpdateOutfitInput{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestUpdateShouldReturnNotFoundWhenOutfitMissing(t *testing.T) {
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	_, err := svc.Update(t.Context(), "user1", "o1", UpdateOutfitInput{})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUpdateShouldReturnForbiddenWhenCallerIsNotOwner(t *testing.T) {
	outfit := outfitWithOwner("o1", "owner")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	_, err := svc.Update(t.Context(), "other", "o1", UpdateOutfitInput{})
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestUpdateShouldReturnRepoErrorWhenSaveFails(t *testing.T) {
	outfit := outfitWithOwner("o1", "user1")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}, saveErr: domain.ErrIO}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	_, err := svc.Update(t.Context(), "user1", "o1", UpdateOutfitInput{})
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestUpdateShouldUpdateFieldsWhenSuccessful(t *testing.T) {
	outfit := outfitWithOwner("o1", "user1")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	name := "Evening Attire"
	notes := "for special occasions"
	got, err := svc.Update(t.Context(), "user1", "o1", UpdateOutfitInput{Name: &name, Notes: &notes})
	require.NoError(t, err)
	require.Equal(t, &name, got.Name)
	require.Equal(t, &notes, got.Notes)
}

// ---- Delete ----

func TestDeleteShouldReturnContextErrorWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.Delete(ctx, "user1", "o1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeleteShouldReturnNotFoundWhenOutfitMissing(t *testing.T) {
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.Delete(t.Context(), "user1", "o1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteShouldReturnForbiddenWhenCallerIsNotOwner(t *testing.T) {
	outfit := outfitWithOwner("o1", "owner")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.Delete(t.Context(), "other", "o1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestDeleteShouldReturnMediaErrorWhenDeletePhotoFails(t *testing.T) {
	photo := domain.OutfitPhoto{ID: "ph1", MediaKey: "outfits/o1/uuid/img.jpg"}
	outfit := outfitWithOwner("o1", "user1")
	outfit.Photos = []domain.OutfitPhoto{photo}
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	media := &mockMediaProvider{deleteErr: domain.ErrIO}
	svc := newOutfitSvc(repo, &mockItemRepo{}, media)
	err := svc.Delete(t.Context(), "user1", "o1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestDeleteShouldReturnRepoErrorWhenDeleteFails(t *testing.T) {
	outfit := outfitWithOwner("o1", "user1")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}, deleteErr: domain.ErrIO}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.Delete(t.Context(), "user1", "o1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestDeleteShouldDeleteMediaAndOutfitWhenSuccessful(t *testing.T) {
	photo := domain.OutfitPhoto{ID: "ph1", MediaKey: "outfits/o1/uuid/img.jpg"}
	outfit := outfitWithOwner("o1", "user1")
	outfit.Photos = []domain.OutfitPhoto{photo}
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	media := &mockMediaProvider{}
	svc := newOutfitSvc(repo, &mockItemRepo{}, media)
	err := svc.Delete(t.Context(), "user1", "o1")
	require.NoError(t, err)
	require.Equal(t, "outfits/o1/uuid/img.jpg", media.deletedKeys[0])
	require.Empty(t, repo.outfits)
}

// ---- AddItem ----

func TestAddItemShouldReturnContextErrorWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.AddItem(ctx, "user1", "o1", "i1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestAddItemShouldReturnNotFoundWhenOutfitMissing(t *testing.T) {
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.AddItem(t.Context(), "user1", "o1", "i1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAddItemShouldReturnForbiddenWhenCallerIsNotOutfitOwner(t *testing.T) {
	outfit := outfitWithOwner("o1", "owner")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.AddItem(t.Context(), "other", "o1", "i1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestAddItemShouldReturnNotFoundWhenItemMissing(t *testing.T) {
	outfit := outfitWithOwner("o1", "user1")
	outfitRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	itemRepo := &mockItemRepo{} // no items
	svc := newOutfitSvc(outfitRepo, itemRepo, &mockMediaProvider{})
	err := svc.AddItem(t.Context(), "user1", "o1", "i1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAddItemShouldReturnForbiddenWhenItemBelongsToDifferentUser(t *testing.T) {
	outfit := outfitWithOwner("o1", "user1")
	outfitRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	item := itemWithOwner("i1", "other")
	itemRepo := &mockItemRepo{items: []domain.Item{item}}
	svc := newOutfitSvc(outfitRepo, itemRepo, &mockMediaProvider{})
	err := svc.AddItem(t.Context(), "user1", "o1", "i1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestAddItemShouldReturnRepoErrorWhenListItemIDsFails(t *testing.T) {
	outfit := outfitWithOwner("o1", "user1")
	outfitRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}, listItemIDErr: domain.ErrIO}
	item := itemWithOwner("i1", "user1")
	itemRepo := &mockItemRepo{items: []domain.Item{item}}
	svc := newOutfitSvc(outfitRepo, itemRepo, &mockMediaProvider{})
	err := svc.AddItem(t.Context(), "user1", "o1", "i1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestAddItemShouldReturnRepoErrorWhenSaveItemFails(t *testing.T) {
	outfit := outfitWithOwner("o1", "user1")
	outfitRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}, saveItemErr: domain.ErrIO}
	item := itemWithOwner("i1", "user1")
	itemRepo := &mockItemRepo{items: []domain.Item{item}}
	svc := newOutfitSvc(outfitRepo, itemRepo, &mockMediaProvider{})
	err := svc.AddItem(t.Context(), "user1", "o1", "i1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestAddItemShouldAssignPositionBasedOnCurrentItemCountWhenSuccessful(t *testing.T) {
	outfit := outfitWithOwner("o1", "user1")
	outfitRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}, itemIDs: []string{"existing1", "existing2"}}
	item := itemWithOwner("i1", "user1")
	itemRepo := &mockItemRepo{items: []domain.Item{item}}
	svc := newOutfitSvc(outfitRepo, itemRepo, &mockMediaProvider{})
	err := svc.AddItem(t.Context(), "user1", "o1", "i1")
	require.NoError(t, err)
	require.Equal(t, "o1", outfitRepo.savedItemOutfitID)
	require.Equal(t, "i1", outfitRepo.savedItemItemID)
	require.Equal(t, 2, outfitRepo.savedItemPosition)
}

// ---- RemoveItem ----

func TestRemoveItemShouldReturnContextErrorWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.RemoveItem(ctx, "user1", "o1", "i1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestRemoveItemShouldReturnNotFoundWhenOutfitMissing(t *testing.T) {
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.RemoveItem(t.Context(), "user1", "o1", "i1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestRemoveItemShouldReturnForbiddenWhenCallerIsNotOwner(t *testing.T) {
	outfit := outfitWithOwner("o1", "owner")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.RemoveItem(t.Context(), "other", "o1", "i1")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestRemoveItemShouldReturnRepoErrorWhenDeleteItemFails(t *testing.T) {
	outfit := outfitWithOwner("o1", "user1")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}, deleteItemErr: domain.ErrIO}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.RemoveItem(t.Context(), "user1", "o1", "i1")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestRemoveItemShouldSucceedWhenCallerIsOwner(t *testing.T) {
	outfit := outfitWithOwner("o1", "user1")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.RemoveItem(t.Context(), "user1", "o1", "i1")
	require.NoError(t, err)
	require.Equal(t, "o1", repo.deletedItemOutfitID)
	require.Equal(t, "i1", repo.deletedItemItemID)
}

// ---- UploadPhoto ----

func TestUploadPhotoShouldReturnContextErrorWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.UploadPhoto(ctx, "user1", "o1", strings.NewReader(""), "img.jpg")
	require.ErrorIs(t, err, context.Canceled)
}

func TestUploadPhotoShouldReturnNotFoundWhenOutfitMissing(t *testing.T) {
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.UploadPhoto(t.Context(), "user1", "o1", strings.NewReader(""), "img.jpg")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestUploadPhotoShouldReturnForbiddenWhenCallerIsNotOwner(t *testing.T) {
	outfit := outfitWithOwner("o1", "owner")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.UploadPhoto(t.Context(), "other", "o1", strings.NewReader(""), "img.jpg")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestUploadPhotoShouldReturnMediaErrorWhenUploadFails(t *testing.T) {
	outfit := outfitWithOwner("o1", "user1")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	media := &mockMediaProvider{uploadErr: domain.ErrIO}
	svc := newOutfitSvc(repo, &mockItemRepo{}, media)
	err := svc.UploadPhoto(t.Context(), "user1", "o1", strings.NewReader(""), "img.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestUploadPhotoShouldReturnRepoErrorWhenSavePhotoFails(t *testing.T) {
	outfit := outfitWithOwner("o1", "user1")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}, savePhotoErr: domain.ErrIO}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.UploadPhoto(t.Context(), "user1", "o1", strings.NewReader(""), "img.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestUploadPhotoShouldStorePhotoWithCorrectKeyWhenSuccessful(t *testing.T) {
	photo := domain.OutfitPhoto{ID: "ph0", MediaKey: "outfits/o1/uuid0/a.jpg"}
	outfit := outfitWithOwner("o1", "user1")
	outfit.Photos = []domain.OutfitPhoto{photo}
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.UploadPhoto(t.Context(), "user1", "o1", strings.NewReader("data"), "img.jpg")
	require.NoError(t, err)
	require.Equal(t, "o1", repo.savedPhotoOutfitID)
	require.NotEmpty(t, repo.savedPhotoPhotoID)
	require.Contains(t, repo.savedPhotoKey, "outfits/o1/")
	require.Contains(t, repo.savedPhotoKey, "/img.jpg")
	require.Equal(t, 1, repo.savedPhotoPosition)
}

// ---- DeletePhoto ----

func TestDeletePhotoShouldReturnContextErrorWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.DeletePhoto(ctx, "user1", "o1", "key")
	require.ErrorIs(t, err, context.Canceled)
}

func TestDeletePhotoShouldReturnNotFoundWhenOutfitMissing(t *testing.T) {
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.DeletePhoto(t.Context(), "user1", "o1", "key")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeletePhotoShouldReturnForbiddenWhenCallerIsNotOwner(t *testing.T) {
	outfit := outfitWithOwner("o1", "owner")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.DeletePhoto(t.Context(), "other", "o1", "key")
	require.ErrorIs(t, err, domain.ErrForbidden)
}

func TestDeletePhotoShouldReturnNotFoundWhenPhotoKeyNotOnOutfit(t *testing.T) {
	outfit := outfitWithOwner("o1", "user1")
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.DeletePhoto(t.Context(), "user1", "o1", "nonexistent-key")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeletePhotoShouldReturnMediaErrorWhenDeleteFails(t *testing.T) {
	photo := domain.OutfitPhoto{ID: "ph1", MediaKey: "outfits/o1/uuid/img.jpg"}
	outfit := outfitWithOwner("o1", "user1")
	outfit.Photos = []domain.OutfitPhoto{photo}
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	media := &mockMediaProvider{deleteErr: domain.ErrIO}
	svc := newOutfitSvc(repo, &mockItemRepo{}, media)
	err := svc.DeletePhoto(t.Context(), "user1", "o1", "outfits/o1/uuid/img.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestDeletePhotoShouldReturnRepoErrorWhenDeletePhotoFails(t *testing.T) {
	photo := domain.OutfitPhoto{ID: "ph1", MediaKey: "outfits/o1/uuid/img.jpg"}
	outfit := outfitWithOwner("o1", "user1")
	outfit.Photos = []domain.OutfitPhoto{photo}
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}, deletePhotoErr: domain.ErrIO}
	svc := newOutfitSvc(repo, &mockItemRepo{}, &mockMediaProvider{})
	err := svc.DeletePhoto(t.Context(), "user1", "o1", "outfits/o1/uuid/img.jpg")
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestDeletePhotoShouldDeleteMediaAndRepoEntryWhenSuccessful(t *testing.T) {
	photo := domain.OutfitPhoto{ID: "ph1", MediaKey: "outfits/o1/uuid/img.jpg"}
	outfit := outfitWithOwner("o1", "user1")
	outfit.Photos = []domain.OutfitPhoto{photo}
	repo := &mockOutfitRepo{outfits: []domain.Outfit{outfit}}
	media := &mockMediaProvider{}
	svc := newOutfitSvc(repo, &mockItemRepo{}, media)
	err := svc.DeletePhoto(t.Context(), "user1", "o1", "outfits/o1/uuid/img.jpg")
	require.NoError(t, err)
	require.Equal(t, "outfits/o1/uuid/img.jpg", media.deletedKeys[0])
	require.Equal(t, "o1", repo.deletedPhotoOutfitID)
	require.Equal(t, "outfits/o1/uuid/img.jpg", repo.deletedPhotoKey)
}

// ---- ListByDateRange ----

func TestListByDateRangeShouldReturnContextErrorWhenContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	_, err := svc.ListByDateRange(ctx, "user1", time.Now(), time.Now().Add(24*time.Hour))
	require.ErrorIs(t, err, context.Canceled)
}

func TestListByDateRangeShouldReturnValidationErrorWhenFromIsAfterTo(t *testing.T) {
	svc := newOutfitSvc(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{})
	from := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.ListByDateRange(t.Context(), "user1", from, to)
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestListByDateRangeShouldReturnRepoErrorWhenLogsFetchFails(t *testing.T) {
	logs := &mockOutfitLogRepo{listByDateRangeErr: domain.ErrIO}
	svc := NewOutfitService(&mockOutfitRepo{}, &mockItemRepo{}, &mockMediaProvider{}, logs)
	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	_, err := svc.ListByDateRange(t.Context(), "user1", from, to)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListByDateRangeShouldReturnRepoErrorWhenOutfitFetchFails(t *testing.T) {
	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

	var log1 domain.OutfitLog
	log1.ID = "l1"
	log1.OutfitID = "o1"
	log1.OwnerID = "user1"
	log1.WornOn = time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	logsRepo := &mockOutfitLogRepo{logs: []domain.OutfitLog{log1}}
	outfitsRepo := &mockOutfitRepo{getErr: domain.ErrIO}
	svc := NewOutfitService(outfitsRepo, &mockItemRepo{}, &mockMediaProvider{}, logsRepo)

	_, err := svc.ListByDateRange(t.Context(), "user1", from, to)
	require.ErrorIs(t, err, domain.ErrIO)
}

func TestListByDateRangeShouldDeduplicateOutfitsWhenMultipleLogsForSameOutfit(t *testing.T) {
	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

	outfit1 := outfitWithOwner("o1", "user1")

	var log1, log2 domain.OutfitLog
	log1.ID = "l1"
	log1.OutfitID = "o1"
	log1.OwnerID = "user1"
	log1.WornOn = time.Date(2024, 6, 5, 0, 0, 0, 0, time.UTC)
	log2.ID = "l2"
	log2.OutfitID = "o1"
	log2.OwnerID = "user1"
	log2.WornOn = time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC)

	logsRepo := &mockOutfitLogRepo{logs: []domain.OutfitLog{log1, log2}}
	outfitsRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit1}}
	svc := NewOutfitService(outfitsRepo, &mockItemRepo{}, &mockMediaProvider{}, logsRepo)

	got, err := svc.ListByDateRange(t.Context(), "user1", from, to)
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestListByDateRangeShouldSkipOutfitsNotFoundWhenDeletedAfterLog(t *testing.T) {
	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)

	var log1 domain.OutfitLog
	log1.ID = "l1"
	log1.OutfitID = "deleted-outfit"
	log1.OwnerID = "user1"
	log1.WornOn = time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	logsRepo := &mockOutfitLogRepo{logs: []domain.OutfitLog{log1}}
	outfitsRepo := &mockOutfitRepo{} // no outfits — simulates deleted outfit
	svc := NewOutfitService(outfitsRepo, &mockItemRepo{}, &mockMediaProvider{}, logsRepo)

	got, err := svc.ListByDateRange(t.Context(), "user1", from, to)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestListByDateRangeShouldReturnOutfitsWithLogsInRangeWhenSuccessful(t *testing.T) {
	from := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)
	wornOn := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	outfit1 := outfitWithOwner("o1", "user1")
	outfit2 := outfitWithOwner("o2", "user1")

	var log1 domain.OutfitLog
	log1.ID = "l1"
	log1.OutfitID = "o1"
	log1.OwnerID = "user1"
	log1.WornOn = wornOn

	logsRepo := &mockOutfitLogRepo{logs: []domain.OutfitLog{log1}}
	outfitsRepo := &mockOutfitRepo{outfits: []domain.Outfit{outfit1, outfit2}}
	svc := NewOutfitService(outfitsRepo, &mockItemRepo{}, &mockMediaProvider{}, logsRepo)

	got, err := svc.ListByDateRange(t.Context(), "user1", from, to)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "o1", got[0].GetID())
}
