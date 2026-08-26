package engine

import (
	"path/filepath"

	"github.com/diviatrix/GORO-Patcher/pkg/grf"
)

type GRFCheckResult struct {
	Name    string   `json:"name"`
	Checked int      `json:"checked"`
	Failed  []string `json:"failed"`
	OK      bool     `json:"ok"`
	Error   string   `json:"error,omitempty"`
}

type GRFCheckReport struct {
	Full bool             `json:"full"`
	GRFs []GRFCheckResult `json:"grfs"`
	OK   bool             `json:"ok"`
}

func GRFTargets(state *LocalState, manifest *Manifest) []string {
	if state == nil || manifest == nil {
		return nil
	}
	found := make(map[string]bool)
	var targets []string
	for _, a := range state.AppliedPatches {
		p := FindPatchByID(manifest.Patches, a.ID)
		if p == nil || p.Type != "grf" || p.Target == "" {
			continue
		}
		if found[p.Target] {
			continue
		}
		found[p.Target] = true
		targets = append(targets, p.Target)
	}
	return targets
}

func CheckGRFIntegrity(gamePath, grfName string) GRFCheckResult {
	res := GRFCheckResult{Name: grfName}
	safe, err := SafePatchComponent(grfName)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	g, err := grf.Open(filepath.Join(gamePath, safe))
	if err != nil {
		res.Error = err.Error()
		return res
	}
	defer g.Close()

	res.Checked = g.FileCount()
	for _, name := range g.ListFiles() {
		if _, err := g.ReadFile(name); err != nil {
			res.Failed = append(res.Failed, name)
		}
	}
	res.OK = len(res.Failed) == 0
	return res
}
