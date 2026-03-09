# Outfitte

[![Build Status](https://img.shields.io/badge/status-early%20development-orange)](#project-status)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

## What is Outfitte?

Outfitte is a modern web application for outfit and wardrobe management. It enables users to organize their clothing collection, create outfit combinations, and discover styling recommendations. The project is built with a clean, domain-driven architecture and is currently in early-stage development (M0 — Foundation phase).

## Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)

### Running with Docker Compose

1. **Clone the repository:**
   ```bash
   git clone https://github.com/outfitte/outfitte.git
   cd outfitte
   ```

2. **Create an environment file:**
   ```bash
   cp .env.example .env
   ```

3. **Start the application:**
   ```bash
   docker compose up
   ```

The application will be available at `http://localhost:8080` (or the port configured via `APP_PORT`).

### Stopping the application:
```bash
docker compose down
```

## Environment Variables

The following environment variables are required to run Outfitte:

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `APP_PORT` | HTTP server port | `8080` | `8080` |
| `APP_ENV` | Application environment | `development` | `development` or `production` |
| `STORAGE_PATH` | Path for persistent data storage | `./data` | `/data/storage` |
| `MEDIA_PATH` | Path for media uploads | `./media` | `/data/media` |
| `LOG_LEVEL` | Logging verbosity | `info` | `debug`, `info`, `warn`, `error` |

A `.env.example` file is provided in the repository with sensible defaults. Copy it to `.env` and adjust as needed for your environment.

## Project Status

**⚠️ Early Development (M0 — Foundation)**

Outfitte is in the foundation phase of development. This release focuses on:
- Core architecture and scaffolding
- Docker containerization
- Health checks and deployment readiness

User-facing features are not yet available. The API and feature set are subject to rapid change as the project matures.

### Roadmap

See the [Issues](https://github.com/outfitte/outfitte/issues) board for planned milestones and tasks.

## Development

### Building from source:

```bash
go mod tidy
go build -o bin/outfitte ./cmd/server
./bin/outfitte
```

### Health Check

The application exposes a health check endpoint at `GET /healthz`:

```bash
curl http://localhost:8080/healthz
```

## Contributing

We welcome contributions! Please see our contributing guidelines (coming soon) for more details.

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

## Support

For issues, questions, or feature requests, please open an [issue on GitHub](https://github.com/outfitte/outfitte/issues).
