# Outfitte

Self-hosted wardrobe management application built in Go.

## Overview

Outfitte lets you catalogue your clothing, organise items into locations, log wear events, and build outfit journals — all from your own infrastructure.

## Getting Started

> The project is currently in **M0 (Foundation)** — scaffolding and core infrastructure only. No user-facing features yet.

Requirements: Go 1.22+, Docker (optional)

```bash
git clone https://github.com/Outfitte/Outfitte
cd Outfitte
go build ./...
```

## Linting

Install [golangci-lint](https://golangci-lint.run/usage/install/) then run:

```bash
golangci-lint run ./...
```

## Roadmap

| Milestone | Description |
|-----------|-------------|
| M0 | Foundation — scaffold, config, health check |
| M1 | Users, Items & Locations |
| M2 | Wear & Archive Lifecycle |
| M3 | Outfits & Calendar |
| M4 | Seller URL & Price Tracking |
| M5 | Family Sharing |
| M6 | Polish & Public V1 Launch |