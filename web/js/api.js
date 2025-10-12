'use strict';

window.api = async function api(path, options = {}) {
    const method = (options.method || 'GET').toUpperCase();
    const res = await fetch(path, {
        headers: { 'Accept': 'application/json', ...(options.headers || {}) },
        ...options
    });
    const text = await res.text();
    let data;
    try { data = JSON.parse(text); } catch { throw new Error(text); }
    if (!res.ok) throw new Error(data.detail || data.error || text);
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
