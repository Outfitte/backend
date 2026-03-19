// Package server constructs and configures the HTTP server.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/outfitte/outfitte/internal/api/handler"
	"github.com/outfitte/outfitte/internal/api/middleware"
	"github.com/outfitte/outfitte/internal/config"
	"github.com/outfitte/outfitte/internal/domain"
	"github.com/outfitte/outfitte/internal/ports"
	"github.com/outfitte/outfitte/internal/service"
)

// New builds a configured *http.Server from cfg, logger, and adapter instances.
func New(
	cfg *config.Config,
	logger *slog.Logger,
	users ports.StorageProvider[domain.User],
	sessions ports.StorageProvider[domain.Session],
	items ports.StorageProvider[domain.Item],
	locations ports.StorageProvider[domain.Location],
	settings ports.SingletonStore[domain.AppSettings],
	media ports.MediaProvider,
) *http.Server {
	userSvc := service.NewUserService(users, settings)
	authSvc := service.NewAuthService(users, sessions, []byte(cfg.JWTSecret))
	categorySvc := service.NewCategoryService()
	itemSvc := service.NewItemService(items, media, locations, categorySvc)
	locationSvc := service.NewLocationService(locations, items)

	authMiddleware := middleware.NewAuthMiddleware([]byte(cfg.JWTSecret))

	authHandler := handler.NewAuthHandler(userSvc, authSvc, authSvc, authSvc, logger)
	itemHandler := handler.NewItemHandler(itemSvc, logger)
	locationHandler := handler.NewLocationHandler(locationSvc, logger)
	categoryHandler := handler.NewCategoryHandler(categorySvc, logger)
	mediaHandler := handler.NewMediaHandler(media, logger)
	settingsHandler := handler.NewSettingsHandler(userSvc, logger)

	auth := authMiddleware.Authenticate
	admin := func(h http.Handler) http.Handler {
		return auth(middleware.RequireAdmin(h))
	}

	mux := http.NewServeMux()
	mux.Handle("GET /health", handler.NewHealthHandler(logger))

	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/refresh", authHandler.Refresh)
	mux.HandleFunc("POST /auth/logout", authHandler.Logout)

	mux.Handle("GET /items", auth(http.HandlerFunc(itemHandler.List)))
	mux.Handle("POST /items", auth(http.HandlerFunc(itemHandler.Create)))
	mux.Handle("GET /items/{id}", auth(http.HandlerFunc(itemHandler.GetByID)))
	mux.Handle("PATCH /items/{id}", auth(http.HandlerFunc(itemHandler.Update)))
	mux.Handle("DELETE /items/{id}", auth(http.HandlerFunc(itemHandler.Delete)))
	mux.Handle("POST /items/{id}/photos", auth(http.HandlerFunc(itemHandler.UploadPhoto)))
	mux.Handle("DELETE /items/{id}/photos/{key...}", auth(http.HandlerFunc(itemHandler.DeletePhoto)))
	mux.Handle("PATCH /items/{id}/location", auth(http.HandlerFunc(itemHandler.AssignLocation)))

	mux.Handle("GET /locations", auth(http.HandlerFunc(locationHandler.List)))
	mux.Handle("POST /locations", auth(http.HandlerFunc(locationHandler.Create)))
	mux.Handle("GET /locations/{id}", auth(http.HandlerFunc(locationHandler.GetByID)))
	mux.Handle("PATCH /locations/{id}", auth(http.HandlerFunc(locationHandler.Update)))
	mux.Handle("DELETE /locations/{id}", auth(http.HandlerFunc(locationHandler.Delete)))
	mux.Handle("PATCH /locations/{id}/move", auth(http.HandlerFunc(locationHandler.Move)))

	mux.Handle("GET /categories", auth(http.HandlerFunc(categoryHandler.List)))

	mux.Handle("GET /media/{key...}", auth(http.HandlerFunc(mediaHandler.Download)))

	mux.Handle("GET /admin/settings", admin(http.HandlerFunc(settingsHandler.GetSettings)))
	mux.Handle("PATCH /admin/settings", admin(http.HandlerFunc(settingsHandler.UpdateSettings)))

	return &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: mux,
	}
}

// Run listens on srv's configured address and shuts down when ctx is done.
func Run(ctx context.Context, srv *http.Server) error {
	l, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return serve(ctx, srv, l)
}

// serve runs srv on l, shutting down gracefully when ctx is done.
func serve(ctx context.Context, srv *http.Server, l net.Listener) error {
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()
	if err := srv.Serve(l); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
