package engine

import (
	"fmt"
	"sync"
)

type ProgressEvent struct {
	Status       string  `json:"status"`
	CurrentFile  string  `json:"currentFile"`
	FilePercent  float64 `json:"filePercent"`
	TotalPercent float64 `json:"totalPercent"`
	Speed        string  `json:"speed"`
}

type ErrorEvent struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Fatal   bool   `json:"fatal"`
}

type EventEmitter interface {
	EmitProgress(evt ProgressEvent)
	EmitError(evt ErrorEvent)
}

type Engine struct {
	mu    sync.RWMutex
	state State

	emitter EventEmitter
}

func New(emitter EventEmitter) *Engine {
	return &Engine{
		state:   StateIdle,
		emitter: emitter,
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

func (e *Engine) EmitProgress(evt ProgressEvent) {
	if e.emitter != nil {
		e.emitter.EmitProgress(evt)
	}
}

func (e *Engine) EmitError(evt ErrorEvent) {
	if e.emitter != nil {
		e.emitter.EmitError(evt)
	}
}
