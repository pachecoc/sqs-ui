'use strict';

// Fetch queue info
window.fetchInfo = async function fetchInfo() {
    if (window.state && window.state.pending.fetchInfo) return;
    const infoOut = document.getElementById('infoOut');
    const msgOut = document.getElementById('msgOut');

    if (infoOut) infoOut.innerHTML = '<p>Fetching queue info...</p>';
    if (window.state) window.state.pending.fetchInfo = true;
    try {
        const info = await api('/info');
        if (window.state) window.state.lastQueueInfo = info || null;

        if (info) {
            window.renderQueueInfo(info);
        }

        if (info && info.status === 'not_connected') {
            const title = 'Queue not connected';
            const msg = info.error || info.message || 'The configured queue is not connected.';
            if (msgOut) window.renderError(msgOut, title, msg, 'Check queue settings and credentials provided.');
            window.clearMessageUI({ clearSendMessage: true });
        }

        return info;
    } catch (err) {
        const title = 'Failed to fetch queue info';
        const msg = err && err.message ? err.message : String(err);
        if (msgOut) window.renderError(msgOut, title, msg, 'Check queue settings and credentials provided.');
        return null;
    } finally {
        if (window.state) window.state.pending.fetchInfo = false;
    }
};

// Refresh info and messages
window.refreshInfoAndMessages = async function refreshInfoAndMessages() {
    await Promise.all([fetchInfo(), fetchMessages()]);
};

// Build app skeleton
function renderAppSkeleton() {
    const root = document.getElementById('app-root');
    if (!root) return;
    root.innerHTML = `
    <div class="flex justify-center gap-3 mb-4">
      <button id="changeQueueBtn" type="button" class="bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded shadow">
        Change Queue
      </button>
      <button id="fetchInfoBtn" type="button" class="bg-gray-500 hover:bg-gray-600 text-white px-4 py-2 rounded shadow">
        Fetch Queue Info
      </button>
    </div>

    <div class="bg-gray-50 border border-gray-200 rounded-lg p-4 text-left text-sm font-mono mb-6">
      <h2 class="text-lg font-semibold mb-2">Queue Info</h2>
      <div id="infoOut" class="whitespace-pre-wrap break-words text-sm text-gray-700">
        <p class="text-gray-400 italic">Click “Fetch Queue Info” to view queue details...</p>
      </div>
    </div>

    <div class="flex flex-wrap justify-center gap-3 mb-6">
      <button id="fetchMessagesBtn" type="button" class="bg-green-500 hover:bg-green-600 text-white px-4 py-2 rounded shadow">
        Fetch Messages
      </button>
      <button id="purgeQueueBtn" type="button" class="bg-red-500 hover:bg-red-600 text-white px-4 py-2 rounded shadow">
        Purge Queue
      </button>
    </div>

    <div id="confirmDialog" class="fixed inset-0 hidden items-center justify-center bg-black bg-opacity-50 z-50">
        <div class="bg-white rounded-lg shadow-lg p-5 w-[32rem] max-w-full text-left">
        <h3 class="text-lg font-semibold mb-2">Confirm</h3>
        <p id="confirmDialogText" class="text-sm text-gray-700 mb-4">Are you sure?</p>
        <div class="flex justify-end gap-2">
            <button id="confirmDialogCancel" type="button" class="px-3 py-1 rounded border border-gray-300 hover:bg-gray-100">Cancel</button>
            <button id="confirmDialogOk" type="button" class="bg-red-500 hover:bg-red-600 text-white px-4 py-1 rounded">Yes</button>
        </div>
        </div>
    </div>

    <div class="bg-gray-50 border border-gray-200 rounded-lg p-4 text-left text-sm font-mono h-96 overflow-auto mb-6">
      <h2 class="text-lg font-semibold mb-2">Messages</h2>
      <div id="msgOut" class="whitespace-pre-wrap break-words text-sm text-gray-700">
        <p class="text-gray-400 italic">No messages fetched yet...</p>
      </div>
    </div>

    <div class="bg-gray-50 border border-gray-200 rounded-lg p-4 text-left text-sm font-mono mb-6">
      <h2 class="text-lg font-semibold mb-2">Send a Message</h2>
      <textarea id="msgInput" rows="4" class="w-full border border-gray-300 rounded-md p-2 focus:ring-blue-500 focus:border-blue-500 mb-2 resize-y font-mono text-sm bg-white text-gray-700"
        placeholder="Type your message here..."></textarea>
      <div id="sendStatus" class="text-sm mb-2 min-h-[1.5rem] text-gray-700">:)</div>
    </div>

    <div class="flex justify-end">
      <button id="sendMessageBtn" type="button" class="bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded shadow">
        Send Message
      </button>
    </div>

    <div id="queueDialog" class="fixed inset-0 hidden items-center justify-center bg-black bg-opacity-50 z-50">
      <div class="bg-white rounded-lg shadow-lg p-6 w-[40rem] max-w-full text-left">
        <h2 class="text-xl font-semibold mb-4">Change Queue</h2>
        <label class="block text-sm font-medium text-gray-700 mb-1">Queue Name (ignored if URL is set)</label>
        <input id="queueNameInput" type="text" placeholder="example-queue"
          class="w-full border border-gray-300 rounded-md p-2 mb-3 focus:ring-blue-500 focus:border-blue-500" />
        <label class="block text-sm font-medium text-gray-700 mb-1">Queue URL (required only for cross-account/region queue)</label>
        <input id="queueUrlInput" type="text" placeholder="https://sqs.us-east-1.amazonaws.com/123456789012/example-queue"
          class="w-full border border-gray-300 rounded-md p-2 mb-4 font-mono text-sm focus:ring-blue-500 focus:border-blue-500" />
        <div id="queueStatus" class="text-sm text-gray-600 mb-3 h-5"></div>
        <div class="flex justify-end gap-2">
          <button id="queueCancelBtn" type="button" class="px-3 py-1 rounded border border-gray-300 hover:bg-gray-100">Cancel</button>
          <button id="queueApplyBtn" type="button"
            class="bg-blue-500 hover:bg-blue-600 text-white px-4 py-1 rounded flex items-center gap-2 disabled:opacity-60 disabled:cursor-not-allowed">
            <span id="queueApplyText">Apply</span>
            <svg id="queueSpinner" class="hidden animate-spin h-4 w-4 text-white"
              xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor"
                d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"></path>
            </svg>
          </button>
        </div>
      </div>
    </div>
  `;
}

// Wire event handlers
function wireEvents() {
    const byId = (id) => document.getElementById(id);

    byId('changeQueueBtn')?.addEventListener('click', openQueueDialog);
    byId('fetchInfoBtn')?.addEventListener('click', fetchInfo);
    byId('fetchMessagesBtn')?.addEventListener('click', fetchMessages);
    byId('purgeQueueBtn')?.addEventListener('click', purgeQueue);
    byId('sendMessageBtn')?.addEventListener('click', sendMessage);
    byId('queueCancelBtn')?.addEventListener('click', closeQueueDialog);
    byId('queueApplyBtn')?.addEventListener('click', updateQueueConfig);
}

// Initialize app
window.addEventListener('DOMContentLoaded', async () => {
    renderAppSkeleton();
    wireEvents();
    await fetchInfo();
});
