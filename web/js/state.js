'use strict';

// Central application state
window.state = {
    lastQueueInfo: null,
    pending: {
        fetchInfo: false,
        fetchMessages: false,
        sendMessage: false,
        updateQueue: false
    },
    ui: {
        lastFocusedBeforeDialog: null
    }
};

// Global namespace
window.SQSUI = window.SQSUI || {};
window.SQSUI.state = window.state;
