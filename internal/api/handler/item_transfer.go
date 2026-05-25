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

// parseStatusFilter parses the optional ?status= query param.
// Returns nil filter if the param is absent; writes 400 and returns false if invalid.
func parseStatusFilter(w http.ResponseWriter, r *http.Request) (*domain.TransferStatus, bool) {
	raw := r.URL.Query().Get("status")
	if raw == "" {
		return nil, true
	}
	s := domain.TransferStatus(raw)
	switch s {
	case domain.TransferStatusPending, domain.TransferStatusAccepted,
		domain.TransferStatusRejected, domain.TransferStatusCancelled:
		return &s, true
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status"})
		return nil, false
	}
}

func toTransferResponses(views []service.TransferView) []transferResponse {
	resp := make([]transferResponse, len(views))
	for i, v := range views {
		resp[i] = toTransferResponse(v)
	}
	return resp
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

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	var req createTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.ItemID == "" || req.RecipientID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "item_id and recipient_id are required"})
		return
	}

	view, err := h.transfers.Create(ctx, callerID, service.CreateTransferInput{
		ItemID:          req.ItemID,
		RecipientID:     req.RecipientID,
		TransferHistory: req.TransferHistory,
	})
	if err != nil {
		if errors.Is(err, domain.ErrSelfTransfer) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "cannot transfer to yourself"})
			return
		}
		if errors.Is(err, domain.ErrConflict) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "item has a pending transfer"})
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
		if errors.Is(err, domain.ErrValidation) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "invalid request"})
			return
		}
		log.ErrorContext(ctx, "create transfer failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "transfer_id", view.Transfer.GetID())
	writeJSON(w, http.StatusCreated, toTransferResponse(view))
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

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	transferID := r.PathValue("id")
	view, err := h.transfers.Get(ctx, callerID, transferID)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		log.ErrorContext(ctx, "get transfer failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "transfer_id", transferID)
	writeJSON(w, http.StatusOK, toTransferResponse(view))
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

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	statusFilter, ok := parseStatusFilter(w, r)
	if !ok {
		return
	}

	views, err := h.transfers.ListOutgoing(ctx, callerID, statusFilter)
	if err != nil {
		log.ErrorContext(ctx, "list outgoing transfers failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "count", len(views))
	writeJSON(w, http.StatusOK, toTransferResponses(views))
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

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	statusFilter, ok := parseStatusFilter(w, r)
	if !ok {
		return
	}

	views, err := h.transfers.ListIncoming(ctx, callerID, statusFilter)
	if err != nil {
		log.ErrorContext(ctx, "list incoming transfers failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
		return
	}

	log.InfoContext(ctx, "succeeded", "count", len(views))
	writeJSON(w, http.StatusOK, toTransferResponses(views))
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

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	transferID := r.PathValue("id")
	view, err := h.transfers.Accept(ctx, callerID, transferID)
	if err != nil {
		h.handleTransferActionError(ctx, w, log, "accept transfer failed", err)
		return
	}

	log.InfoContext(ctx, "succeeded", "transfer_id", transferID)
	writeJSON(w, http.StatusOK, toTransferResponse(view))
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

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	transferID := r.PathValue("id")
	view, err := h.transfers.Reject(ctx, callerID, transferID)
	if err != nil {
		h.handleTransferActionError(ctx, w, log, "reject transfer failed", err)
		return
	}

	log.InfoContext(ctx, "succeeded", "transfer_id", transferID)
	writeJSON(w, http.StatusOK, toTransferResponse(view))
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

	callerID, ok := callerIDFromContext(ctx, w, log)
	if !ok {
		return
	}

	transferID := r.PathValue("id")
	view, err := h.transfers.Cancel(ctx, callerID, transferID)
	if err != nil {
		h.handleTransferActionError(ctx, w, log, "cancel transfer failed", err)
		return
	}

	log.InfoContext(ctx, "succeeded", "transfer_id", transferID)
	writeJSON(w, http.StatusOK, toTransferResponse(view))
}

// handleTransferActionError maps domain errors to HTTP responses for Accept/Reject/Cancel.
func (h *ItemTransferHandler) handleTransferActionError(ctx context.Context, w http.ResponseWriter, log *slog.Logger, msg string, err error) {
	if errors.Is(err, domain.ErrForbidden) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	if errors.Is(err, domain.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	if errors.Is(err, domain.ErrValidation) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "invalid request"})
		return
	}
	if errors.Is(err, domain.ErrConflict) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "item has a pending transfer"})
		return
	}
	log.ErrorContext(ctx, msg, "error", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}
