package engine

import (
	"fmt"
	"sync"
)

type Engine struct {
	mu    sync.RWMutex
	state State
}

func New() *Engine {
	return &Engine{
		state: StateIdle,
	}
}

func (e *Engine) State() State {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.state
}

func (e *Engine) SetState(next State) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.state.CanTransitionTo(next) {
		return fmt.Errorf("invalid transition: %s -> %s", e.state, next)
	}
	e.state = next
	return nil
}
