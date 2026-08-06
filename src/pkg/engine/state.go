package engine

import "fmt"

type State int

const (
	StateIdle State = iota
	StateChecking
	StateDownloading
	StatePatching
	StateReady
	StateError
)

var stateNames = map[State]string{
	StateIdle:        "IDLE",
	StateChecking:    "CHECKING",
	StateDownloading: "DOWNLOADING",
	StatePatching:    "PATCHING",
	StateReady:       "READY",
	StateError:       "ERROR",
}

func (s State) String() string {
	if name, ok := stateNames[s]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN(%d)", int(s))
}

// validTransitions defines allowed state changes.
var validTransitions = map[State][]State{
	StateIdle:        {StateChecking},
	StateChecking:    {StateDownloading, StateReady, StateError},
	StateDownloading: {StatePatching, StateError},
	StatePatching:    {StateReady, StateError},
	StateReady:       {StateChecking, StateIdle},
	StateError:       {StateChecking, StateIdle},
}

// CanTransitionTo returns true if the transition from current to next is allowed.
func (s State) CanTransitionTo(next State) bool {
	allowed, ok := validTransitions[s]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == next {
			return true
		}
	}
	return false
}
