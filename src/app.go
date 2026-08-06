package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	a.engine = engine.New(a)
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

	version, _ := engine.ReadLocalVersion(a.gamePath)
	if version > 0 {
		a.progressInfo = ProgressInfo{
			Status:       fmt.Sprintf("Ready (patch %d)", version),
			TotalPercent: 100,
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

func (a *App) EmitProgress(evt engine.ProgressEvent) {}
func (a *App) EmitError(evt engine.ErrorEvent)       {}

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

	a.engine.EmitProgress(engine.ProgressEvent{
		Status:       status,
		CurrentFile:  currentFile,
		FilePercent:  filePct,
		TotalPercent: totalPct,
		Speed:        speed,
	})
}

func (a *App) StartCheck() error {
	state := a.engine.State()
	if state != engine.StateIdle && state != engine.StateReady && state != engine.StateError {
		return fmt.Errorf("cannot start check in state %s", state)
	}

	go a.runCheckCycle()
	return nil
}

func (a *App) runCheckCycle() {
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelFn = cancel
	defer cancel()

	a.engine.SetState(engine.StateChecking)
	a.setProgress("Checking for updates...", "", 0, 0, "")

	// 1. Fetch manifest
	manifest, err := a.fetchManifest(ctx)
	if err != nil {
		a.handleError("ERR_MANIFEST", err.Error(), true)
		return
	}

	// 2. Self-update check
	if a.updater != nil && engine.NeedsSelfUpdate(manifest) {
		a.setProgress("Checking patcher update...", "", 0, 0, "")
		updated, err := a.updater.Update(ctx, manifest.PatcherURL, manifest.PatcherHash)
		if err != nil {
			a.engine.EmitError(engine.ErrorEvent{
				Code:    "self_update_error",
				Message: fmt.Sprintf("Self-update failed: %v", err),
			})
		} else if updated {
			if ok, _ := a.updater.Restart(); ok {
				os.Exit(0)
			}
			return
		}
	}

	// 3. Check game not running
	running, _ := detector.IsGameRunning(a.config.ExeName)
	if running {
		a.handleError("ERR_GAME_RUNNING", "Game is running. Please close it first.", false)
		return
	}

	// 4. Apply patches one at a time
	totalApplied := 0

	for {
		// Get next patch
		queue, _, err := a.evaluatePatchQueue(manifest)
		if err != nil {
			a.handleError("ERR_LOCAL_STATE", err.Error(), false)
			return
		}

		if len(queue) == 0 {
			break
		}

		patch := queue[0]

		// Download
		a.engine.SetState(engine.StateDownloading)
		a.setProgress(fmt.Sprintf("Downloading %s...", patch.Name), patch.Name, 0, 0, "")

		patchPath, err := a.acquirePatch(ctx, manifest.PatchBaseURL, patch)
		if err != nil {
			a.handleError("ERR_DOWNLOAD", fmt.Sprintf("Failed to get %s: %v", patch.Name, err), false)
			return
		}

		// Verify — if hash fails, delete and re-download once
		if err := downloader.VerifyFile(patchPath, patch.Hash); err != nil {
			log.Printf("[patch] Hash mismatch for %s, deleting and re-downloading: %v", patch.Name, err)
			os.Remove(patchPath)
			patchPath, err = a.acquirePatch(ctx, manifest.PatchBaseURL, patch)
			if err != nil {
				a.handleError("ERR_DOWNLOAD", fmt.Sprintf("Re-download %s failed: %v", patch.Name, err), false)
				return
			}
			if err := downloader.VerifyFile(patchPath, patch.Hash); err != nil {
				os.Remove(patchPath)
				a.handleError("ERR_HASH", fmt.Sprintf("Hash mismatch for %s after re-download: %v", patch.Name, err), false)
				return
			}
		}

		// Apply
		a.engine.SetState(engine.StatePatching)
		a.setProgress(fmt.Sprintf("Applying %s...", patch.Name), patch.Name, 100, 0, "")

		if err := a.applyPatch(patch, patchPath, 1, 1); err != nil {
			a.handleError("ERR_PATCH", fmt.Sprintf("Failed to apply %s: %v", patch.Name, err), false)
			return
		}

		// Cleanup downloaded file
		os.Remove(patchPath)

		// Write version
		if err := engine.WriteLocalVersion(a.gamePath, patch.ID); err != nil {
			a.handleError("ERR_LOCAL_STATE", fmt.Sprintf("Failed to write version: %v", err), false)
			return
		}

		totalApplied++
		log.Printf("[patch] Applied %s (id=%d)", patch.Name, patch.ID)
	}

	engine.CleanupBackups(a.gamePath)

	finalVersion, _ := engine.ReadLocalVersion(a.gamePath)
	a.engine.SetState(engine.StateReady)

	if totalApplied == 0 {
		log.Printf("[runCheckCycle] Already up to date, version=%d", finalVersion)
		a.setAlreadyUpToDate(finalVersion)
	} else {
		log.Printf("[runCheckCycle] Complete, applied %d patches, version=%d", totalApplied, finalVersion)
		a.setComplete(totalApplied, finalVersion)
	}
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

	return manifest, nil
}

func (a *App) evaluatePatchQueue(manifest *engine.Manifest) ([]engine.Patch, int, error) {
	localVersion, err := engine.ReadLocalVersion(a.gamePath)
	if err != nil {
		return nil, 0, fmt.Errorf("read local version: %w", err)
	}

	queue := engine.BuildQueue(manifest, localVersion)
	return queue, localVersion, nil
}

func (a *App) acquirePatch(ctx context.Context, patchBaseURL string, patch engine.Patch) (string, error) {
	if patchBaseURL == "" {
		return "", fmt.Errorf("no patch_base_url in manifest")
	}

	downloadPath := filepath.Join(a.gamePath, patch.Name)
	patchURL := patchBaseURL + patch.Name
	if err := a.dl.Fetch(ctx, patchURL, downloadPath, nil); err != nil {
		return "", fmt.Errorf("download: %w", err)
	}

	return downloadPath, nil
}

func (a *App) applyPatch(patch engine.Patch, patchPath string, totalPatches int, currentPatch int) error {
	switch patch.Type {
	case "grf":
		grfPath := filepath.Join(a.gamePath, patch.Target)

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
		merged.Close()
		return nil
	case "raw":
		return engine.PatchRaw(a.gamePath, patchPath)
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
	return engine.ReadLocalVersion(a.gamePath)
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

func (a *App) CancelDownload() error {
	if a.cancelFn != nil {
		a.cancelFn()
	}
	return nil
}

func (a *App) handleError(code, message string, fatal bool) {
	a.engine.SetState(engine.StateError)
	a.setProgress("Error: "+message, "", 0, 0, "")
	a.engine.EmitError(engine.ErrorEvent{
		Code:    code,
		Message: message,
		Fatal:   fatal,
	})
}

func (a *App) setComplete(patchCount, newVersion int) {
	a.setProgress(fmt.Sprintf("Ready (patch %d)", newVersion), "", 100, 100, "")
}

func (a *App) setAlreadyUpToDate(version int) {
	a.setProgress(fmt.Sprintf("Up to date (patch %d)", version), "", 100, 100, "")
}
