package main

import (
	"context"
	"sync"
	"time"
)

type ResizablePool struct {
	ctx     context.Context
	f       func(context.Context)
	workers []worker
	mu      sync.Mutex
	reset   *time.Timer
}

type worker struct {
	cancel context.CancelFunc
}

func NewResizablePool(ctx context.Context,
	f func(context.Context)) *ResizablePool {

	return &ResizablePool{
		ctx: ctx,
		f:   f,
	}
}

func (p *ResizablePool) Resize(targetCount int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// // Schedule a reset to zero workers after idle time
	// if targetCount > 0 {
	// 	if p.reset != nil {
	// 		p.reset.Stop()
	// 	}
	// 	p.reset = time.AfterFunc(5*time.Minute, func() { p.Resize(0) })
	// }

	// Remove workers
	for len(p.workers) > targetCount {
		if len(p.workers) > 0 {
			p.workers[len(p.workers)-1].cancel()
			p.workers = p.workers[:len(p.workers)-1]
		}
	}

	// Add workers
	for len(p.workers) < targetCount {
		workerCtx, cancel := context.WithCancel(p.ctx)

		worker := worker{
			cancel: cancel,
		}

		go func() {
			for {
				select {
				case <-workerCtx.Done():
					return
				default:
					p.f(workerCtx)
				}
			}
		}()

		p.workers = append(p.workers, worker)
	}
}
