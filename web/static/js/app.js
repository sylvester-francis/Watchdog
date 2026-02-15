// WatchDog Application JavaScript

// HTMX Configuration
document.addEventListener('DOMContentLoaded', function() {
    htmx.config.defaultSwapStyle = 'innerHTML';
    htmx.config.defaultSettleDelay = 0;

    document.body.addEventListener('htmx:beforeRequest', function(event) {
        const target = event.detail.elt;
        if (target.querySelector('.htmx-indicator')) {
            target.querySelector('.htmx-indicator').classList.remove('hidden');
        }
    });

    document.body.addEventListener('htmx:afterRequest', function(event) {
        const target = event.detail.elt;
        if (target.querySelector('.htmx-indicator')) {
            target.querySelector('.htmx-indicator').classList.add('hidden');
        }
    });

    document.body.addEventListener('htmx:responseError', function(event) {
        showToast('An error occurred. Please try again.', 'error');
    });
});

// Toast Notification System
function showToast(message, type = 'success') {
    const container = document.getElementById('toast-container');
    if (!container) return;

    const toast = document.createElement('div');
    toast.className = 'toast-enter flex items-center space-x-3 px-4 py-3 rounded-lg border shadow-lg max-w-sm';

    if (type === 'error') {
        toast.classList.add('bg-red-500/10', 'border-red-500/20', 'text-red-400');
    } else if (type === 'warning') {
        toast.classList.add('bg-yellow-500/10', 'border-yellow-500/20', 'text-yellow-400');
    } else {
        toast.classList.add('bg-emerald-500/10', 'border-emerald-500/20', 'text-emerald-400');
    }

    toast.innerHTML = `<span class="text-sm">${message}</span>`;
    container.appendChild(toast);

    setTimeout(() => {
        toast.classList.remove('toast-enter');
        toast.classList.add('toast-exit');
        setTimeout(() => toast.remove(), 300);
    }, 4000);
}

// Confirmation Dialog
function confirmAction(message) {
    return confirm(message);
}

// SSE Connection Management
let sseConnection = null;
let sseReconnectAttempts = 0;
const maxReconnectAttempts = 5;
const reconnectDelay = 3000;

function connectSSE() {
    if (sseConnection) {
        sseConnection.close();
    }

    sseConnection = new EventSource('/sse/events');

    sseConnection.onopen = function() {
        sseReconnectAttempts = 0;
    };

    sseConnection.onerror = function(error) {
        sseConnection.close();

        if (sseReconnectAttempts < maxReconnectAttempts) {
            sseReconnectAttempts++;
            setTimeout(connectSSE, reconnectDelay);
        }
    };

    sseConnection.addEventListener('agent-status', function(event) {
        const data = JSON.parse(event.data);
        updateAgentStatus(data);
    });

    sseConnection.addEventListener('incident-count', function(event) {
        const data = JSON.parse(event.data);
        updateIncidentCount(data.count);
    });
}

function updateAgentStatus(agent) {
    const agentElement = document.getElementById(`agent-${agent.id}`);
    if (!agentElement) return;

    const statusDot = agentElement.querySelector('.w-2.h-2');
    const statusBadge = agentElement.querySelector('span[class*="text-xs"]');

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
    const incidentBadge = document.querySelector('[href="/incidents"] .bg-destructive\\/20');
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

// Keyboard shortcuts
document.addEventListener('keydown', function(event) {
    if (event.key === 'Escape') {
        const modals = document.querySelectorAll('[id$="-modal"]:not(.hidden)');
        modals.forEach(modal => modal.classList.add('hidden'));
    }
});

// Close modal when clicking backdrop
document.addEventListener('click', function(event) {
    if (event.target.classList.contains('fixed') && event.target.classList.contains('inset-0')) {
        event.target.classList.add('hidden');
    }
});

// Format relative time
function formatTimeAgo(timestamp) {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now - date;

    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}d ago`;
    if (hours > 0) return `${hours}h ago`;
    if (minutes > 0) return `${minutes}m ago`;
    return 'just now';
}

// Utility: Copy to clipboard
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        showToast('Copied to clipboard');
    }).catch(() => {
        showToast('Failed to copy', 'error');
    });
}
