'use strict';

// HTTP helper (JSON if possible)
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
        if (!res.ok) throw new Error(raw);
        return raw;
    }

    if (!res.ok) {
        const msg = (data && (data.detail || data.error)) || raw || `HTTP ${res.status}`;
        const err = new Error(msg);
        err.status = res.status;
        throw err;
    }
    return data;
};
