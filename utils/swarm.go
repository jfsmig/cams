package utils

import (
	"context"
	"math"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
)

type SwarmFunc func(context.Context)

type Swarm interface {
	Cancel()
	Wait()
	Run(cb SwarmFunc)
	Count() uint32
}

func SwarmRun(ctx0 context.Context, callbacks ...SwarmFunc) {
	Logger.Trace().Str("action", "spawn").Strs("f", funcNames(callbacks...)).Msg("swarm")

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
	active uint32
}

func newRealSwarm(ctx context.Context) *realSwarm {
	ctx2, cancel := context.WithCancel(ctx)
	return &realSwarm{
		wg:     sync.WaitGroup{},
		ctx:    ctx2,
		cancel: cancel,
	}
}

func NewSwarm(ctx context.Context) Swarm { return newRealSwarm(ctx) }

func (s *realSwarm) Cancel() { s.cancel() }

func (s *realSwarm) Wait() { s.wg.Wait() }

func (s *realSwarm) Count() uint32 { return atomic.LoadUint32(&s.active) }

const (
	MinusOne uint32 = math.MaxUint32 - 1
)

func funcName(f SwarmFunc) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

func funcNames(allFuncs ...SwarmFunc) []string {
	names := make([]string, 0, len(allFuncs))
	for _, f := range allFuncs {
		names = append(names, funcName(f))
	}
	return names
}

func (s *realSwarm) Run(cb SwarmFunc) {
	s.runMaybeLog(cb, true)
}

func (s *realSwarm) runMaybeLog(cb SwarmFunc, log bool) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		atomic.AddUint32(&s.active, 1)
		defer atomic.AddUint32(&s.active, MinusOne)
		cb(s.ctx)
	}()
}

func GroupRun(ctx0 context.Context, callbacks ...SwarmFunc) {
	Logger.Trace().Str("action", "spawn").Strs("f", funcNames(callbacks...)).Msg("group")

	s := NewGroup(ctx0)
	defer s.Cancel() // avoids a leak
	defer s.Wait()
	for _, cb := range callbacks {
		s.Run(cb)
	}
}

type realGroup struct {
	swarm realSwarm
}

func NewGroup(ctx context.Context) Swarm { return &realGroup{*newRealSwarm(ctx)} }

func (s *realGroup) Cancel() { s.swarm.Cancel() }

func (s *realGroup) Wait() { s.swarm.Wait() }

func (s *realGroup) Count() uint32 { return s.swarm.Count() }

func (s *realGroup) Run(cb SwarmFunc) {
	s.swarm.runMaybeLog(func(ctx context.Context) {
		// Whatever the exit cause of the cb, this cancellation triggers the
		// exit of all the other cb of the Group
		defer s.Cancel()
		cb(ctx)
	}, false)
}
