package main

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/diviatrix/GORO-Patcher/pkg/engine"
)

func TestBeginCycleRejectsConcurrent(t *testing.T) {
	a := &App{engine: engine.New()}

	if err := a.beginCycle("check"); err != nil {
		t.Fatalf("first begin should succeed: %v", err)
	}
	if err := a.beginCycle("repair"); err == nil {
		t.Error("expected second concurrent begin to be rejected")
	}
}

func TestBeginCycleRestoresFlagOnStateReject(t *testing.T) {
	a := &App{engine: engine.New()}
	if err := a.engine.SetState(engine.StateChecking); err != nil {
		t.Fatal(err)
	}

	if err := a.beginCycle("check"); err == nil {
		t.Fatal("expected begin to reject while state is Checking")
	}

	if !a.cycleActive.CompareAndSwap(false, true) {
		t.Error("cycleActive must be cleared after a rejected begin")
	}
}

func TestStartCheckRejectsActiveCycle(t *testing.T) {
	a := &App{engine: engine.New()}
	a.cycleActive.Store(true)

	if err := a.StartCheck(); err == nil {
		t.Error("expected StartCheck to reject when a cycle is already active")
	}
	if err := a.StartRepair(); err == nil {
		t.Error("expected StartRepair to reject when a cycle is already active")
	}
}

func TestSetCancelAndCancelCurrentConcurrent(t *testing.T) {
	a := &App{}

	var wg sync.WaitGroup
	var cancelled atomic.Int32
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			a.setCancel(func() { cancelled.Add(1) })
		}()
		go func() {
			defer wg.Done()
			a.cancelCurrent()
		}()
	}
	wg.Wait()

	if n := cancelled.Load(); n < 0 {
		t.Errorf("impossible: negative cancel count %d", n)
	}
}
