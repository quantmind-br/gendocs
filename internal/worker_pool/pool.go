package worker_pool

import (
	"context"
	"runtime"
	"sync"
)

// Task represents a unit of work to execute
type Task func(ctx context.Context) (interface{}, error)

// Result represents the result of a task execution
type Result struct {
	Value interface{}
	Error error
}

// WorkerPool executes tasks concurrently with semaphore-based limiting
type WorkerPool struct {
	maxWorkers int
	semaphore  chan struct{}
}

// DefaultMaxWorkers is the default number of workers for LLM API calls
// Using a conservative value to avoid rate limiting and connection issues
const DefaultMaxWorkers = 2

// NewWorkerPool creates a new worker pool
func NewWorkerPool(maxWorkers int) *WorkerPool {
	if maxWorkers <= 0 {
		// Use conservative default for LLM API calls
		// Higher concurrency can cause rate limiting and HTTP/2 stream errors
		maxWorkers = DefaultMaxWorkers
	}

	// Cap at reasonable maximum to prevent overwhelming the API
	maxCPU := runtime.NumCPU()
	if maxWorkers > maxCPU {
		maxWorkers = maxCPU
	}

	return &WorkerPool{
		maxWorkers: maxWorkers,
		semaphore:  make(chan struct{}, maxWorkers),
	}
}

// Run executes all tasks concurrently and returns results in order
func (wp *WorkerPool) Run(ctx context.Context, tasks []Task) []Result {
	if len(tasks) == 0 {
		return []Result{}
	}

	numTasks := len(tasks)
	results := make([]Result, numTasks)
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		go func(index int, t Task) {
			defer wg.Done()

			// Acquire semaphore (blocks if max workers already running)
			select {
			case wp.semaphore <- struct{}{}:
				defer func() { <-wp.semaphore }()
			case <-ctx.Done():
				results[index] = Result{Error: ctx.Err()}
				return
			}

			// Execute the task
			value, err := t(ctx)
			results[index] = Result{Value: value, Error: err}
		}(i, task)
	}

	wg.Wait()
	return results
}

// GetMaxWorkers returns the maximum number of workers
func (wp *WorkerPool) GetMaxWorkers() int {
	return wp.maxWorkers
}
