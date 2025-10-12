'use strict';

window.api = async function api(path, options = {}) {
    const method = (options.method || 'GET').toUpperCase();
    const headers = {
        'Accept': 'application/json',
        ...(options.headers || {})
    };
    const res = await fetch(path, {
        ...options,
        method,
        headers
    });

    const raw = await res.text();
    let data;
    try {
        data = raw ? JSON.parse(raw) : null;
    } catch {
        // If server returned non-JSON but status is not OK, surface raw text
        if (!res.ok) throw new Error(raw);
        // For OK responses with non-JSON, just return the raw text
        return raw;
    }

    if (!res.ok) {
        const msg = (data && (data.detail || data.error)) || raw || `HTTP ${res.status}`;
        throw new Error(msg);
    }
    return data;
};

window.prettyJSON = function prettyJSON(data) {
    try {
        if (typeof data === 'string') data = JSON.parse(data);
        return JSON.stringify(data, null, 2);
    } catch {
        return data;
    }
};
