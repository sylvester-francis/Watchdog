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

    // HTMX progress bar
    document.body.addEventListener('htmx:beforeRequest', function(event) {
        var bar = document.getElementById('htmx-progress');
        if (bar) {
            bar.classList.remove('done');
            bar.classList.add('active');
        }

        var target = event.detail.elt;
        var indicator = target.querySelector('.htmx-indicator');
        if (indicator) indicator.classList.remove('hidden');

        // Button loading state: disable and show spinner
        if (target.tagName === 'BUTTON' || target.tagName === 'FORM') {
            var btn = target.tagName === 'FORM' ? target.querySelector('button[type="submit"]') : target;
            if (btn && !btn.dataset.htmxLoading) {
                btn.dataset.htmxLoading = 'true';
                btn.disabled = true;
                btn.style.opacity = '0.7';
            }
        }
    });

    document.body.addEventListener('htmx:afterRequest', function(event) {
        var bar = document.getElementById('htmx-progress');
        if (bar) {
            bar.classList.remove('active');
            bar.classList.add('done');
            setTimeout(function() { bar.classList.remove('done'); }, 300);
        }

        var elt = event.detail.elt;
        var indicator = elt.querySelector('.htmx-indicator');
        if (indicator) indicator.classList.add('hidden');

        // Restore button loading state
        var btn = elt.tagName === 'FORM' ? elt.querySelector('button[type="submit"]') : elt;
        if (btn && btn.dataset.htmxLoading) {
            delete btn.dataset.htmxLoading;
            btn.disabled = false;
            btn.style.opacity = '';
        }

        // CSP-safe replacements for hx-on::after-request (eval is blocked by CSP)
        if (event.detail.successful) {
            var id = elt.id;
            if (id === 'channel-form') {
                elt.reset();
                // Close channel modal via Alpine
                var channelModal = document.getElementById('new-channel-modal');
                if (channelModal && channelModal._x_dataStack) {
                    channelModal._x_dataStack[0].show = false;
                }
                var empty = document.getElementById('channel-empty');
                if (empty) empty.remove();
            } else if (id === 'token-form') {
                elt.reset();
            } else if (id === 'monitor-form') {
                // Close monitor modal via Alpine
                var monModal = document.getElementById('new-monitor-modal');
                if (monModal && monModal._x_dataStack) {
                    monModal._x_dataStack[0].show = false;
                }
                var monEmpty = document.getElementById('monitors-empty');
                if (monEmpty) monEmpty.remove();
                var monTable = document.getElementById('monitors-table-container');
                if (monTable) monTable.classList.remove('hidden');
            } else if (id === 'admin-user-form') {
                document.getElementById('new-user-modal').classList.add('hidden');
                elt.reset();
            }

            // Show success toast for create/delete operations
            var verb = event.detail.requestConfig && event.detail.requestConfig.verb;
            if (verb === 'post' && id && id.indexOf('-form') !== -1) {
                showToast('Created successfully');
            } else if (verb === 'delete') {
                showToast('Deleted successfully');
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
        // Show live indicator
        var liveEl = document.getElementById('sse-live-indicator');
        if (liveEl) liveEl.classList.remove('hidden');
    };

    sseConnection.onerror = function() {
        sseConnection.close();
        // Hide live indicator
        var liveEl = document.getElementById('sse-live-indicator');
        if (liveEl) liveEl.classList.add('hidden');
        if (sseReconnectAttempts < maxReconnectAttempts) {
            sseReconnectAttempts++;
            if (sseReconnectAttempts > 1) {
                showToast('Reconnecting to live updates...', 'warning');
            }
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

    // Highlight flash on update
    agentElement.classList.add('highlight-flash');
    setTimeout(function() { agentElement.classList.remove('highlight-flash'); }, 1000);

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
