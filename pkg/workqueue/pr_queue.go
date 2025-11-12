package workqueue

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// PRWorkQueue manages debounced processing of PR preview resources
type PRWorkQueue struct {
	pending   map[int]*prWork
	mu        sync.Mutex
	processor PRProcessor
	logger    logr.Logger
	debounce  time.Duration
}

// prWork represents pending work for a PR
type prWork struct {
	prNumber    int
	lastEventAt time.Time
	timer       *time.Timer
	mu          sync.Mutex
}

// PRProcessor is the callback interface for processing a PR's resources
type PRProcessor interface {
	ProcessPR(ctx context.Context, prNumber int) error
}

// NewPRWorkQueue creates a new PR work queue with the specified debounce duration
func NewPRWorkQueue(processor PRProcessor, logger logr.Logger, debounce time.Duration) *PRWorkQueue {
	return &PRWorkQueue{
		pending:   make(map[int]*prWork),
		processor: processor,
		logger:    logger,
		debounce:  debounce,
	}
}

// Enqueue adds or updates a PR in the work queue
// If the PR is already queued, it resets the debounce timer
func (q *PRWorkQueue) Enqueue(ctx context.Context, prNumber int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	work, exists := q.pending[prNumber]
	if !exists {
		// Create new work item
		work = &prWork{
			prNumber:    prNumber,
			lastEventAt: time.Now(),
		}
		q.pending[prNumber] = work

		q.logger.V(1).Info("Enqueued PR for processing", "prNumber", prNumber, "debounce", q.debounce)
	} else {
		// Reset existing timer
		work.mu.Lock()
		if work.timer != nil {
			work.timer.Stop()
		}
		work.lastEventAt = time.Now()
		work.mu.Unlock()

		q.logger.V(1).Info("Reset debounce timer for PR", "prNumber", prNumber)
	}

	// Start debounce timer
	work.mu.Lock()
	work.timer = time.AfterFunc(q.debounce, func() {
		q.processPR(ctx, prNumber)
	})
	work.mu.Unlock()
}

// processPR executes the processor callback and removes the work item
func (q *PRWorkQueue) processPR(ctx context.Context, prNumber int) {
	q.mu.Lock()
	work, exists := q.pending[prNumber]
	if !exists {
		q.mu.Unlock()
		return
	}
	delete(q.pending, prNumber)
	q.mu.Unlock()

	q.logger.Info("Processing PR after debounce",
		"prNumber", prNumber,
		"lastEventAge", time.Since(work.lastEventAt),
	)

	if err := q.processor.ProcessPR(ctx, prNumber); err != nil {
		q.logger.Error(err, "Failed to process PR", "prNumber", prNumber)
		// Note: We don't re-queue on error. Periodic reconciliation will catch it.
	}
}

// Shutdown stops all pending timers
func (q *PRWorkQueue) Shutdown() {
	q.mu.Lock()
	defer q.mu.Unlock()

	for prNumber, work := range q.pending {
		work.mu.Lock()
		if work.timer != nil {
			work.timer.Stop()
		}
		work.mu.Unlock()
		q.logger.Info("Cancelled pending work", "prNumber", prNumber)
	}

	q.pending = make(map[int]*prWork)
}

// PendingCount returns the number of PRs currently in the queue
func (q *PRWorkQueue) PendingCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.pending)
}
