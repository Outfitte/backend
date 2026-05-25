package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/outfitte/backend/internal/domain"
	"github.com/outfitte/backend/internal/service"
)

type transferService interface {
	Create(ctx context.Context, callerID string, input service.CreateTransferInput) (service.TransferView, error)
	Get(ctx context.Context, callerID, transferID string) (service.TransferView, error)
	ListOutgoing(ctx context.Context, callerID string, status *domain.TransferStatus) ([]service.TransferView, error)
	ListIncoming(ctx context.Context, callerID string, status *domain.TransferStatus) ([]service.TransferView, error)
	Accept(ctx context.Context, callerID, transferID string) (service.TransferView, error)
	Reject(ctx context.Context, callerID, transferID string) (service.TransferView, error)
	Cancel(ctx context.Context, callerID, transferID string) (service.TransferView, error)
}

// ItemTransferHandler handles item-transfer-related HTTP endpoints.
type ItemTransferHandler struct {
	transfers transferService
	log       *slog.Logger
}

// NewItemTransferHandler creates an ItemTransferHandler with a logger pre-scoped to handler=item_transfer.
func NewItemTransferHandler(transfers transferService, log *slog.Logger) *ItemTransferHandler {
	return &ItemTransferHandler{transfers: transfers, log: log.With("handler", "item_transfer")}
}

type createTransferRequest struct {
	ItemID          string `json:"item_id"`
	RecipientID     string `json:"recipient_id"`
	TransferHistory bool   `json:"transfer_history"`
}

type transferResponse struct {
	ID              string                `json:"id"`
	Status          domain.TransferStatus `json:"status"`
	TransferHistory bool                  `json:"transfer_history"`
	Item            itemResponse          `json:"item"`
	Sender          userSummaryResponse   `json:"sender"`
	Recipient       userSummaryResponse   `json:"recipient"`
	CreatedAt       time.Time             `json:"created_at"`
	DecidedAt       *time.Time            `json:"decided_at"`
}

func toTransferResponse(v service.TransferView) transferResponse {
	return transferResponse{
		ID:              v.Transfer.GetID(),
		Status:          v.Transfer.Status,
		TransferHistory: v.Transfer.TransferHistory,
		Item:            toItemResponse(v.Item),
		Sender:          toUserSummaryResponse(v.Sender),
		Recipient:       toUserSummaryResponse(v.Recipient),
		CreatedAt:       v.Transfer.CreatedAt,
		DecidedAt:       v.Transfer.DecidedAt,
	}
}

// Create handles POST /transfers.
func (h *ItemTransferHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Create")
	log.InfoContext(ctx, "started")

	if err := ctx.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request cancelled"})
		return
	}

	errors.New("not implemented")
}

// Get handles GET /transfers/{id}.
func (h *ItemTransferHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Get")
	log.InfoContext(ctx, "started")

	if err := ctx.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request cancelled"})
		return
	}

	errors.New("not implemented")
}

// ListOutgoing handles GET /transfers/outgoing.
func (h *ItemTransferHandler) ListOutgoing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "ListOutgoing")
	log.InfoContext(ctx, "started")

	if err := ctx.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request cancelled"})
		return
	}

	errors.New("not implemented")
}

// ListIncoming handles GET /transfers/incoming.
func (h *ItemTransferHandler) ListIncoming(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "ListIncoming")
	log.InfoContext(ctx, "started")

	if err := ctx.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request cancelled"})
		return
	}

	errors.New("not implemented")
}

// Accept handles POST /transfers/{id}/accept.
func (h *ItemTransferHandler) Accept(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Accept")
	log.InfoContext(ctx, "started")

	if err := ctx.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request cancelled"})
		return
	}

	errors.New("not implemented")
}

// Reject handles POST /transfers/{id}/reject.
func (h *ItemTransferHandler) Reject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Reject")
	log.InfoContext(ctx, "started")

	if err := ctx.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request cancelled"})
		return
	}

	errors.New("not implemented")
}

// Cancel handles POST /transfers/{id}/cancel.
func (h *ItemTransferHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := h.log.With("call", "Cancel")
	log.InfoContext(ctx, "started")

	if err := ctx.Err(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "request cancelled"})
		return
	}

	errors.New("not implemented")
}
