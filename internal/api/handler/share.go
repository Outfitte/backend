package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/service"
)

type shareService interface {
	Create(ctx context.Context, callerID string, input service.CreateShareInput) (domain.Share, error)
	ListOutgoing(ctx context.Context, callerID string) ([]service.ShareView, error)
	ListSharedWithMe(ctx context.Context, callerID string) (service.SharedWithMeResult, error)
	Revoke(ctx context.Context, callerID, shareID string) error
}

// ShareHandler handles share-related HTTP endpoints.
type ShareHandler struct {
	shares shareService
	log    *slog.Logger
}

// NewShareHandler creates a ShareHandler with a logger pre-scoped to handler=share.
func NewShareHandler(shares shareService, log *slog.Logger) *ShareHandler {
	return &ShareHandler{shares: shares, log: log.With("handler", "share")}
}

type createShareRequest struct {
	RecipientID string                  `json:"recipient_id"`
	TargetType  domain.ShareTargetType  `json:"target_type"`
	TargetID    string                  `json:"target_id"`
}

type shareResponse struct {
	ID          string                  `json:"id"`
	RecipientID string                  `json:"recipient_id"`
	TargetType  domain.ShareTargetType  `json:"target_type"`
	TargetID    string                  `json:"target_id"`
	CreatedAt   time.Time               `json:"created_at"`
}

// Create handles POST /shares.
func (h *ShareHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Create")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	var req createShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	share, err := h.shares.Create(ctx, callerID, service.CreateShareInput{
		RecipientID: req.RecipientID,
		TargetType:  req.TargetType,
		TargetID:    req.TargetID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrSelfShare) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrDuplicateShare) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "share already exists"})
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		log.ErrorContext(ctx, "create share failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "share_id", share.ID)
	writeJSON(w, http.StatusCreated, shareResponse{
		ID:          share.ID,
		RecipientID: share.RecipientID,
		TargetType:  share.TargetType,
		TargetID:    share.TargetID,
		CreatedAt:   share.CreatedAt,
	})
}

type userSummaryResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type shareViewResponse struct {
	ID         string                 `json:"id"`
	Recipient  userSummaryResponse    `json:"recipient"`
	TargetType domain.ShareTargetType `json:"target_type"`
	TargetID   string                 `json:"target_id"`
	CreatedAt  time.Time              `json:"created_at"`
}

// ListOutgoing handles GET /shares.
func (h *ShareHandler) ListOutgoing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "ListOutgoing")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	views, err := h.shares.ListOutgoing(ctx, callerID)
	if err != nil {
		log.ErrorContext(ctx, "list outgoing shares failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	resp := make([]shareViewResponse, len(views))
	for i, v := range views {
		resp[i] = shareViewResponse{
			ID:         v.Share.ID,
			Recipient:  userSummaryResponse{ID: v.Recipient.ID, Email: v.Recipient.Email},
			TargetType: v.Share.TargetType,
			TargetID:   v.Share.TargetID,
			CreatedAt:  v.Share.CreatedAt,
		}
	}

	log.InfoContext(ctx, "succeeded", "count", len(views))
	writeJSON(w, http.StatusOK, resp)
}

type sharedItemResponse struct {
	itemResponse
	SharedBy userSummaryResponse `json:"shared_by"`
}

type sharedOutfitResponse struct {
	outfitResponse
	SharedBy userSummaryResponse `json:"shared_by"`
}

type locationResponse struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	ParentID  *string   `json:"parent_id"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
}

type sharedLocationResponse struct {
	Location locationResponse    `json:"location"`
	Items    []itemResponse      `json:"items"`
	SharedBy userSummaryResponse `json:"shared_by"`
}

type sharedWithMeResponse struct {
	Items     []sharedItemResponse     `json:"items"`
	Outfits   []sharedOutfitResponse   `json:"outfits"`
	Locations []sharedLocationResponse `json:"locations"`
}

// ListSharedWithMe handles GET /shares/with-me.
func (h *ShareHandler) ListSharedWithMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "ListSharedWithMe")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	result, err := h.shares.ListSharedWithMe(ctx, callerID)
	if err != nil {
		log.ErrorContext(ctx, "list shared with me failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	resp := buildSharedWithMeResponse(result)
	log.InfoContext(ctx, "succeeded", "item_count", len(result.Items), "outfit_count", len(result.Outfits), "location_count", len(result.Locations))
	writeJSON(w, http.StatusOK, resp)
}

func toLocationResponse(loc domain.Location) locationResponse {
	return locationResponse{
		ID:        loc.GetID(),
		OwnerID:   loc.OwnerID,
		ParentID:  loc.ParentID,
		Label:     loc.Label,
		CreatedAt: loc.CreatedAt,
	}
}

func toUserSummaryResponse(u service.UserSummary) userSummaryResponse {
	return userSummaryResponse{ID: u.ID, Email: u.Email}
}

func buildSharedWithMeResponse(result service.SharedWithMeResult) sharedWithMeResponse {
	items := make([]sharedItemResponse, len(result.Items))
	for i, se := range result.Items {
		items[i] = sharedItemResponse{
			itemResponse: toItemResponse(se.Entity),
			SharedBy:     toUserSummaryResponse(se.SharedBy),
		}
	}
	outfits := make([]sharedOutfitResponse, len(result.Outfits))
	for i, se := range result.Outfits {
		outfits[i] = sharedOutfitResponse{
			outfitResponse: toOutfitResponse(se.Entity),
			SharedBy:       toUserSummaryResponse(se.SharedBy),
		}
	}
	locations := make([]sharedLocationResponse, len(result.Locations))
	for i, sl := range result.Locations {
		locItems := make([]itemResponse, len(sl.Items))
		for j, it := range sl.Items {
			locItems[j] = toItemResponse(it)
		}
		locations[i] = sharedLocationResponse{
			Location: toLocationResponse(sl.Location),
			Items:    locItems,
			SharedBy: toUserSummaryResponse(sl.SharedBy),
		}
	}
	return sharedWithMeResponse{Items: items, Outfits: outfits, Locations: locations}
}

// Revoke handles DELETE /shares/{id}.
func (h *ShareHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Revoke")
	log.InfoContext(ctx, "started")

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	shareID := r.PathValue("id")
	if err := h.shares.Revoke(ctx, callerID, shareID); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		log.ErrorContext(ctx, "revoke share failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "share_id", shareID)
	w.WriteHeader(http.StatusNoContent)
}
