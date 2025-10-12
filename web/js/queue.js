'use strict';

window.openQueueDialog = function openQueueDialog() {
    document.getElementById('queueDialog').classList.remove('hidden');
    document.getElementById('queueDialog').classList.add('flex');
};

window.closeQueueDialog = function closeQueueDialog() {
    const dialog = document.getElementById('queueDialog');
    dialog.classList.add('hidden');
    dialog.classList.remove('flex');
    document.getElementById('queueNameInput').value = '';
    document.getElementById('queueUrlInput').value = '';
    document.getElementById('queueStatus').textContent = '';
};

window.applyQueueChange = async function applyQueueChange() {
    const name = document.getElementById('queueNameInput').value.trim();
    const url = document.getElementById('queueUrlInput').value.trim();
    const statusEl = document.getElementById('queueStatus');
    const btn = document.getElementById('queueApplyBtn');
    const spinner = document.getElementById('queueSpinner');
    const text = document.getElementById('queueApplyText');

    if (!name && !url) {
        statusEl.textContent = 'Please enter a queue name or URL.';
        statusEl.className = 'text-sm text-red-600 mb-3';
        return;
    }

    btn.disabled = true; spinner.classList.remove('hidden'); text.textContent = 'Applying...';
    try {
        await api('/api/config/queue', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ queue_name: name, queue_url: url })
        });
        closeQueueDialog();

        window.clearMessageBoxes({ clearAll: true });

        fetchInfo();
    } catch (err) {
        statusEl.textContent = `Failed to update queue: ${err.message}`;
        statusEl.className = 'text-sm text-red-600 mb-3';
    } finally {
        btn.disabled = false; spinner.classList.add('hidden'); text.textContent = 'Apply';
    }
};
