package main

import (
	"sync"
	"testing"

	"github.com/diviatrix/GORO-Patcher/pkg/engine"
)

func TestRepairFlagsAndConfigConcurrent(t *testing.T) {
	a := &App{
		config: &engine.Config{ManifestURL: "https://example.com/m", ExeName: "ragexe.exe"},
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(4)
		go func() {
			defer wg.Done()
			a.NeedsRepair()
		}()
		go func() {
			defer wg.Done()
			a.setRepairNeeded(7)
		}()
		go func() {
			defer wg.Done()
			a.clearRepair()
		}()
		go func() {
			defer wg.Done()
			a.configSnapshot()
		}()
	}
	wg.Wait()
}

func TestConfigSnapshotStableUnderSetExeName(t *testing.T) {
	a := &App{
		gamePath: t.TempDir(),
		config:   &engine.Config{ManifestURL: "u", ExeName: "ragexe.exe"},
	}

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			a.configSnapshot()
		}()
		go func() {
			defer wg.Done()
			if err := a.SetExeName("ragexe.exe"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}