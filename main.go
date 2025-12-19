package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lavishpal/TCPSERVER/threadpool"
)

func main() {
	// Configure thread pool
	config := threadpool.Config{
		WorkerCount:       10,               // Number of worker goroutines
		QueueSize:         100,              // Buffered job queue size
		ConnectionTimeout: 30 * time.Second, // Max connection duration
		ReadTimeout:       10 * time.Second, // Max time to read request
		WriteTimeout:      10 * time.Second, // Max time to send response
		ShutdownTimeout:   5 * time.Second,  // Graceful shutdown timeout
		ProcessingDelay:   2 * time.Second,  // Simulated processing delay
	}

	// Create and start thread pool
	pool := threadpool.New(config)
	pool.Start()

	// Setup TCP listener
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	log.Println("TCP Server listening on :8080")
	log.Printf("Press Ctrl+C for graceful shutdown")

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start stats reporter
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				active, total, rejected, queueSize := pool.GetStats()
				log.Printf("Stats - Active: %d, Total: %d, Rejected: %d, Queue: %d/%d",
					active, total, rejected, queueSize, pool.QueueCapacity())
			}
		}
	}()

	// Connection counter
	var connID int64

	// Accept connections in a goroutine
	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Set accept deadline for graceful shutdown
			listener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))

			conn, err := listener.Accept()
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue
				}
				select {
				case <-ctx.Done():
					return
				default:
					log.Printf("Accept error: %v", err)
					continue
				}
			}

			connID++
			currentConnID := connID
			log.Printf("New connection %d from %s", currentConnID, conn.RemoteAddr())

			// Submit to thread pool
			if !pool.Submit(conn, currentConnID) {
				// Queue full, reject connection
				conn.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\nServer busy\n"))
				conn.Close()
				log.Printf("Connection %d rejected - server busy", currentConnID)
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutdown signal received, starting graceful shutdown...")

	// Cancel context to stop accepting
	cancel()

	// Wait for accept goroutine
	<-acceptDone

	// Shutdown thread pool
	if err := pool.Shutdown(); err != nil {
		log.Printf("Thread pool shutdown error: %v", err)
	}

	log.Println("Server stopped successfully")
}
