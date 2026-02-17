// WatchDog Application JavaScript

// HTMX Configuration
document.addEventListener('DOMContentLoaded', function() {
    htmx.config.defaultSwapStyle = 'innerHTML';
    htmx.config.defaultSettleDelay = 0;

    // Include CSRF token in all HTMX requests
    document.body.addEventListener('htmx:configRequest', function(event) {
        var csrfMeta = document.querySelector('meta[name="csrf-token"]');
        if (csrfMeta) {
            event.detail.headers['X-CSRF-Token'] = csrfMeta.content;
        }
    });

    document.body.addEventListener('htmx:beforeRequest', function(event) {
        var target = event.detail.elt;
        var indicator = target.querySelector('.htmx-indicator');
        if (indicator) indicator.classList.remove('hidden');
    });

    document.body.addEventListener('htmx:afterRequest', function(event) {
        var elt = event.detail.elt;
        var indicator = elt.querySelector('.htmx-indicator');
        if (indicator) indicator.classList.add('hidden');

        // CSP-safe replacements for hx-on::after-request (eval is blocked by CSP)
        if (event.detail.successful) {
            var id = elt.id;
            if (id === 'channel-form') {
                elt.reset();
                document.getElementById('new-channel-modal').classList.add('hidden');
            } else if (id === 'token-form') {
                elt.reset();
            } else if (id === 'monitor-form') {
                document.getElementById('new-monitor-modal').classList.add('hidden');
            } else if (id === 'admin-user-form') {
                document.getElementById('new-user-modal').classList.add('hidden');
                elt.reset();
            }
        }
    });

    document.body.addEventListener('htmx:responseError', function() {
        showToast('An error occurred. Please try again.', 'error');
    });

    // Initialize Lucide icons
    if (typeof lucide !== 'undefined') {
        lucide.createIcons();
    }

    // Re-init Lucide after HTMX swaps (new elements may have data-lucide attrs)
    document.body.addEventListener('htmx:afterSwap', function() {
        if (typeof lucide !== 'undefined') {
            lucide.createIcons();
        }
    });
});

// Toast Notification System
function showToast(message, type) {
    type = type || 'success';
    var container = document.getElementById('toast-container');
    if (!container) return;

    var toast = document.createElement('div');
    toast.className = 'toast-enter relative flex items-center space-x-3 px-4 py-3 rounded-lg border shadow-lg max-w-sm overflow-hidden';

    var iconName = 'check-circle';
    var colorClasses = 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400';
    var progressColor = 'bg-emerald-400';

    if (type === 'error') {
        iconName = 'x-circle';
        colorClasses = 'bg-red-500/10 border-red-500/20 text-red-400';
        progressColor = 'bg-red-400';
    } else if (type === 'warning') {
        iconName = 'alert-triangle';
        colorClasses = 'bg-yellow-500/10 border-yellow-500/20 text-yellow-400';
        progressColor = 'bg-yellow-400';
    }

    toast.className += ' ' + colorClasses;
    toast.innerHTML =
        '<i data-lucide="' + iconName + '" class="w-4 h-4 shrink-0"></i>' +
        '<span class="text-sm flex-1">' + message + '</span>' +
        '<button onclick="this.parentElement.remove()" class="shrink-0 p-0.5 rounded hover:bg-white/10 transition-smooth">' +
        '<i data-lucide="x" class="w-3 h-3"></i>' +
        '</button>' +
        '<div class="absolute bottom-0 left-0 h-0.5 toast-progress ' + progressColor + '"></div>';

    container.appendChild(toast);

    // Init icons in this toast
    if (typeof lucide !== 'undefined') {
        lucide.createIcons({ nodes: [toast] });
    }

    setTimeout(function() {
        toast.classList.remove('toast-enter');
        toast.classList.add('toast-exit');
        setTimeout(function() { toast.remove(); }, 300);
    }, 4000);
}

// Confirmation Dialog
function confirmAction(message) {
    return confirm(message);
}

// SSE Connection Management
var sseConnection = null;
var sseReconnectAttempts = 0;
var maxReconnectAttempts = 5;
var reconnectDelay = 3000;

function connectSSE() {
    if (sseConnection) {
        sseConnection.close();
    }

    sseConnection = new EventSource('/sse/events');

    sseConnection.onopen = function() {
        sseReconnectAttempts = 0;
    };

    sseConnection.onerror = function() {
        sseConnection.close();
        if (sseReconnectAttempts < maxReconnectAttempts) {
            sseReconnectAttempts++;
            setTimeout(connectSSE, reconnectDelay);
        }
    };

    sseConnection.addEventListener('agent-status', function(event) {
        var data = JSON.parse(event.data);
        updateAgentStatus(data);
    });

    sseConnection.addEventListener('incident-count', function(event) {
        var data = JSON.parse(event.data);
        updateIncidentCount(data.count);
    });
}

function updateAgentStatus(agent) {
    var agentElement = document.getElementById('agent-' + agent.id);
    if (!agentElement) return;

    var statusDot = agentElement.querySelector('.w-2.h-2');
    var statusBadge = agentElement.querySelector('span[class*="text-xs"]');

    if (agent.status === 'online') {
        if (statusDot) statusDot.className = 'w-2 h-2 rounded-full bg-emerald-400 animate-pulse-dot';
        if (statusBadge) {
            statusBadge.className = 'text-xs px-2 py-0.5 rounded-md bg-emerald-500/15 text-emerald-400';
            statusBadge.textContent = 'online';
        }
    } else {
        if (statusDot) statusDot.className = 'w-2 h-2 rounded-full bg-zinc-500';
        if (statusBadge) {
            statusBadge.className = 'text-xs px-2 py-0.5 rounded-md bg-muted text-muted-foreground';
            statusBadge.textContent = 'offline';
        }
    }
}

function updateIncidentCount(count) {
    var incidentBadge = document.querySelector('[href="/incidents"] .bg-destructive\\/20');
    if (count > 0) {
        if (incidentBadge) {
            incidentBadge.textContent = count;
        }
    } else if (incidentBadge) {
        incidentBadge.remove();
    }
}

// Initialize SSE if on a page that needs it
if (document.querySelector('[sse-connect]') || window.location.pathname === '/dashboard') {
    // SSE is handled by HTMX sse extension
}

// Keyboard shortcuts — two-key sequences (g then d/m/i)
(function() {
    var pendingG = false;
    var pendingTimer = null;

    document.addEventListener('keydown', function(event) {
        // Skip when user is typing in inputs/textareas
        var tag = event.target.tagName;
        if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || event.target.isContentEditable) {
            return;
        }

        // Escape closes modals
        if (event.key === 'Escape') {
            var modals = document.querySelectorAll('[id$="-modal"]:not(.hidden)');
            modals.forEach(function(modal) { modal.classList.add('hidden'); });
            return;
        }

        if (event.key === 'g' && !event.metaKey && !event.ctrlKey) {
            pendingG = true;
            clearTimeout(pendingTimer);
            pendingTimer = setTimeout(function() { pendingG = false; }, 500);
            return;
        }

        if (pendingG) {
            pendingG = false;
            clearTimeout(pendingTimer);
            if (event.key === 'd') {
                window.location.href = '/dashboard';
            } else if (event.key === 'm') {
                window.location.href = '/monitors';
            } else if (event.key === 'i') {
                window.location.href = '/incidents';
            }
        }
    });
})();

// Close modal when clicking backdrop
document.addEventListener('click', function(event) {
    if (event.target.classList.contains('fixed') && event.target.classList.contains('inset-0')) {
        event.target.classList.add('hidden');
    }
});

// Format relative time
function formatTimeAgo(timestamp) {
    var date = new Date(timestamp);
    var now = new Date();
    var diff = now - date;

    var seconds = Math.floor(diff / 1000);
    var minutes = Math.floor(seconds / 60);
    var hours = Math.floor(minutes / 60);
    var days = Math.floor(hours / 24);

    if (days > 0) return days + 'd ago';
    if (hours > 0) return hours + 'h ago';
    if (minutes > 0) return minutes + 'm ago';
    return 'just now';
}

// Utility: Copy to clipboard
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(function() {
        showToast('Copied to clipboard');
    }).catch(function() {
        showToast('Failed to copy', 'error');
    });
}

// Monitor detail panel fetcher
function loadMonitorDetail(monitorId) {
    fetch('/api/monitors/' + monitorId + '/heartbeats?period=24h')
        .then(function(res) { return res.json(); })
        .then(function(data) {
            if (data && data.length > 0) {
                var labels = data.map(function(p) { return p.Time || ''; });
                var latencies = data.map(function(p) { return p.LatencyMs || 0; });
                createLatencyChart('detail-latency-chart', labels, latencies);
            }
        })
        .catch(function() {
            // Silently fail — chart area stays empty
        });
}
