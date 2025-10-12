'use strict';

// Render queue info into the designated output area.
window.renderQueueInfo = function renderQueueInfo(info) {
  const infoOut = document.getElementById('infoOut');
  if (!info || !infoOut) return;

  const lines = [
    { label: 'Current Region', value: info.current_region || '-' },
    { label: 'Queue Name', value: info.queue_name || '-' },
    { label: 'Queue URL', value: info.queue_url || '-' },
    { label: 'Total Messages', value: info.number_of_messages ?? '-' },
    { label: 'Status', value: info.status || '-' },
  ];

  const formatted = lines
    .map(({ label, value }) => label.padEnd(16, ' ') + ': ' + value)
    .join('\n')
    .trimEnd();

  infoOut.innerHTML = `<pre class="bg-gray-800 text-gray-200 rounded p-2 text-left font-mono overflow-auto whitespace-pre leading-snug break-all">${escapeHTML(formatted)}</pre>`;
};

// Render messages list
window.renderMessages = function renderMessages(data) {
  const msgOut = document.getElementById('msgOut');
  if (!msgOut) return;

  if (!Array.isArray(data) || data.length === 0) {
    msgOut.innerHTML = '<p class="text-gray-500 italic">No messages in the queue.</p>';
    return;
  }

  let json;
  try {
    json = JSON.stringify(data, null, 2);
  } catch {
    json = String(data);
  }

  msgOut.innerHTML =
    `<p class="text-gray-600 mb-2 text-left">Fetched ${data.length} message${data.length > 1 ? 's' : ''}</p>` +
    `<pre class="bg-gray-800 text-gray-200 rounded p-3 text-left overflow-auto whitespace-pre-wrap break-words text-sm leading-snug">${escapeHTML(json)}</pre>`;
};

// Clear message UI
window.clearMessageUI = function clearMessageUI(opts = {}) {
  const { clearFetchMessage = false, clearSendMessage = false, clearAll = false } = opts;

  if (clearFetchMessage || clearAll) {
    const msgOut = document.getElementById('msgOut');
    if (msgOut) {
      msgOut.innerHTML = '<p class="text-gray-500 italic">No messages fetched yet...</p>';
    }
  }

  if (clearSendMessage || clearAll) {
    const sendStatus = document.getElementById('sendStatus');
    if (sendStatus) {
      sendStatus.textContent = '';
    }
  }

  if (clearSendMessage || clearAll) {
    const msgInput = document.getElementById('msgInput');
    if (msgInput) {
      msgInput.value = '';
    }
  }
};

// Render error panel
window.renderError = function renderError(target, title, msg, hint) {
  if (!target) return;
  target.innerHTML = `
    <div class="bg-red-50 border border-red-100 rounded-md px-3 py-1 text-left">
      <p class="text-red-600 font-semibold text-sm leading-tight m-0">${escapeHTML(title)}</p>
      <p class="text-[12px] text-gray-700 leading-tight m-0">${escapeHTML(msg)}</p>
      <p class="text-[11px] text-gray-500 italic leading-tight m-0">${escapeHTML(hint)}</p>
    </div>`;
};
