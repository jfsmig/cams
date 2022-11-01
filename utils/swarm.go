package utils

import (
	"context"
	"sync"
)

type SwarmFunc func(context.Context)

type Swarm interface {
	Cancel()
	Wait()
	Run(cb SwarmFunc)
}

func SwarmRun(ctx0 context.Context, callbacks ...SwarmFunc) {
	s := NewSwarm(ctx0)
	defer s.Cancel() // avoids a leak
	defer s.Wait()
	for _, cb := range callbacks {
		s.Run(cb)
	}
}

type realSwarm struct {
	wg     sync.WaitGroup
	cancel context.CancelFunc
	ctx    context.Context
}

func NewSwarm(ctx context.Context) Swarm {
	ctx2, cancel := context.WithCancel(ctx)
	return &realSwarm{
		wg:     sync.WaitGroup{},
		ctx:    ctx2,
		cancel: cancel,
	}
}

func (s *realSwarm) Cancel() { s.cancel() }

func (s *realSwarm) Wait() { s.wg.Wait() }

func (s *realSwarm) Run(cb SwarmFunc) {
	go func() {
		// Whatever the exit cause of the cb, this cancellation triggers the
		// exit of all the other cb of the swarm
		defer s.cancel()
		s.wg.Add(1)
		cb(s.ctx)
	}()
}
