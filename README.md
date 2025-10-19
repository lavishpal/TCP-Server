# TCP Server (Go)

A simple, production-ready TCP server in Go:

- Worker pool with limited concurrency
- Connection timeouts (read/write/overall)
- Graceful shutdown via signals
- Basic stats logging
- Configurable via constants in `main.go`

## Run

```bash
go run main.go
```

Then in another terminal:

```bash
curl http://localhost:8080
curl http://localhost:8080
```

## Project Structure

- `main.go` — Server implementation

## Notes

- Backlog tuning is mostly governed by OS (e.g., `somaxconn`). This code sets `SO_REUSEADDR` and logs desired backlog; adjust OS settings for true backlog control.
- For production, consider running behind a load balancer and add observability (metrics/tracing).
