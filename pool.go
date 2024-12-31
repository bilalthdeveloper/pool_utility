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
	WorkerStack    *WorkerStack
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
	go pool.dispatch()
	go pool.adjust()

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
				var activeWorkers int
				for _, worker := range p.workers {
					if p.WorkerStack.workers[worker] {
						activeWorkers++
					}
				}
				if activeworker < p.maxWorkers {
					p.scaleDown()
				}
				if activeworker < len(p.taskQueue) {
					p.scaleUp()
				}
			}
		}
	}()

}

func (p *Pool) scaleDown() {

}

func (p *Pool) scaleUp() {

}
func (p *Pool) dispatch() {

}

func (p *Pool) Release() {

}
