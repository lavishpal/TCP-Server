package threadpool

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Job represents a connection to be processed by the thread pool
type Job struct {
	Conn   net.Conn
	ConnID int64
}

// Config holds thread pool configuration
type Config struct {
	WorkerCount       int           // Number of worker goroutines
	QueueSize         int           // Job queue buffer size
	ConnectionTimeout time.Duration // Max connection duration
	ReadTimeout       time.Duration // Max time to read request
	WriteTimeout      time.Duration // Max time to send response
	ShutdownTimeout   time.Duration // Graceful shutdown timeout
	ProcessingDelay   time.Duration // Simulated processing delay
}

// ThreadPool manages worker goroutines and job distribution
type ThreadPool struct {
	config       Config
	jobQueue     chan Job
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	activeJobs   int64 // Atomic counter for active jobs
	totalJobs    int64 // Atomic counter for total jobs processed
	rejectedJobs int64 // Atomic counter for rejected jobs
}

// New creates a new thread pool with the given configuration
func New(config Config) *ThreadPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &ThreadPool{
		config:   config,
		jobQueue: make(chan Job, config.QueueSize),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start initializes and starts all worker goroutines
func (tp *ThreadPool) Start() {
	log.Printf("Starting thread pool with %d workers and queue size %d",
		tp.config.WorkerCount, cap(tp.jobQueue))

	for i := 0; i < tp.config.WorkerCount; i++ {
		tp.wg.Add(1)
		go tp.worker(i)
	}

	log.Println("Thread pool started successfully")
}

// worker is the goroutine that processes jobs from the queue
func (tp *ThreadPool) worker(workerID int) {
	defer tp.wg.Done()
	log.Printf("Worker %d started", workerID)

	for {
		select {
		case <-tp.ctx.Done():
			log.Printf("Worker %d shutting down", workerID)
			return
		case job := <-tp.jobQueue:
			atomic.AddInt64(&tp.activeJobs, 1)
			log.Printf("Worker %d processing connection %d from %s",
				workerID, job.ConnID, job.Conn.RemoteAddr())

			tp.handleConnection(job.Conn, job.ConnID)

			atomic.AddInt64(&tp.activeJobs, -1)
			atomic.AddInt64(&tp.totalJobs, 1)
			log.Printf("Worker %d completed connection %d", workerID, job.ConnID)
		}
	}
}

// Submit adds a job to the queue (non-blocking with timeout)
func (tp *ThreadPool) Submit(conn net.Conn, connID int64) bool {
	select {
	case tp.jobQueue <- Job{Conn: conn, ConnID: connID}:
		log.Printf("Connection %d queued (queue size: %d/%d)",
			connID, len(tp.jobQueue), cap(tp.jobQueue))
		return true
	case <-time.After(1 * time.Second):
		atomic.AddInt64(&tp.rejectedJobs, 1)
		log.Printf("Connection %d rejected - queue full", connID)
		return false
	}
}

// handleConnection processes a single connection with timeouts
func (tp *ThreadPool) handleConnection(conn net.Conn, connID int64) {
	defer func() {
		conn.Close()
		log.Printf("Connection %d closed", connID)
	}()

	// Set overall connection deadline
	if tp.config.ConnectionTimeout > 0 {
		conn.SetDeadline(time.Now().Add(tp.config.ConnectionTimeout))
	}

	// Set read timeout
	if tp.config.ReadTimeout > 0 {
		conn.SetReadDeadline(time.Now().Add(tp.config.ReadTimeout))
	}

	// Read request
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		log.Printf("Connection %d read error: %v", connID, err)
		return
	}

	log.Printf("Connection %d received %d bytes", connID, n)
	log.Printf("Connection %d processing request...", connID)

	// Simulate processing time
	if tp.config.ProcessingDelay > 0 {
		time.Sleep(tp.config.ProcessingDelay)
	}

	// Set write timeout
	if tp.config.WriteTimeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(tp.config.WriteTimeout))
	}

	// Send response
	response := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\n"+
			"Content-Type: text/plain\r\n"+
			"Connection: close\r\n"+
			"\r\n"+
			"Hello from Thread Pool Server!\n"+
			"Connection ID: %d\n"+
			"Processed at: %s\n"+
			"Active jobs: %d\n"+
			"Total processed: %d\n"+
			"Rejected: %d\n",
		connID,
		time.Now().Format(time.RFC3339),
		atomic.LoadInt64(&tp.activeJobs),
		atomic.LoadInt64(&tp.totalJobs),
		atomic.LoadInt64(&tp.rejectedJobs),
	)

	_, err = conn.Write([]byte(response))
	if err != nil {
		log.Printf("Connection %d write error: %v", connID, err)
		return
	}

	log.Printf("Connection %d response sent successfully", connID)
}

// Shutdown gracefully shuts down the thread pool
func (tp *ThreadPool) Shutdown() error {
	log.Println("Initiating thread pool shutdown...")

	// Signal all workers to stop
	tp.cancel()

	// Close job queue
	close(tp.jobQueue)

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		tp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Thread pool shutdown completed")
		return nil
	case <-time.After(tp.config.ShutdownTimeout):
		log.Println("Thread pool shutdown timeout exceeded")
		return fmt.Errorf("shutdown timeout exceeded")
	}
}

// GetStats returns current pool statistics
func (tp *ThreadPool) GetStats() (active, total, rejected int64, queueSize int) {
	return atomic.LoadInt64(&tp.activeJobs),
		atomic.LoadInt64(&tp.totalJobs),
		atomic.LoadInt64(&tp.rejectedJobs),
		len(tp.jobQueue)
}

// QueueSize returns the current number of jobs in the queue
func (tp *ThreadPool) QueueSize() int {
	return len(tp.jobQueue)
}

// QueueCapacity returns the maximum queue capacity
func (tp *ThreadPool) QueueCapacity() int {
	return cap(tp.jobQueue)
}
