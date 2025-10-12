'use strict';

window.renderQueueInfo = function renderQueueInfo(info) {
  const infoOut = document.getElementById('infoOut');
  if (!info) return;

  // Define aligned key-value pairs
  const lines = [
    { label: 'Current Region', value: info.current_region || '-' },
    { label: 'Queue Name', value: info.queue_name || '-' },
    { label: 'Queue URL', value: info.queue_url || '-' },
    { label: 'Total Messages', value: info.number_of_messages ?? '-' },
    { label: 'Status', value: info.status || '-' },
  ];

  // Build formatted text with padding (aligned at colon)
  const formatted = lines
    .map(({ label, value }) => label.padEnd(16, ' ') + ': ' + value)
    .join('\n')
    .trimEnd(); // remove any trailing newline/space to avoid an extra blank line

  // Render block with reduced padding for a tighter box
  infoOut.innerHTML = `<pre class="bg-gray-800 text-gray-200 rounded p-2 text-left font-mono overflow-auto whitespace-pre leading-snug break-all">${formatted}</pre>`;
};

window.renderMessages = function renderMessages(data) {
  const msgOut = document.getElementById('msgOut');
  if (!Array.isArray(data) || data.length === 0) {
    msgOut.innerHTML = '<p class="text-gray-500 italic">No messages in the queue.</p>';
    return;
  }

  msgOut.innerHTML = `<p class="text-gray-600 mb-2">Fetched ${data.length} message${data.length > 1 ? 's' : ''}:</p><pre class="bg-black text-gray-200 rounded p-3 text-left overflow-auto whitespace-pre-wrap break-words">${prettyJSON(data)}</pre>`;
};

window.renderError = function renderError(target, title, msg, hint) {
  target.innerHTML = `
    <div class="bg-red-50 border border-red-100 rounded-md px-3 py-1 text-left">
      <p class="text-red-600 font-semibold text-sm leading-tight m-0">${title}</p>
      <p class="text-[12px] text-gray-700 leading-tight m-0">${msg}</p>
      <p class="text-[11px] text-gray-500 italic leading-tight m-0">${hint}</p>
    </div>`;
};

// Helper: clear fetched messages and send-message input/status
// Accepts options: { clearInput: true } (default)
window.clearMessageBoxes = function clearMessageBoxes(opts = {}) {
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
