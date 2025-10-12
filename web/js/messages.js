// ...existing code...
'use strict';

let sendTimer = null;

window.fetchMsgs = async function fetchMsgs() {
  const msgOut = document.getElementById('msgOut');
  msgOut.textContent = 'Fetching messages...';
  try {
    const data = await api('/api/messages');
    renderMessages(data);
  } catch (err) {
    renderError(msgOut, 'Failed to fetch messages:', err.message, 'Check queue settings and credentials provided.');
  }
};

window.sendMsg = async function sendMsg() {
  const msgBox = document.getElementById('msgInput');
  const msg = msgBox.value.trim();
  const sendStatus = document.getElementById('sendStatus');

  if (!msg) {
    sendStatus.innerHTML = '<p class="text-red-600">Please enter a message before sending.</p>';
    return;
  }

  // prevent multiple clicks
  sendStatus.textContent = '';
  const statusP = document.createElement('p');
  statusP.className = 'text-gray-500 italic';
  statusP.textContent = 'Sending message...';
  sendStatus.appendChild(statusP);

  try {
    await api('/api/send', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message: msg })
    });

    msgBox.value = '';
    sendStatus.innerHTML = '<p class="text-green-600 font-semibold mb-1">Message sent successfully.</p>';

    // clear any previous timer
    if (sendTimer) {
      clearInterval(sendTimer);
      sendTimer = null;
    }

    let seconds = 8;
    const countdownEl = document.createElement('p');
    countdownEl.className = 'text-blue-600 text-sm mt-1';
    countdownEl.textContent = `Refreshing in ${seconds}s...`;
    sendStatus.appendChild(countdownEl);

    sendTimer = setInterval(() => {
      seconds--;
      if (seconds > 0) {
        countdownEl.textContent = `Refreshing in ${seconds}s...`;
      } else {
        clearInterval(sendTimer);
        sendTimer = null;
        countdownEl.textContent = 'Refreshing now...';
        // refreshQueueState is expected to call /api/info and update UI
        refreshQueueState().finally(() => {
          countdownEl.textContent = 'Queue refreshed successfully.';
          setTimeout(() => countdownEl.remove(), 1500);
        });
      }
    }, 1000);
  } catch (err) {
    sendStatus.innerHTML = `<p class="text-red-600 font-semibold">Error sending message: ${err.message}</p>`;
  }
};

// small modal-based confirm helper that returns a Promise<boolean>
window.confirmModal = function confirmModal(message) {
  return new Promise((resolve) => {
    const dlg = document.getElementById('confirmDialog');
    const txt = document.getElementById('confirmDialogText');
    const ok = document.getElementById('confirmDialogOk');
    const cancel = document.getElementById('confirmDialogCancel');

    if (!dlg || !ok || !cancel || !txt) {
      // fallback to native confirm if modal is not present
      return resolve(window.confirm(message));
    }

    txt.textContent = message;
    dlg.classList.remove('hidden');
    dlg.classList.add('flex');

    const cleanup = (result) => {
      dlg.classList.add('hidden');
      dlg.classList.remove('flex');
      ok.removeEventListener('click', onOk);
      cancel.removeEventListener('click', onCancel);
      resolve(result);
    };

    const onOk = () => cleanup(true);
    const onCancel = () => cleanup(false);

    ok.addEventListener('click', onOk);
    cancel.addEventListener('click', onCancel);

    // optional: close on Escape
    const onKey = (e) => {
      if (e.key === 'Escape') { cleanup(false); }
    };
    document.addEventListener('keydown', onKey, { once: true });
  });
};

window.purgeQueue = async function purgeQueue() {
  const msgOut = document.getElementById('msgOut');

  const confirmed = await window.confirmModal('This will delete all messages from the queue. Continue?');
  if (!confirmed) return;

  try {
    msgOut.textContent = 'Purging queue...';
    await api('/api/purge', { method: 'POST' });

    // keep success visible for a short moment so user notices it
    msgOut.innerHTML = `
      <p class="text-green-600 font-semibold mb-1">Queue purged.</p>
      <p class="text-gray-600">Refreshing…</p>`;

    // small deliberate pause so the user sees the purged state
    await new Promise((res) => setTimeout(res, 2000));

    // clear message panels and refresh info
    window.clearMessageBoxes({ clearAll: true });
    await fetchInfo();

  } catch (err) {
    renderError(msgOut, 'Failed to purge queue', err.message, 'Check queue settings and server logs.');
  }
};
// ...existing code...
