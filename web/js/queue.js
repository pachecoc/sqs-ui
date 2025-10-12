'use strict';

// Open queue config dialog
window.openQueueDialog = function openQueueDialog() {
    const dlg = document.getElementById('queueDialog');
    if (!dlg) return;
    dlg.classList.remove('hidden');
    dlg.classList.add('flex');
    const first = document.getElementById('queueNameInput');
    if (first) first.focus();
};

// Close queue config dialog
window.closeQueueDialog = function closeQueueDialog() {
    const dialog = document.getElementById('queueDialog');
    const nameInput = document.getElementById('queueNameInput');
    const urlInput = document.getElementById('queueUrlInput');
    const statusEl = document.getElementById('queueStatus');

    if (dialog) {
        dialog.classList.add('hidden');
        dialog.classList.remove('flex');
    }
    if (nameInput) nameInput.value = '';
    if (urlInput) urlInput.value = '';
    if (statusEl) {
        statusEl.textContent = '';
        statusEl.className = 'text-sm text-gray-600 mb-3 h-5';
    }
};

// Update queue configuration
window.updateQueueConfig = async function updateQueueConfig() {
    const nameInput = document.getElementById('queueNameInput');
    const urlInput = document.getElementById('queueUrlInput');
    const statusEl = document.getElementById('queueStatus');
    const btn = document.getElementById('queueApplyBtn');
    const spinner = document.getElementById('queueSpinner');
    const text = document.getElementById('queueApplyText');

    if (!nameInput || !urlInput || !statusEl || !btn || !spinner || !text) return;

    const name = nameInput.value.trim();
    const url = urlInput.value.trim();

    if (!name && !url) {
        statusEl.textContent = 'Please enter a queue name or URL.';
        statusEl.className = 'text-sm text-red-600 mb-3';
        return;
    }

    btn.disabled = true;
    spinner.classList.remove('hidden');
    text.textContent = 'Applying...';
    statusEl.textContent = 'Updating queue configuration...';
    statusEl.className = 'text-sm text-gray-600 mb-3';

    try {
        await api('/api/config/queue', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ queue_name: name, queue_url: url })
        });

        closeQueueDialog();
        window.clearMessageUI({ clearAll: true });
        await fetchInfo();
    } catch (err) {
        statusEl.textContent = `Failed to update queue: ${err.message}`;
        statusEl.className = 'text-sm text-red-600 mb-3';
    } finally {
        btn.disabled = false;
        spinner.classList.add('hidden');
        text.textContent = 'Apply';
    }
};
