package pool

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	Fatal_Error_Code = 1
	Debug_Error_Code = 2
	Log_Info         = 3
)

type Task func() (interface{}, error)
type Option func(*Pool)

func WithLock(lock sync.Locker) Option {
	return func(p *Pool) {
		p.lock = lock
		p.cond = sync.NewCond(p.lock)
	}
}

func WithMinWorkers(minWorkers int) Option {
	return func(p *Pool) {
		p.minWorkers = minWorkers
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(p *Pool) {
		p.timeout = timeout
	}
}

func WithResultCallback(callback func(interface{})) Option {
	return func(p *Pool) {
		p.resultCallback = callback
	}
}

func WithErrorCallback(callback func(error)) Option {
	return func(p *Pool) {
		p.errorCallback = callback
	}
}

func WithRetryCount(retryCount int) Option {
	return func(p *Pool) {
		p.retryCount = retryCount
	}
}

func WithTaskQueueSize(size int) Option {
	return func(p *Pool) {
		p.taskQueueSize = size
	}
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func LogMessage(message string, code int) {
	switch code {
	case Fatal_Error_Code:
		_, file, line, ok := runtime.Caller(1)
		if ok {
			fmt.Printf("\nFATAL ERROR: %s\nFile: %s, Line: %d\n", message, file, line)
		} else {
			fmt.Printf("\nFATAL ERROR: %s\n", message)
		}
		fmt.Println("The application will exit now.")
		os.Exit(1)
	case Debug_Error_Code:
		fmt.Printf("\nDEBUG: %s\n", message)
	case Log_Info:
		fmt.Printf("\nINFO: %s\n", message)
	default:
		fmt.Println("Unknown error code.")
	}
}
