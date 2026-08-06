import { App } from './bindings/github.com/diviatrix/GORO-Patcher/index.js';

const statusEl = document.getElementById('status');
const currentFileEl = document.getElementById('current-file');
const progressBar = document.getElementById('progress-bar');
const percentEl = document.getElementById('percent');
const speedEl = document.getElementById('speed');
const btnStart = document.getElementById('btn-start');
const btnLaunch = document.getElementById('btn-launch');
const errorSection = document.getElementById('error-section');
const errorMessage = document.getElementById('error-message');
const btnRetry = document.getElementById('btn-retry');
const btnMinimize = document.getElementById('btn-minimize');
const btnClose = document.getElementById('btn-close');
const btnSettings = document.getElementById('btn-settings');
const settingsSection = document.getElementById('settings-section');
const inputManifestURL = document.getElementById('input-manifest-url');
const inputExeName = document.getElementById('input-exe-name');
const btnSaveSettings = document.getElementById('btn-settings-save');
const btnCancelSettings = document.getElementById('btn-settings-cancel');

let isChecking = false;
let pollTimer = null;

function updateUI(status, currentFile, filePercent, totalPercent, speed) {
    statusEl.textContent = status;
    currentFileEl.textContent = currentFile || '';
    progressBar.style.width = totalPercent.toFixed(1) + '%';
    percentEl.textContent = totalPercent.toFixed(1) + '%';
    speedEl.textContent = speed || '';
}

function startPolling() {
    if (pollTimer) return;
    pollTimer = setInterval(async () => {
        try {
            const p = await App.GetProgress();
            updateUI(p.status, p.currentFile, p.filePercent, p.totalPercent, p.speed);

        if (p.status === 'Complete' || p.status.startsWith('Up to date') || p.status.startsWith('Ready')) {
            stopPolling();
            onFinished();
        } else if (p.status.startsWith('Error:')) {
                stopPolling();
                onError({ code: 'ERR_PATCH', message: p.status.slice(7), fatal: false });
            }
        } catch (_e) {}
    }, 100);
}

function stopPolling() {
    if (pollTimer) {
        clearInterval(pollTimer);
        pollTimer = null;
    }
}

function onFinished() {
    btnStart.disabled = false;
    btnStart.textContent = 'Check for Updates';
    btnLaunch.disabled = false;
    errorSection.classList.add('hidden');
    isChecking = false;
}

function onError(evt) {
    statusEl.textContent = 'Error';
    errorMessage.textContent = evt.message;
    errorSection.classList.remove('hidden');
    btnRetry.classList.toggle('hidden', evt.fatal);
    btnStart.disabled = false;
    btnStart.textContent = 'Check for Updates';
    isChecking = false;
}

btnStart.addEventListener('click', async () => {
    if (isChecking) {
        try {
            await App.CancelDownload();
        } catch (_e) {}
        stopPolling();
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

    startPolling();

    try {
        await App.StartCheck();
    } catch (e) {
        stopPolling();
        onError({ code: 'ERR_START', message: e.message, fatal: false });
    }
});

btnLaunch.addEventListener('click', async () => {
    try {
        await App.LaunchGame();
    } catch (e) {
        onError({ code: 'ERR_LAUNCH', message: e.message, fatal: false });
    }
});

btnRetry.addEventListener('click', () => {
    errorSection.classList.add('hidden');
    btnStart.click();
});

btnMinimize.addEventListener('click', () => {
    window.wails.Window.Minimise();
});

btnClose.addEventListener('click', () => {
    window.wails.Application.Quit();
});

btnSettings.addEventListener('click', async () => {
    try {
        inputManifestURL.value = await App.GetManifestURL() || '';
        inputExeName.value = await App.GetExeName() || '';
    } catch (_e) {}
    settingsSection.classList.remove('hidden');
});

btnSaveSettings.addEventListener('click', async () => {
    try {
        await App.SetManifestURL(inputManifestURL.value);
        await App.SetExeName(inputExeName.value);
        settingsSection.classList.add('hidden');
    } catch (e) {
        onError({ code: 'ERR_SETTINGS', message: 'Failed to save settings: ' + e.message, fatal: false });
    }
});

btnCancelSettings.addEventListener('click', () => {
    settingsSection.classList.add('hidden');
});

async function init() {
    try {
        const running = await App.IsGameRunning();
        if (running) {
            statusEl.textContent = 'Game is running';
            btnStart.disabled = true;
            return;
        }
    } catch (_e) {}

    try {
        const p = await App.GetProgress();
        if (p.status) {
            updateUI(p.status, p.currentFile, p.filePercent, p.totalPercent, p.speed);
        }
        if (p.totalPercent >= 100) {
            btnLaunch.disabled = false;
        }
    } catch (_e) {}

    btnStart.click();
}

init();
