interface ProgressEvent {
    status: string;
    currentFile: string;
    filePercent: number;
    totalPercent: number;
    speed: string;
}

interface ErrorEvent {
    code: string;
    message: string;
    fatal: boolean;
}

interface CompleteEvent {
    patchCount: number;
    newVersion: number;
    readyToLaunch: boolean;
}

// Wails v3 auto-generated bindings will be available after running `wails3 dev`
// The import path is: ../bindings/main (based on package name)
let App: any;

async function initBindings() {
    try {
        const bindings = await import('../bindings/main');
        App = bindings.App;
    } catch (e) {
        console.error('Failed to load bindings:', e);
        // Retry after a delay
        setTimeout(initBindings, 1000);
    }
}

const statusEl = document.getElementById('status')!;
const currentFileEl = document.getElementById('current-file')!;
const progressBar = document.getElementById('progress-bar')!;
const percentEl = document.getElementById('percent')!;
const speedEl = document.getElementById('speed')!;
const btnStart = document.getElementById('btn-start') as HTMLButtonElement;
const btnLaunch = document.getElementById('btn-launch') as HTMLButtonElement;
const errorSection = document.getElementById('error-section')!;
const errorMessage = document.getElementById('error-message')!;
const btnRetry = document.getElementById('btn-retry') as HTMLButtonElement;
const btnMinimize = document.getElementById('btn-minimize')!;
const btnClose = document.getElementById('btn-close')!;
const btnSettings = document.getElementById('btn-settings')!;
const settingsSection = document.getElementById('settings-section')!;
const inputManifestURL = document.getElementById('input-manifest-url') as HTMLInputElement;
const inputPatchBaseURL = document.getElementById('input-patch-base-url') as HTMLInputElement;
const inputGamePath = document.getElementById('input-game-path') as HTMLInputElement;
const btnSaveSettings = document.getElementById('btn-settings-save')!;
const btnCancelSettings = document.getElementById('btn-settings-cancel')!;

let isChecking = false;

function onProgress(evt: ProgressEvent): void {
    statusEl.textContent = evt.status;
    currentFileEl.textContent = evt.currentFile || '';
    progressBar.style.width = evt.totalPercent.toFixed(1) + '%';
    percentEl.textContent = evt.totalPercent.toFixed(1) + '%';
    speedEl.textContent = evt.speed || '';
}

function onError(evt: ErrorEvent): void {
    statusEl.textContent = 'Error';
    errorMessage.textContent = evt.message;
    errorSection.classList.remove('hidden');
    btnRetry.classList.toggle('hidden', evt.fatal);
    btnStart.disabled = false;
    btnStart.textContent = 'Check for Updates';
    isChecking = false;
}

function onComplete(_evt: CompleteEvent): void {
    statusEl.textContent = 'Ready to play!';
    currentFileEl.textContent = '';
    progressBar.style.width = '100%';
    percentEl.textContent = '100%';
    speedEl.textContent = '';
    btnStart.disabled = false;
    btnStart.textContent = 'Check for Updates';
    btnLaunch.disabled = false;
    errorSection.classList.add('hidden');
    isChecking = false;
}

function onSelfUpdateReady(): void {
    statusEl.textContent = 'Patcher updated. Restarting...';
}

btnStart.addEventListener('click', async () => {
    if (!App) {
        statusEl.textContent = 'Waiting for backend...';
        return;
    }

    if (isChecking) {
        try {
            await App.CancelDownload();
        } catch (_e) {
            // ignore
        }
        isChecking = false;
        btnStart.disabled = false;
        btnStart.textContent = 'Check for Updates';
        statusEl.textContent = 'Cancelled';
        return;
    }

    isChecking = true;
    btnStart.disabled = false;
    btnStart.textContent = 'Cancel';
    btnLaunch.disabled = true;
    errorSection.classList.add('hidden');
    progressBar.style.width = '0%';

    try {
        await App.StartCheck();
    } catch (e: unknown) {
        onError({ code: 'ERR_START', message: (e as Error).message, fatal: false });
    }
});

btnLaunch.addEventListener('click', async () => {
    if (!App) return;

    try {
        await App.LaunchGame();
    } catch (e: unknown) {
        onError({ code: 'ERR_LAUNCH', message: (e as Error).message, fatal: false });
    }
});

btnRetry.addEventListener('click', () => {
    errorSection.classList.add('hidden');
    btnStart.click();
});

btnMinimize.addEventListener('click', () => {
    // Wails v3 runtime is available at window.runtime
    if ((window as any).runtime?.WindowMinimise) {
        (window as any).runtime.WindowMinimise();
    }
});

btnClose.addEventListener('click', () => {
    if ((window as any).runtime?.Quit) {
        (window as any).runtime.Quit();
    }
});

btnSettings.addEventListener('click', async () => {
    if (!App) return;

    try {
        inputManifestURL.value = await App.GetManifestURL() || '';
        inputPatchBaseURL.value = await App.GetPatchBaseURL() || '';
        inputGamePath.value = await App.GetGamePath() || '';
    } catch (_e) {
        // ignore
    }
    settingsSection.classList.remove('hidden');
});

btnSaveSettings.addEventListener('click', async () => {
    if (!App) return;

    try {
        await App.SetManifestURL(inputManifestURL.value);
        await App.SetPatchBaseURL(inputPatchBaseURL.value);
        await App.SetGamePath(inputGamePath.value);
        settingsSection.classList.add('hidden');
    } catch (e: unknown) {
        onError({ code: 'ERR_SETTINGS', message: 'Failed to save settings: ' + (e as Error).message, fatal: false });
    }
});

btnCancelSettings.addEventListener('click', () => {
    settingsSection.classList.add('hidden');
});

async function init(): Promise<void> {
    await initBindings();

    // Register event listeners using Wails v3 runtime
    const runtime = (window as any).runtime;
    if (runtime?.EventsOn) {
        runtime.EventsOn('patch_progress', onProgress);
        runtime.EventsOn('patch_error', onError);
        runtime.EventsOn('patch_complete', onComplete);
        runtime.EventsOn('self_update_ready', onSelfUpdateReady);
    }

    // Wait for bindings to be ready
    let retries = 0;
    while (!App && retries < 50) {
        await new Promise(resolve => setTimeout(resolve, 100));
        retries++;
    }

    if (!App) {
        statusEl.textContent = 'Error: Failed to load bindings. Run `wails3 dev`';
        return;
    }

    try {
        const running = await App.IsGameRunning();
        if (running) {
            statusEl.textContent = 'Game is running';
            btnStart.disabled = true;
        }
    } catch (_e) {
        // ignore
    }

    try {
        const version = await App.GetLocalVersion();
        if (version > 0) {
            statusEl.textContent = 'Ready (patch ' + version + ')';
            btnLaunch.disabled = false;
        }
    } catch (_e) {
        // ignore
    }
}

init();
