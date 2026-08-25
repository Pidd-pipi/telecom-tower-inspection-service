# Verification

The enterprise layout migration was verified on 2026-08-22.

- `gofmt -w .`: passed from the project root.
- `cd backend && go test ./...`: passed, including the `httpapi` package and the `web` embed package.
- `cd backend && go build ./...`: passed.
- `python3 runtime_smoke.py .`: passed; `GET /healthz` returned HTTP 200 while running `go run .` with `backend` as the workdir.
- The smoke process exited cleanly and no listener remained on port 8080.
