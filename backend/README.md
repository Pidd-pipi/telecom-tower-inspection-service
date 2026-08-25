# Telecom Tower Inspection Backend

This directory contains the Go module, HTTP entrypoint, domain packages, embedded page assets, and tests.

```bash
go test ./...
go build ./...
go run .
```

The `main` package starts the HTTP service on `PORT` (default `8080`). It exposes `GET /healthz`, `GET /api/towers`, `POST /api/towers/status`, `/`, and `/app.js`.
