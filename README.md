# TCP Server with Thread Pool (Go)

A production-ready TCP server in Go featuring a custom thread pool implementation with worker reuse, connection queueing, and graceful shutdown.

## Features

### Thread Pool Implementation
- **Fixed worker pool**: Configurable number of worker goroutines (default: 10)
- **Worker reuse**: Workers stay alive and process multiple connections
- **Job queue**: Buffered channel (default: 100) for connection queueing
- **Backpressure handling**: Rejects connections when queue is full (503 response)
- **Atomic counters**: Thread-safe tracking of active/total connections

### Timeout Management
- **Connection timeout**: 30s max connection duration
- **Read timeout**: 10s max time to read client request
- **Write timeout**: 10s max time to send response
- **Accept timeout**: 1s deadline for graceful shutdown

### Production Features
- **Graceful shutdown**: Signal handling (SIGINT/SIGTERM) with 5s timeout
- **Stats reporting**: Every 10s logs active jobs, total processed, queue size
- **Connection ID tracking**: Each connection gets unique ID for debugging
- **Error handling**: Comprehensive logging and timeout error recovery

## Configuration

Edit constants in `main.go`:

```go
const (
    WorkerPoolSize    = 10               // Number of worker goroutines
    JobQueueSize      = 100              // Buffered job queue size
    ConnectionTimeout = 30 * time.Second // Max connection duration
    ReadTimeout       = 10 * time.Second // Max time to read request
    WriteTimeout      = 10 * time.Second // Max time to send response
    ShutdownTimeout   = 5 * time.Second  // Graceful shutdown timeout
)
```

## Quick Start

### Run the server
```bash
go run main.go
```

Output:
```
Starting thread pool with 10 workers and queue size 100
Worker 0 started
Worker 1 started
...
TCP Server listening on :8080
Press Ctrl+C for graceful shutdown
```

### Test with curl
```bash
# Single request
curl http://localhost:8080

# Multiple concurrent requests
for i in {1..5}; do curl http://localhost:8080 & done
```

### Graceful shutdown
Press `Ctrl+C` and observe:
```
Shutdown signal received, starting graceful shutdown...
Initiating thread pool shutdown...
Worker 0 shutting down
...
Thread pool shutdown completed
Server stopped successfully
```

## Architecture

```
┌─────────────────────────────────────────────────┐
│              TCP Listener (:8080)               │
└────────────────┬────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────┐
│          Job Queue (buffered channel)           │
│              Size: 100 connections              │
└────────────────┬────────────────────────────────┘
                 │
      ┌──────────┴──────────┬──────────┬─────────┐
      ▼          ▼           ▼          ▼         ▼
  Worker 0   Worker 1    Worker 2   ...    Worker 9
      │          │           │          │         │
      └──────────┴───────────┴──────────┴─────────┘
                         │
                         ▼
              handleConnection()
              (with timeouts)
```

## How It Works

1. **Listener** accepts incoming TCP connections
2. **Connection** is assigned a unique ID and submitted to job queue
3. **Available worker** picks up the job from queue
4. **Worker** processes connection with timeouts:
   - Reads request (10s timeout)
   - Processes (simulated 2s delay)
   - Writes response (10s timeout)
5. **Worker** returns to pool and waits for next job
6. **Stats reporter** logs metrics every 10 seconds

### Benefits vs Unlimited Goroutines

| Approach | Memory | CPU | Predictability |
|----------|--------|-----|----------------|
| Unlimited goroutines | High (unbounded) | High (context switching) | Poor |
| **Thread pool** | **Fixed** | **Controlled** | **Excellent** |

## Example Response

```
HTTP/1.1 200 OK
Content-Type: text/plain
Connection: close

Hello from Thread Pool Server!
Connection ID: 42
Processed at: 2025-12-20T10:30:45Z
Active jobs: 3
Total processed: 42
```

## Logs Example

```
2025/12/20 10:30:43 New connection 1 from 127.0.0.1:54321
2025/12/20 10:30:43 Connection 1 queued (queue size: 1/100)
2025/12/20 10:30:43 Worker 2 processing connection 1 from 127.0.0.1:54321
2025/12/20 10:30:43 Connection 1 received 78 bytes
2025/12/20 10:30:43 Connection 1 processing request...
2025/12/20 10:30:45 Connection 1 response sent successfully
2025/12/20 10:30:45 Worker 2 completed connection 1
2025/12/20 10:30:45 Connection 1 closed
2025/12/20 10:30:53 Stats - Active: 0, Total: 1, Queue: 0/100
```

## Project Structure

```
.
├── main.go                  # Server entry point and connection handler
├── threadpool/
│   └── threadpool.go        # Thread pool implementation
├── go.mod                   # Go module definition
├── README.md                # Documentation of the project
└── .gitignore               # Git ignore patterns
```

### Key Files

- **`main.go`**: Server setup, listener configuration, signal handling, and connection acceptance loop
- **`threadpool/threadpool.go`**: Complete thread pool implementation with:
  - `Config`: Configuration struct for pool settings
  - `ThreadPool`: Main pool structure with worker management
  - `New()`: Factory function to create a new pool
  - `Start()`: Initializes all workers
  - `Submit()`: Adds connections to the queue
  - `Shutdown()`: Graceful shutdown with timeout
  - `GetStats()`: Real-time statistics

