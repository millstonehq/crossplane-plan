package workqueue

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

type mockProcessor struct {
	mu        sync.Mutex
	processed []int
	err       error
}

func (m *mockProcessor) ProcessPR(ctx context.Context, prNumber int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processed = append(m.processed, prNumber)
	return m.err
}

func (m *mockProcessor) getProcessed() []int {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]int, len(m.processed))
	copy(result, m.processed)
	return result
}

func TestPRWorkQueue_Enqueue(t *testing.T) {
	processor := &mockProcessor{}
	logger := logr.Discard()
	queue := NewPRWorkQueue(processor, logger, 50*time.Millisecond)
	defer queue.Shutdown()

	ctx := context.Background()

	// Enqueue PR #5
	queue.Enqueue(ctx, 5)

	// Should be pending
	if queue.PendingCount() != 1 {
		t.Errorf("expected 1 pending item, got %d", queue.PendingCount())
	}

	// Wait for debounce
	time.Sleep(100 * time.Millisecond)

	// Should be processed
	processed := processor.getProcessed()
	if len(processed) != 1 || processed[0] != 5 {
		t.Errorf("expected [5], got %v", processed)
	}

	// Should be removed from queue
	if queue.PendingCount() != 0 {
		t.Errorf("expected 0 pending items, got %d", queue.PendingCount())
	}
}

func TestPRWorkQueue_Debounce(t *testing.T) {
	processor := &mockProcessor{}
	logger := logr.Discard()
	queue := NewPRWorkQueue(processor, logger, 50*time.Millisecond)
	defer queue.Shutdown()

	ctx := context.Background()

	// Enqueue PR #5 multiple times rapidly
	for i := 0; i < 5; i++ {
		queue.Enqueue(ctx, 5)
		time.Sleep(10 * time.Millisecond) // Less than debounce
	}

	// Should still be pending (timer keeps resetting)
	if queue.PendingCount() != 1 {
		t.Errorf("expected 1 pending item, got %d", queue.PendingCount())
	}

	// Wait for final debounce
	time.Sleep(100 * time.Millisecond)

	// Should be processed exactly once
	processed := processor.getProcessed()
	if len(processed) != 1 {
		t.Errorf("expected 1 processing call, got %d: %v", len(processed), processed)
	}
}

func TestPRWorkQueue_MultiplePRs(t *testing.T) {
	processor := &mockProcessor{}
	logger := logr.Discard()
	queue := NewPRWorkQueue(processor, logger, 50*time.Millisecond)
	defer queue.Shutdown()

	ctx := context.Background()

	// Enqueue multiple PRs
	queue.Enqueue(ctx, 5)
	queue.Enqueue(ctx, 10)
	queue.Enqueue(ctx, 15)

	// All should be pending
	if queue.PendingCount() != 3 {
		t.Errorf("expected 3 pending items, got %d", queue.PendingCount())
	}

	// Wait for debounce
	time.Sleep(100 * time.Millisecond)

	// All should be processed
	processed := processor.getProcessed()
	if len(processed) != 3 {
		t.Errorf("expected 3 processing calls, got %d: %v", len(processed), processed)
	}

	// Verify all PRs were processed (order may vary)
	seen := make(map[int]bool)
	for _, pr := range processed {
		seen[pr] = true
	}
	if !seen[5] || !seen[10] || !seen[15] {
		t.Errorf("not all PRs were processed: %v", processed)
	}
}

func TestPRWorkQueue_Shutdown(t *testing.T) {
	processor := &mockProcessor{}
	logger := logr.Discard()
	queue := NewPRWorkQueue(processor, logger, 200*time.Millisecond)

	ctx := context.Background()

	// Enqueue PR
	queue.Enqueue(ctx, 5)

	// Shutdown before debounce completes
	queue.Shutdown()

	// Should be cleared
	if queue.PendingCount() != 0 {
		t.Errorf("expected 0 pending items after shutdown, got %d", queue.PendingCount())
	}

	// Wait to ensure it doesn't process
	time.Sleep(300 * time.Millisecond)

	// Should not have processed
	processed := processor.getProcessed()
	if len(processed) != 0 {
		t.Errorf("expected no processing after shutdown, got %v", processed)
	}
}
