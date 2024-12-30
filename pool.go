package pool

import (
	"context"
	"sync"
	"time"
)

type Task func(...interface{}) (interface{}, error)

type pool interface {
	AddTask(t Task)

	Wait()

	Release()

	Running() int

	GetWorkerCount() int

	GetTaskQueueSize() int
}

type Pool struct {
	workers        []*Worker
	maxWorkers     int
	taskQueue      chan Task
	taskQueueSize  int
	retryCount     int
	timeout        time.Duration
	resultCallback func(interface{})
	errorCallback  func(error)
	adjustInterval time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
	lock           sync.Locker
	cond           *sync.Cond
}

func NewPool(maxWorkers int, opts ...Option) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &Pool{
		maxWorkers: maxWorkers,
		ctx:        ctx,
		cancel:     cancel,
	}
	for _, opt := range opts {
		opt(pool)
	}

	pool.taskQueue = make(chan Task, pool.taskQueueSize)
	pool.lock = new(sync.Mutex)
	pool.cond = sync.NewCond(pool.lock)
	go pool.adjust()
	go pool.dispatch()
	return pool
}

// Manage Active Worker and Non Active Worker gracefully and scale them efficietilly
func (p *Pool) adjust() {
	ticker := time.NewTicker(p.adjustInterval)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
				queueSize := len(p.taskQueue)
				activeWorkers := len(p.workers)
				idleWorkers := p.maxWorkers - activeWorkers

				if queueSize > p.taskQueueSize/2 && activeWorkers < p.maxWorkers {
					newWorkers := activeWorkers + 1
					if newWorkers > p.maxWorkers {
						newWorkers = p.maxWorkers
					}
					go func() {
						for i := 0; i < newWorkers-activeWorkers; i++ {
							worker := NewWorker()
							p.workers = append(p.workers, worker)
						}
					}()
				}
				if queueSize == 0 && idleWorkers > 0 {
					go func() {
						for i := 0; i < idleWorkers; i++ {
							if len(p.workers) > 0 {
								worker := p.workers[len(p.workers)-1]
								worker.Stop()
								p.workers = p.workers[:len(p.workers)-1]
							}
						}
					}()
				}
			}
		}
	}()

}

func (p *Pool) dispatch() {

}

func (p *Pool) Release() {

}
