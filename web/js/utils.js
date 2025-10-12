'use strict';

// Minimal HTML escape to avoid XSS when inserting dynamic content.
function escapeHTML(str = '') {
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

// Helper: clear fetched messages and send-message input/status
window.prettyJSON = function prettyJSON(data) {
    try {
        if (typeof data === 'string') data = JSON.parse(data);
        return JSON.stringify(data, null, 2);
    } catch {
        return data;
    }
};
