package banner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"port-scanner/internal/domain"
)

// BannerGrabJob represents a banner grabbing job
type BannerGrabJob struct {
	IP       string
	Port     int
	Priority int
	Result   chan *BannerGrabResult
}

// BannerGrabResult represents the result of a banner grab job
type BannerGrabResult struct {
	BannerInfo *domain.BannerInfo
	Error      error
	Duration   time.Duration
}

// ZGrabWorkerPool manages a pool of ZGrab2 workers
type ZGrabWorkerPool struct {
	workers      int
	jobQueue     chan *BannerGrabJob
	timeout      time.Duration
	zgrabService *ZGrabBannerService
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewZGrabWorkerPool creates a new ZGrab2 worker pool
func NewZGrabWorkerPool(workers int, timeout time.Duration) *ZGrabWorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &ZGrabWorkerPool{
		workers:      workers,
		jobQueue:     make(chan *BannerGrabJob, workers*2), // Buffer for job queue
		timeout:      timeout,
		zgrabService: NewZGrabBannerService(timeout),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Start workers
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker processes banner grab jobs
func (p *ZGrabWorkerPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case job := <-p.jobQueue:
			if job == nil {
				return // Shutdown signal
			}

			// Process the job
			start := time.Now()
			bannerInfo, err := p.zgrabService.GetBanner(job.IP, job.Port)
			duration := time.Since(start)

			// Send result
			job.Result <- &BannerGrabResult{
				BannerInfo: bannerInfo,
				Error:      err,
				Duration:   duration,
			}

		case <-p.ctx.Done():
			return
		}
	}
}

// SubmitJob submits a banner grab job to the pool
func (p *ZGrabWorkerPool) SubmitJob(ip string, port int, priority int) (*BannerGrabResult, error) {
	// Create job
	job := &BannerGrabJob{
		IP:       ip,
		Port:     port,
		Priority: priority,
		Result:   make(chan *BannerGrabResult, 1),
	}

	// Submit job with timeout
	select {
	case p.jobQueue <- job:
		// Job submitted successfully
	case <-time.After(p.timeout):
		return nil, fmt.Errorf("job queue timeout for %s:%d", ip, port)
	case <-p.ctx.Done():
		return nil, fmt.Errorf("worker pool shutdown for %s:%d", ip, port)
	}

	// Wait for result with timeout
	select {
	case result := <-job.Result:
		return result, nil
	case <-time.After(p.timeout):
		return nil, fmt.Errorf("banner grab timeout for %s:%d", ip, port)
	case <-p.ctx.Done():
		return nil, fmt.Errorf("worker pool shutdown for %s:%d", ip, port)
	}
}

// Shutdown gracefully shuts down the worker pool
func (p *ZGrabWorkerPool) Shutdown() {
	p.cancel()

	// Send shutdown signals to all workers
	for i := 0; i < p.workers; i++ {
		p.jobQueue <- nil
	}

	// Wait for all workers to finish
	p.wg.Wait()
	close(p.jobQueue)
}

// GetStats returns pool statistics
func (p *ZGrabWorkerPool) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"workers":     p.workers,
		"queue_size":  len(p.jobQueue),
		"queue_cap":   cap(p.jobQueue),
		"active_jobs": p.workers - len(p.jobQueue),
	}
}
