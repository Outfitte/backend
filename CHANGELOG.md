# Changelog

All notable changes to this project will be documented in this file.
Keep a Changelog 1.0.0 / semver — see <https://keepachangelog.com>.

> **0.x stability notice:** Outfitte is pre-1.0. The REST API contract may
> change between minor releases. Upgrade notes will appear in this file.

## [Unreleased]

## [0.1.0] - 2026-06-09

### Added

- **Wardrobe items** — create, view, edit, and delete clothing items with title,
  brand, colour, category, purchase details, seller URL, and an archive/dispose
  lifecycle.
- **Item photos** — attach and remove photos for any item.
- **Locations** — organise items with hierarchical storage locations (e.g. a
  drawer inside a wardrobe, a shelf inside a room).
- **Wear logging** — record when individual items were worn and view the full
  wear history per item.
- **Outfits** — build outfits from multiple items, attach photos, and log outfit
  wear events on a calendar.
- **Outfit calendar** — list, update, and delete outfit-wear log entries; logging
  an outfit automatically creates a wear-log entry for each item in it.
- **Sharing** — share individual items, outfits, and locations with other users
  on the same instance (read-only access for recipients).
- **Item transfers** — transfer item ownership to another user; the recipient
  can accept or reject the transfer, and the sender can cancel it while pending.
- **Multi-user / family support** — multiple user accounts per self-hosted
  instance with role-based access (admin / member).
- **REST API** — full OpenAPI 3.1 specification with a rendered HTML reference.
- **Docker Compose self-hosting** — run a complete Outfitte instance (API +
  SQLite + local media storage) with a single `docker compose up` via the
  companion [outfitte/deploy](https://github.com/Outfitte/deploy) repository.

[Unreleased]: https://github.com/Outfitte/backend/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Outfitte/backend/releases/tag/v0.1.0
