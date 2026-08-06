package engine

import "testing"

func TestStateTransitions(t *testing.T) {
	tests := []struct {
		from    State
		to      State
		allowed bool
	}{
		{StateIdle, StateChecking, true},
		{StateIdle, StateDownloading, false},
		{StateIdle, StateError, false},
		{StateChecking, StateDownloading, true},
		{StateChecking, StateReady, true},
		{StateChecking, StateError, true},
		{StateChecking, StateIdle, false},
		{StateDownloading, StatePatching, true},
		{StateDownloading, StateError, true},
		{StateDownloading, StateReady, false},
		{StatePatching, StateReady, true},
		{StatePatching, StateError, true},
		{StatePatching, StateIdle, false},
		{StateReady, StateChecking, true},
		{StateReady, StateIdle, true},
		{StateReady, StateDownloading, false},
		{StateError, StateChecking, true},
		{StateError, StateIdle, true},
		{StateError, StatePatching, false},
	}

	for _, tt := range tests {
		got := tt.from.CanTransitionTo(tt.to)
		if got != tt.allowed {
			t.Errorf("%s -> %s: got %v, want %v", tt.from, tt.to, got, tt.allowed)
		}
	}
}

func TestStateString(t *testing.T) {
	if StateIdle.String() != "IDLE" {
		t.Errorf("expected IDLE, got %s", StateIdle.String())
	}
	if State(99).String() != "UNKNOWN(99)" {
		t.Errorf("expected UNKNOWN(99), got %s", State(99).String())
	}
}

func TestSetStateInvalid(t *testing.T) {
	e := New(nil)
	err := e.SetState(StateReady)
	if err == nil {
		t.Error("expected error for invalid transition IDLE -> READY")
	}
}

func TestSetStateValid(t *testing.T) {
	e := New(nil)
	err := e.SetState(StateChecking)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if e.State() != StateChecking {
		t.Errorf("expected CHECKING, got %s", e.State())
	}
}
