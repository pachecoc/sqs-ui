'use strict';

// Countdown timer for refresh
let sendTimer = null;

// Fetch messages
window.fetchMessages = async function fetchMessages() {
  if (window.state && window.state.pending.fetchMessages) return;
  const msgOut = document.getElementById('msgOut');
  if (!msgOut) return;
  if (window.state) window.state.pending.fetchMessages = true;
  msgOut.textContent = 'Fetching messages...';
  try {
    const data = await api('/api/messages');
    renderMessages(data);
  } catch (err) {
    renderError(msgOut, 'Failed to fetch messages:', err.message, 'Check queue settings and credentials provided.');
  } finally {
    if (window.state) window.state.pending.fetchMessages = false;
  }
};

// Send a message
window.sendMessage = async function sendMessage() {
  if (window.state && window.state.pending.sendMessage) return;
  const msgBox = document.getElementById('msgInput');
  const sendStatus = document.getElementById('sendStatus');
  if (!msgBox || !sendStatus) return;

  const msg = msgBox.value.trim();

  if (!msg) {
    sendStatus.innerHTML = '<p class="text-red-600">Please enter a message before sending.</p>';
    return;
  }

  if (window.state) window.state.pending.sendMessage = true;
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
        refreshInfoAndMessages().finally(() => {
          countdownEl.textContent = 'Queue refreshed successfully.';
          setTimeout(() => countdownEl.remove(), 1500);
        });
      }
    }, 1000);
  } catch (err) {
    sendStatus.innerHTML = `<p class="text-red-600 font-semibold">Error sending message: ${err.message}</p>`;
  } finally {
    if (window.state) window.state.pending.sendMessage = false;
  }
};

// Confirm dialog
window.confirmDialog = function confirmDialog(message) {
  return new Promise((resolve) => {
    const dlg = document.getElementById('confirmDialog');
    const txt = document.getElementById('confirmDialogText');
    const ok = document.getElementById('confirmDialogOk');
    const cancel = document.getElementById('confirmDialogCancel');

    if (!dlg || !ok || !cancel || !txt) {
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

    const onKey = (e) => {
      if (e.key === 'Escape') { cleanup(false); }
    };
    document.addEventListener('keydown', onKey, { once: true });
  });
};

// Purge queue
window.purgeQueue = async function purgeQueue() {
  const msgOut = document.getElementById('msgOut');
  if (!msgOut) return;

  const confirmed = await window.confirmDialog('This will delete all messages from the queue. Continue?');
  if (!confirmed) return;

  try {
    msgOut.textContent = 'Purging queue...';
    await api('/api/purge', { method: 'POST' });

    msgOut.innerHTML = `
      <p class="text-green-600 font-semibold mb-1">Queue purged.</p>
      <p class="text-gray-600">Refreshing…</p>`;

    await new Promise((res) => setTimeout(res, 2000));

    window.clearMessageUI({ clearAll: true });
    await fetchInfo();

  } catch (err) {
    renderError(msgOut, 'Failed to purge queue', err.message, 'Check queue settings and server logs.');
  }
};
