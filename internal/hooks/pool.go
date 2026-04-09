package hooks

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

const (
	defaultAsyncWorkerCount   = 4
	defaultAsyncQueueCapacity = 64
	defaultAsyncDrainTimeout  = 10 * time.Second
)

type asyncTask struct {
	hook RegisteredHook
	run  func(context.Context)
}

type asyncPoolConfig struct {
	WorkerCount   int
	QueueCapacity int
	DrainTimeout  time.Duration
	Logger        *slog.Logger
	Metrics       *hookMetrics
}

type asyncPool struct {
	logger        *slog.Logger
	workerCount   int
	queueCapacity int
	drainTimeout  time.Duration
	metrics       *hookMetrics

	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	tasks   chan asyncTask
	wg      sync.WaitGroup
	started bool
	closed  bool
}

func newAsyncPool(cfg asyncPoolConfig) *asyncPool {
	workerCount := cfg.WorkerCount
	if workerCount <= 0 {
		workerCount = defaultAsyncWorkerCount
	}

	queueCapacity := cfg.QueueCapacity
	if queueCapacity <= 0 {
		queueCapacity = defaultAsyncQueueCapacity
	}

	drainTimeout := cfg.DrainTimeout
	if drainTimeout <= 0 {
		drainTimeout = defaultAsyncDrainTimeout
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &asyncPool{
		logger:        logger,
		workerCount:   workerCount,
		queueCapacity: queueCapacity,
		drainTimeout:  drainTimeout,
		metrics:       cfg.Metrics,
	}
}

func (p *asyncPool) Start(parent context.Context) {
	if p == nil {
		return
	}
	if parent == nil {
		parent = context.Background()
	}

	p.mu.Lock()
	if p.started || p.closed {
		p.mu.Unlock()
		return
	}

	p.ctx, p.cancel = context.WithCancel(parent)
	p.tasks = make(chan asyncTask, p.queueCapacity)
	p.started = true

	workerCtx := p.ctx
	tasks := p.tasks
	workerCount := p.workerCount
	p.wg.Add(workerCount)
	p.mu.Unlock()

	for range workerCount {
		go p.worker(workerCtx, tasks)
	}
}

func (p *asyncPool) Submit(task asyncTask) bool {
	if p == nil {
		return false
	}

	p.mu.RLock()
	if !p.started || p.closed || p.tasks == nil {
		p.mu.RUnlock()
		return false
	}

	select {
	case p.tasks <- task:
		p.metrics.observeQueueDepth(len(p.tasks))
		p.mu.RUnlock()
		return true
	default:
		queueDepth := len(p.tasks)
		logger := p.logger
		p.metrics.observeAsyncDrop(queueDepth)
		p.mu.RUnlock()

		logger.Warn(
			"hook.dispatch.async_dropped",
			"hook", task.hook.Name,
			"event", task.hook.Event.String(),
			"source", task.hook.Source.String(),
			"queue_depth", queueDepth,
			"queue_capacity", p.queueCapacity,
		)
		return false
	}
}

func (p *asyncPool) Close() {
	if p == nil {
		return
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true

	if !p.started || p.tasks == nil {
		p.mu.Unlock()
		return
	}

	tasks := p.tasks
	cancel := p.cancel
	drainTimeout := p.drainTimeout
	p.mu.Unlock()

	if cancel != nil {
		defer cancel()
	}

	close(tasks)

	drainCtx, stopDrain := context.WithTimeout(context.Background(), drainTimeout)
	defer stopDrain()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return
	case <-drainCtx.Done():
		discardAsyncTasks(tasks)
		if cancel != nil {
			cancel()
		}
		<-done
	}
}

func (p *asyncPool) worker(ctx context.Context, tasks <-chan asyncTask) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-tasks:
			if !ok {
				return
			}
			p.runTask(ctx, task)
		}
	}
}

func (p *asyncPool) runTask(ctx context.Context, task asyncTask) {
	if task.run == nil {
		return
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			p.logger.ErrorContext(
				ctx,
				"hook.dispatch.async_panic",
				"hook", task.hook.Name,
				"event", task.hook.Event.String(),
				"source", task.hook.Source.String(),
				"panic", recovered,
			)
		}
	}()

	task.run(ctx)
}

func discardAsyncTasks(tasks <-chan asyncTask) {
	for {
		select {
		case _, ok := <-tasks:
			if !ok {
				return
			}
		default:
			return
		}
	}
}
