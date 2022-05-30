package main

import (
	"context"
	"github.com/qmuntal/stateless"
	"log"
)

const (
	stateInit   = "init"
	stateIdle   = "idle"
	stateActive = "active"
)

const (
	triggerStart      = "start"
	triggerActivate   = "on"
	triggerDeactivate = "off"
)

func main() {
	fsm := stateless.NewStateMachineWithMode(stateInit, stateless.FiringQueued)

	fsm.Configure(stateInit).
		OnEntry(func(ctx context.Context, args ...interface{}) error {
			log.Println("OnEntry", fsm.String())
			fsm.Activate()
			return nil
		}).
		OnActive(func(_ context.Context) error {
			log.Println("OnActive", fsm.String())
			return nil
		}).
		Permit(triggerStart, stateIdle)

	fsm.Configure(stateIdle).
		OnEntry(func(ctx context.Context, args ...interface{}) error {
			log.Println("OnEntry", fsm.String())
			fsm.Activate()
			fsm.Fire(triggerActivate)
			return nil
		}).
		OnActive(func(_ context.Context) error {
			log.Println("OnActive", fsm.String())
			return nil
		}).
		Permit(triggerActivate, stateActive)

	fsm.Configure(stateActive).
		OnEntry(func(ctx context.Context, args ...interface{}) error {
			log.Println("OnEntry", fsm.String())
			fsm.Activate()
			fsm.Fire(triggerDeactivate)
			return nil
		}).
		OnActive(func(_ context.Context) error {
			log.Println("OnActive", fsm.String())
			return nil
		}).
		Permit(triggerDeactivate, stateIdle)

	log.Println("Go!", fsm.String())
	fsm.Fire(triggerStart)
	log.Println("Bye!")
}
