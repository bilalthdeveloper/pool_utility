package main

import (
	"context"
	"sort"
	"sync"
	"time"
)

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
	workerStack    []int
	maxWorkers     int
	minWorkers     int
	taskQueue      chan Task
	taskQueueSize  int
	retryCount     int
	lock           sync.Locker
	cond           *sync.Cond
	timeout        time.Duration
	resultCallback func(interface{})
	errorCallback  func(error)
	adjustInterval time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
}

func NewPool(maxWorkers int, opts ...Option) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &Pool{
		maxWorkers:     maxWorkers,
		minWorkers:     maxWorkers,
		workers:        nil,
		workerStack:    nil,
		taskQueue:      nil,
		taskQueueSize:  1e6,
		retryCount:     0,
		lock:           new(sync.Mutex),
		timeout:        0,
		adjustInterval: 1 * time.Second,
		ctx:            ctx,
		cancel:         cancel,
	}
	for _, opt := range opts {
		opt(pool)
	}

	pool.taskQueue = make(chan Task, pool.taskQueueSize)
	pool.workers = make([]*Worker, pool.minWorkers)
	pool.workerStack = make([]int, pool.minWorkers)

	if pool.cond == nil {
		pool.cond = sync.NewCond(pool.lock)
	}
	for i := 0; i < pool.minWorkers; i++ {
		worker := newWorker()
		pool.workers[i] = worker
		pool.workerStack[i] = i
		worker.start(pool, i)
	}
	go pool.Adjust()
	go pool.Dispatch()
	return pool
}
func (p *Pool) Wait() {
	for {
		p.lock.Lock()
		workerStackLen := len(p.workerStack)
		p.lock.Unlock()

		if len(p.taskQueue) == 0 && workerStackLen == len(p.workers) {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func (p *Pool) AddTask(t Task) {
	p.taskQueue <- t
}

func (p *Pool) Release() {
	close(p.taskQueue)
	p.cancel()
	p.cond.L.Lock()
	for len(p.workerStack) != p.minWorkers {
		p.cond.Wait()
	}
	p.cond.L.Unlock()
	for _, worker := range p.workers {
		close(worker.taskQueue)
	}
	p.workers = nil
	p.workerStack = nil
}

func (p *Pool) Pop() int {
	p.lock.Lock()
	workerIndex := p.workerStack[len(p.workerStack)-1]
	p.workerStack = p.workerStack[:len(p.workerStack)-1]
	p.lock.Unlock()
	return workerIndex
}

func (p *Pool) Push(workerIndex int) {
	p.lock.Lock()
	p.workerStack = append(p.workerStack, workerIndex)
	p.lock.Unlock()
	p.cond.Signal()
}

func (p *Pool) Adjust() {
	ticker := time.NewTicker(p.adjustInterval)
	defer ticker.Stop()

	var adjustFlag bool

	for {
		adjustFlag = false
		select {
		case <-ticker.C:
			p.cond.L.Lock()
			if len(p.taskQueue) > len(p.workers)*3/4 && len(p.workers) < p.maxWorkers {
				adjustFlag = true

				newWorkers := min(len(p.workers)*2, p.maxWorkers) - len(p.workers)
				for i := 0; i < newWorkers; i++ {
					worker := newWorker()
					p.workers = append(p.workers, worker)

					p.workerStack = append(p.workerStack, len(p.workers)-1)
					worker.start(p, len(p.workers)-1)
				}
			} else if len(p.taskQueue) == 0 && len(p.workerStack) == len(p.workers) && len(p.workers) > p.minWorkers {
				adjustFlag = true

				removeWorkers := (len(p.workers) - p.minWorkers + 1) / 2

				sort.Ints(p.workerStack)
				p.workers = p.workers[:len(p.workers)-removeWorkers]
				p.workerStack = p.workerStack[:len(p.workerStack)-removeWorkers]
			}
			p.cond.L.Unlock()
			if adjustFlag {
				p.cond.Broadcast()
			}
		case <-p.ctx.Done():
			return
		}
	}
}
func (p *Pool) Dispatch() {
	for t := range p.taskQueue {
		p.cond.L.Lock()
		for len(p.workerStack) == 0 {
			p.cond.Wait()
		}
		p.cond.L.Unlock()
		workerIndex := p.Pop()
		p.workers[workerIndex].taskQueue <- t
	}
}

func (p *Pool) Running() int {
	p.lock.Lock()
	defer p.lock.Unlock()
	return len(p.workers) - len(p.workerStack)
}

func (p *Pool) Count() int {
	p.lock.Lock()
	defer p.lock.Unlock()
	return len(p.workers)
}

func (p *Pool) GetTaskQueueSize() int {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.taskQueueSize
}
