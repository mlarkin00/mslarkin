/**
 * logging.js
 * Handles client-side event logging by sending data to the /api/log endpoint.
 */

(function() {
    // Buffer for logs to avoid spamming the server?
    // For now, immediate send for simplicity and real-time feel,
    // but with a small debounce/throttle in real prod maybe.

    function sendLog(eventType, details) {
        const payload = {
            timestamp: new Date().toISOString(),
            type: eventType,
            location: window.location.href,
            details: details || {}
        };

        let basePath = (window.APP_CONFIG && window.APP_CONFIG.basePath) ? window.APP_CONFIG.basePath : '';
        // Remove trailing slash if present to avoid double slashes
        if (basePath.endsWith('/')) {
            basePath = basePath.slice(0, -1);
        }

        // Use sendBeacon if available for better reliability on unload,
        // but fetch is fine for general events.
        // using fetch to ensure we can capture errors easily if needed.
        fetch(`${basePath}/api/log`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(payload)
        }).catch(err => {
            console.error("Failed to send log:", err);
        });
    }

    // Expose globally
    window.LogEvent = sendLog;

    // 1. Page Load
    window.addEventListener('load', () => {
        sendLog('PAGE_LOAD', {
            userAgent: navigator.userAgent,
            referrer: document.referrer
        });
    });

    // 2. HTMX Events
    // htmx:trigger - When an element triggers a request
    // htmx:afterRequest - When the request finishes
    document.body.addEventListener('htmx:trigger', (evt) => {
        sendLog('HTMX_TRIGGER', {
            target: evt.target.id || evt.target.tagName,
            triggerName: evt.detail.elt ? (evt.detail.elt.id || evt.detail.elt.tagName) : 'unknown'
        });
    });

    document.body.addEventListener('htmx:afterRequest', (evt) => {
        const detail = evt.detail;
        sendLog('HTMX_RESPONSE', {
            target: evt.target.id || evt.target.tagName,
            method: detail.requestConfig?.method,
            path: detail.path, // or detail.requestConfig.path
            status: detail.xhr?.status,
            success: detail.successful
        });
    });

    // 3. Chat Interaction (if chat uses custom JS instead of HTMX)
    // Assuming chat-widget.js might handle things manually?
    // We can hook into it or just listen for specific events if we knew them.
    // For now, generic click logging on buttons might cover it.

    document.body.addEventListener('click', (evt) => {
        // Log clicks on buttons or links that might not use HTMX
        const target = evt.target.closest('button, a');
        if (target) {
            // Avoid double logging if HTMX handles it?
            // HTMX event is separate. This is "User Interaction".
            sendLog('USER_CLICK', {
                tagName: target.tagName,
                id: target.id,
                text: target.innerText ? target.innerText.substring(0, 50) : '',
                href: target.href || null
            });
        }
    });

    console.log("Logging initialized.");
})();
