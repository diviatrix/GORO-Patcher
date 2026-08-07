import { App } from './bindings/github.com/diviatrix/GORO-Patcher/index.js';

document.addEventListener('dragover', (e) => e.preventDefault());
document.addEventListener('drop', (e) => e.preventDefault());

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
const btnRepairSettings = document.getElementById('btn-repair-settings');

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
            } else if (p.status.includes('Repair needed')) {
                stopPolling();
                onRepairNeeded();
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

async function updateRepairState() {
    try {
        const needs = await App.NeedsRepair();
        if (needs) {
            btnStart.textContent = 'Repair';
            btnStart.classList.add('repair');
        } else {
            btnStart.textContent = 'Check for Updates';
            btnStart.classList.remove('repair');
        }
    } catch (_e) {}
}

function onFinished() {
    isChecking = false;
    btnStart.disabled = false;
    btnLaunch.disabled = false;
    errorSection.classList.add('hidden');
    updateRepairState();
}

function onRepairNeeded() {
    isChecking = false;
    btnStart.disabled = false;
    btnStart.textContent = 'Repair';
    btnStart.classList.add('repair');
    btnLaunch.disabled = true;
    errorSection.classList.add('hidden');
}

function onError(evt) {
    statusEl.textContent = 'Error';
    errorMessage.textContent = evt.message;
    errorSection.classList.remove('hidden');
    btnRetry.classList.toggle('hidden', evt.fatal);
    btnStart.disabled = false;
    isChecking = false;
    updateRepairState();
}

btnStart.addEventListener('click', async () => {
    if (isChecking) {
        try {
            await App.CancelDownload();
        } catch (_e) {}
        stopPolling();
        isChecking = false;
        btnStart.disabled = false;
        statusEl.textContent = 'Cancelled';
        updateRepairState();
        return;
    }

    isChecking = true;
    btnStart.disabled = false;
    btnStart.textContent = 'Cancel';
    btnStart.classList.remove('repair');
    btnLaunch.disabled = true;
    errorSection.classList.add('hidden');

    startPolling();

    try {
        const needsRepair = await App.NeedsRepair();
        if (needsRepair) {
            await App.StartRepair();
        } else {
            await App.StartCheck();
        }
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

btnRepairSettings.addEventListener('click', async () => {
    settingsSection.classList.add('hidden');
    btnStart.click();
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

    await updateRepairState();
    btnStart.click();
    loadNotes();
}

async function loadNotes() {
    try {
        const [notes, css] = await App.GetNotes();
        if (css) {
            document.getElementById('notes-custom-css').textContent = css;
        }
        const container = document.getElementById('notes-content');
        container.innerHTML = '';
        if (notes && notes.length > 0) {
            for (const note of notes) {
                const div = document.createElement('div');
                div.className = 'note-entry';
                div.innerHTML = note.content;
                container.appendChild(div);
            }
        }
    } catch (_e) {}
}

document.getElementById('notes-content').addEventListener('click', (e) => {
    const link = e.target.closest('a');
    if (link && link.href) {
        e.preventDefault();
        window.wails.Browser.OpenURL(link.href);
    }
});

init();
