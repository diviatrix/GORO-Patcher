package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/diviatrix/GORO-Patcher/pkg/detector"
	"github.com/diviatrix/GORO-Patcher/pkg/downloader"
	"github.com/diviatrix/GORO-Patcher/pkg/engine"
	"github.com/diviatrix/GORO-Patcher/pkg/grf"
	"github.com/diviatrix/GORO-Patcher/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type ProgressInfo struct {
	Status       string  `json:"status"`
	CurrentFile  string  `json:"currentFile"`
	FilePercent  float64 `json:"filePercent"`
	TotalPercent float64 `json:"totalPercent"`
	Speed        string  `json:"speed"`
}

type App struct {
	app      *application.App
	ctx      context.Context
	engine   *engine.Engine
	config   *engine.Config
	gamePath string
	dl       *downloader.Downloader
	updater  *updater.Updater
	cancelFn context.CancelFunc

	progressMu   sync.Mutex
	progressInfo ProgressInfo
	needsRepair  bool
	repairFromID int
}

func NewApp() *App {
	exe, err := os.Executable()
	if err != nil {
		exe = "."
	}
	gamePath := filepath.Dir(exe)

	return &App{
		gamePath: gamePath,
	}
}

func (a *App) SetApp(app *application.App) {
	a.app = app
}

func (a *App) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	a.ctx = ctx
	a.engine = engine.New()
	a.dl = downloader.New(3)

	cfg, err := engine.LoadConfig(a.gamePath)
	if err != nil {
		cfg = engine.DefaultConfig()
	}
	a.config = cfg

	u, err := updater.New(a.dl)
	if err == nil {
		a.updater = u
	}

	engine.CleanupStaleFiles(a.gamePath)

	state, err := engine.ReadLocalState(a.gamePath)
	if err == nil {
		version := engine.LocalVersion(state)
		if version > 0 {
			a.progressInfo = ProgressInfo{
				Status:       fmt.Sprintf("Ready (patch %d)", version),
				TotalPercent: 100,
			}
		}
	}

	return nil
}

func (a *App) ServiceShutdown(ctx context.Context) error {
	if a.cancelFn != nil {
		a.cancelFn()
	}
	return nil
}

func (a *App) GetProgress() ProgressInfo {
	a.progressMu.Lock()
	defer a.progressMu.Unlock()
	return a.progressInfo
}

func (a *App) setProgress(status, currentFile string, filePct, totalPct float64, speed string) {
	a.progressMu.Lock()
	a.progressInfo = ProgressInfo{
		Status:       status,
		CurrentFile:  currentFile,
		FilePercent:  filePct,
		TotalPercent: totalPct,
		Speed:        speed,
	}
	a.progressMu.Unlock()
}

func (a *App) NeedsRepair() bool {
	return a.needsRepair
}

func (a *App) StartCheck() error {
	state := a.engine.State()
	if state != engine.StateIdle && state != engine.StateReady && state != engine.StateError {
		return fmt.Errorf("cannot start check in state %s", state)
	}

	go a.runCheckCycle()
	return nil
}

func (a *App) StartRepair() error {
	state := a.engine.State()
	if state != engine.StateIdle && state != engine.StateReady && state != engine.StateError {
		return fmt.Errorf("cannot start repair in state %s", state)
	}

	go a.runRepairCycle()
	return nil
}

func (a *App) runCheckCycle() {
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelFn = cancel
	defer cancel()

	a.engine.SetState(engine.StateChecking)
	a.setProgress("Checking for updates...", "", 0, 0, "")

	manifest, err := a.fetchManifest(ctx)
	if err != nil {
		a.handleError("ERR_MANIFEST", err.Error(), true)
		return
	}

	if a.updater != nil && engine.NeedsSelfUpdate(manifest) {
		a.setProgress("Checking patcher update...", "", 0, 0, "")
		updated, err := a.updater.Update(ctx, manifest.PatcherURL, manifest.PatcherHash)
		if err != nil {
			log.Printf("[self-update] failed: %v", err)
		} else if updated {
			if ok, _ := a.updater.Restart(); !ok {
				a.handleError("ERR_RESTART", "New version downloaded but failed to restart.", false)
				return
			}
			os.Exit(0)
		}
	}

	a.prunePatchCache(manifest)

	localState, err := engine.ReadLocalState(a.gamePath)
	if err != nil {
		a.handleError("ERR_LOCAL_STATE", err.Error(), false)
		return
	}

	if mismatchID, ok := engine.ValidateAgainstDisk(a.gamePath, localState, manifest); !ok {
		a.needsRepair = true
		a.repairFromID = mismatchID
		a.engine.SetState(engine.StateReady)
		a.setProgress(fmt.Sprintf("Patch %d corrupted — Repair needed", mismatchID), "", 100, 100, "")
		log.Printf("[check] Mismatch at patch %d, repair needed", mismatchID)
		return
	}

	running, _ := detector.IsGameRunning(a.config.ExeName)
	if running {
		a.handleError("ERR_GAME_RUNNING", "Game is running. Please close it first.", false)
		return
	}

	queue := engine.BuildQueue(manifest, engine.LocalVersion(localState))

	totalApplied, err := a.applyPatches(ctx, manifest, queue, localState)
	if err != nil {
		return
	}

	engine.CleanupBackups(a.gamePath)

	a.needsRepair = false
	a.repairFromID = 0
	finalVersion := engine.LocalVersion(localState)
	a.engine.SetState(engine.StateReady)

	if totalApplied == 0 {
		log.Printf("[runCheckCycle] Already up to date, version=%d", finalVersion)
		a.setAlreadyUpToDate(finalVersion)
	} else {
		log.Printf("[runCheckCycle] Complete, applied %d patches, version=%d", totalApplied, finalVersion)
		a.setComplete(finalVersion)
	}
}

func (a *App) runRepairCycle() {
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelFn = cancel
	defer cancel()

	a.engine.SetState(engine.StateChecking)
	a.setProgress("Starting repair...", "", 0, 0, "")

	manifest, err := a.fetchManifest(ctx)
	if err != nil {
		a.handleError("ERR_MANIFEST", err.Error(), true)
		return
	}

	running, _ := detector.IsGameRunning(a.config.ExeName)
	if running {
		a.handleError("ERR_GAME_RUNNING", "Game is running. Please close it first.", false)
		return
	}

	localState, err := engine.ReadLocalState(a.gamePath)
	if err != nil {
		a.handleError("ERR_LOCAL_STATE", err.Error(), false)
		return
	}

	mismatchID := a.repairFromID
	if mismatchID <= 0 {

		mismatchID, _ = engine.ValidateAgainstDisk(a.gamePath, localState, manifest)
		if mismatchID < 0 {
			a.engine.SetState(engine.StateReady)
			a.setAlreadyUpToDate(engine.LocalVersion(localState))
			return
		}
	}

	queue := make([]engine.Patch, 0, len(manifest.Patches))
	for _, p := range manifest.Patches {
		if p.ID >= mismatchID {
			queue = append(queue, p)
		}
	}
	sort.Slice(queue, func(i, j int) bool { return queue[i].ID < queue[j].ID })

	var repaired []engine.LocalPatch
	for _, p := range localState.AppliedPatches {
		if p.ID < mismatchID {
			repaired = append(repaired, p)
		}
	}
	localState.AppliedPatches = repaired
	if err := engine.WriteLocalState(a.gamePath, localState); err != nil {
		a.handleError("ERR_LOCAL_STATE", fmt.Sprintf("Failed to write state: %v", err), false)
		return
	}

	totalApplied, err := a.applyPatches(ctx, manifest, queue, localState)
	if err != nil {
		return
	}

	engine.CleanupBackups(a.gamePath)

	a.needsRepair = false
	a.repairFromID = 0
	finalVersion := engine.LocalVersion(localState)
	a.engine.SetState(engine.StateReady)

	log.Printf("[repair] Complete, repaired %d patches, version=%d", totalApplied, finalVersion)
	a.setComplete(finalVersion)
}

func (a *App) fetchManifest(ctx context.Context) (*engine.Manifest, error) {
	data, err := a.dl.FetchBytes(ctx, a.config.ManifestURL)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}

	manifest, err := engine.ParseManifest(data)
	if err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	if !engine.VerifyManifestSignature(manifest) {
		return nil, fmt.Errorf("manifest signature is invalid")
	}

	return manifest, nil
}

func (a *App) applyPatches(ctx context.Context, manifest *engine.Manifest, queue []engine.Patch, localState *engine.LocalState) (int, error) {
	totalApplied := 0
	for _, patch := range queue {
		a.engine.SetState(engine.StateDownloading)
		a.setProgress(fmt.Sprintf("Downloading %s...", patch.Name), patch.Name, 0, 0, "")

		patchPath, cached, err := a.acquirePatch(ctx, manifest.PatchBaseURL, patch)
		if err != nil {
			a.handleError("ERR_DOWNLOAD", fmt.Sprintf("Failed to get %s: %v", patch.Name, err), false)
			return totalApplied, err
		}

		if !cached {
			if err := downloader.VerifyFile(patchPath, patch.Hash); err != nil {
				log.Printf("[patch] Hash mismatch for %s, re-downloading: %v", patch.Name, err)
				os.Remove(patchPath)
				patchPath, cached, err = a.acquirePatch(ctx, manifest.PatchBaseURL, patch)
				if err != nil {
					a.handleError("ERR_DOWNLOAD", fmt.Sprintf("Re-download %s failed: %v", patch.Name, err), false)
					return totalApplied, err
				}
				if err := downloader.VerifyFile(patchPath, patch.Hash); err != nil {
					os.Remove(patchPath)
					a.handleError("ERR_HASH", fmt.Sprintf("Hash mismatch for %s after re-download: %v", patch.Name, err), false)
					return totalApplied, err
				}
			}
		}

		a.engine.SetState(engine.StatePatching)
		a.setProgress(fmt.Sprintf("Applying %s...", patch.Name), patch.Name, 100, 0, "")

		if err := a.applyPatch(patch, patchPath, len(queue), totalApplied+1); err != nil {
			a.handleError("ERR_PATCH", fmt.Sprintf("Failed to apply %s: %v", patch.Name, err), false)
			return totalApplied, err
		}

		a.cachePatch(patch, patchPath, cached)

		localState.AppliedPatches = append(localState.AppliedPatches, engine.LocalPatch{
			ID:   patch.ID,
			Name: patch.Name,
			Hash: patch.Hash,
			Size: patch.Size,
		})
		if err := engine.WriteLocalState(a.gamePath, localState); err != nil {
			a.handleError("ERR_LOCAL_STATE", fmt.Sprintf("Failed to write state: %v", err), false)
			return totalApplied, err
		}

		totalApplied++
		log.Printf("[patch] Applied %s (id=%d)", patch.Name, patch.ID)
	}
	return totalApplied, nil
}

func cacheDir(gamePath string) string {
	return filepath.Join(gamePath, ".goro-patches")
}

func cachePathFor(gamePath string, patch engine.Patch) string {
	tag := patch.Hash
	if len(tag) > 12 {
		tag = tag[:12]
	}
	return filepath.Join(cacheDir(gamePath), patch.Name+"."+tag+".patch")
}

func (a *App) acquirePatch(ctx context.Context, patchBaseURL string, patch engine.Patch) (string, bool, error) {
	cachePath := cachePathFor(a.gamePath, patch)
	if _, err := os.Stat(cachePath); err == nil && downloader.VerifyFile(cachePath, patch.Hash) == nil {
		return cachePath, true, nil
	}

	if patchBaseURL == "" {
		return "", false, fmt.Errorf("no patch_base_url in manifest")
	}

	safeName, err := engine.SafePatchComponent(patch.Name)
	if err != nil {
		return "", false, err
	}

	downloadPath := filepath.Join(a.gamePath, safeName)
	patchURL := patchBaseURL + patch.Name
	if err := a.dl.Fetch(ctx, patchURL, downloadPath, nil); err != nil {
		return "", false, fmt.Errorf("download: %w", err)
	}

	return downloadPath, false, nil
}

func (a *App) cachePatch(patch engine.Patch, path string, fromCache bool) {
	if fromCache {
		return
	}
	os.MkdirAll(cacheDir(a.gamePath), 0755)
	os.Rename(path, cachePathFor(a.gamePath, patch))
}

func (a *App) prunePatchCache(manifest *engine.Manifest) {
	want := make(map[string]struct{}, len(manifest.Patches))
	for _, p := range manifest.Patches {
		want[cachePathFor(a.gamePath, p)] = struct{}{}
	}
	matches, _ := filepath.Glob(filepath.Join(cacheDir(a.gamePath), "*.patch"))
	for _, m := range matches {
		if _, ok := want[m]; !ok {
			os.Remove(m)
		}
	}
}

func (a *App) applyPatch(patch engine.Patch, patchPath string, totalPatches int, currentPatch int) error {
	switch patch.Type {
	case "grf":
		target, err := engine.SafePatchComponent(patch.Target)
		if err != nil {
			return err
		}
		grfPath := filepath.Join(a.gamePath, target)

		var lastUpdate time.Time
		grfProgress := func(current, total int, filename string) {
			now := time.Now()
			if now.Sub(lastUpdate) < 200*time.Millisecond && current != 1 && current != total {
				return
			}
			lastUpdate = now

			filePct := float64(current) / float64(total) * 100
			totalPct := (float64(currentPatch-1) + float64(current)/float64(total)) / float64(totalPatches) * 100

			a.setProgress(
				fmt.Sprintf("Applying %s: %d/%d files", patch.Target, current, total),
				filename,
				filePct,
				totalPct,
				fmt.Sprintf("%d/%d", current, total),
			)
		}

		if err := grf.PatchGRFWithProgress(grfPath, patchPath, grfProgress); err != nil {
			return err
		}

		merged, err := grf.Open(grfPath)
		if err != nil {
			return fmt.Errorf("merged GRF is corrupt: %w", err)
		}
		defer merged.Close()

		for entry, h := range patch.FileHashes {
			data, err := merged.ReadFile(entry)
			if err != nil {
				return fmt.Errorf("entry %s missing after apply: %w", entry, err)
			}
			if downloader.HashBytes(data) != h {
				return fmt.Errorf("entry %s content mismatch after apply", entry)
			}
		}
		return nil
	case "raw":
		if err := engine.PatchRaw(a.gamePath, patchPath); err != nil {
			return err
		}
		for f, h := range patch.FileHashes {
			s := engine.SanitizePath(f)
			if s == "" || s != f {
				continue
			}
			if err := downloader.VerifyFile(filepath.Join(a.gamePath, s), h); err != nil {
				return fmt.Errorf("installed file %s verification failed: %w", s, err)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown patch type: %s", patch.Type)
	}
}

func (a *App) LaunchGame() error {
	running, _ := detector.IsGameRunning(a.config.ExeName)
	if running {
		return fmt.Errorf("game is already running")
	}

	exePath := filepath.Join(a.gamePath, a.config.ExeName)
	return engine.LaunchGame(exePath, nil)
}

func (a *App) GetLocalVersion() (int, error) {
	state, err := engine.ReadLocalState(a.gamePath)
	if err != nil {
		return 0, err
	}
	return engine.LocalVersion(state), nil
}

func (a *App) GetManifestURL() string {
	return a.config.ManifestURL
}

func (a *App) SetManifestURL(url string) error {
	a.config.ManifestURL = url
	return engine.SaveConfig(a.gamePath, a.config)
}

func (a *App) GetExeName() string {
	return a.config.ExeName
}

func (a *App) SetExeName(name string) error {
	a.config.ExeName = name
	return engine.SaveConfig(a.gamePath, a.config)
}

func (a *App) IsGameRunning() (bool, error) {
	return detector.IsGameRunning(a.config.ExeName)
}

func (a *App) GetNotes() ([]engine.RenderedNote, string, error) {
	if a.config.NotesURL == "" {
		return nil, "", nil
	}

	data, err := a.dl.FetchBytes(a.ctx, a.config.NotesURL)
	if err != nil {
		log.Printf("[notes] Failed to fetch notes config: %v", err)
		return nil, "", nil
	}

	manifest, err := engine.ParseNotesManifest(data)
	if err != nil {
		log.Printf("[notes] Failed to parse notes config: %v", err)
		return nil, "", nil
	}

	var notes []engine.RenderedNote
	for _, entry := range manifest.Notes {
		noteURL := manifest.NotesBaseURL + entry.Name
		mdData, err := a.dl.FetchBytes(a.ctx, noteURL)
		if err != nil {
			log.Printf("[notes] Failed to fetch note %s: %v", entry.Name, err)
			continue
		}

		notes = append(notes, engine.RenderedNote{
			ID:      entry.ID,
			Name:    entry.Name,
			Content: engine.RenderMarkdown(mdData),
		})
	}

	engine.SortNotesByIDDesc(notes)

	var css string
	if manifest.NotesCSSURL != "" {
		cssData, err := a.dl.FetchBytes(a.ctx, manifest.NotesCSSURL)
		if err != nil {
			log.Printf("[notes] Failed to fetch notes CSS: %v", err)
		} else {
			css = string(cssData)
		}
	}

	return notes, css, nil
}

func (a *App) CancelDownload() error {
	if a.cancelFn != nil {
		a.cancelFn()
	}
	return nil
}

func (a *App) FullGRFCheck() (engine.GRFCheckReport, error) {
	state, err := engine.ReadLocalState(a.gamePath)
	if err != nil {
		return engine.GRFCheckReport{}, err
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()
	manifest, err := a.fetchManifest(ctx)
	if err != nil {
		return engine.GRFCheckReport{}, err
	}

	report := engine.GRFCheckReport{Full: true}
	report.OK = true
	for _, target := range engine.GRFTargets(state, manifest) {
		res := engine.CheckGRFIntegrity(a.gamePath, target)
		report.GRFs = append(report.GRFs, res)
		if !res.OK {
			report.OK = false
		}
	}
	return report, nil
}

func (a *App) handleError(code, message string, fatal bool) {
	a.engine.SetState(engine.StateError)
	a.setProgress("Error: "+message, "", 0, 0, "")
	log.Printf("[error] %s: %s (fatal=%t)", code, message, fatal)
}

func (a *App) setComplete(newVersion int) {
	a.setProgress(fmt.Sprintf("Ready (patch %d)", newVersion), "", 100, 100, "")
}

func (a *App) setAlreadyUpToDate(version int) {
	a.setProgress(fmt.Sprintf("Up to date (patch %d)", version), "", 100, 100, "")
}
