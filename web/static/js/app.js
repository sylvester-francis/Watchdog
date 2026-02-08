// WatchDog Application JavaScript

// HTMX Configuration
document.addEventListener('DOMContentLoaded', function() {
    // Configure HTMX
    htmx.config.defaultSwapStyle = 'innerHTML';
    htmx.config.defaultSettleDelay = 0;

    // Add loading indicator to HTMX requests
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

    // Handle HTMX errors
    document.body.addEventListener('htmx:responseError', function(event) {
        showToast('An error occurred. Please try again.', 'error');
    });
});

// Toast Notification System
function showToast(message, type = 'success') {
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    document.body.appendChild(toast);

    // Auto-remove after 5 seconds
    setTimeout(() => {
        toast.style.animation = 'slide-out 0.3s ease-in forwards';
        setTimeout(() => toast.remove(), 300);
    }, 5000);
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
        console.log('SSE connection established');
        sseReconnectAttempts = 0;
    };

    sseConnection.onerror = function(error) {
        console.error('SSE connection error:', error);
        sseConnection.close();

        if (sseReconnectAttempts < maxReconnectAttempts) {
            sseReconnectAttempts++;
            console.log(`Reconnecting SSE (attempt ${sseReconnectAttempts}/${maxReconnectAttempts})...`);
            setTimeout(connectSSE, reconnectDelay);
        } else {
            console.log('Max SSE reconnection attempts reached');
        }
    };

    // Handle agent status updates
    sseConnection.addEventListener('agent-status', function(event) {
        const data = JSON.parse(event.data);
        updateAgentStatus(data);
    });

    // Handle incident count updates
    sseConnection.addEventListener('incident-count', function(event) {
        const data = JSON.parse(event.data);
        updateIncidentCount(data.count);
    });
}

function updateAgentStatus(agent) {
    const agentElement = document.getElementById(`agent-${agent.id}`);
    if (!agentElement) return;

    const statusDot = agentElement.querySelector('.w-3.h-3');
    const statusText = agentElement.querySelector('.text-sm');
    const statusBadge = agentElement.querySelector('span[class*="px-2"]');

    if (agent.status === 'online') {
        statusDot.className = 'w-3 h-3 rounded-full bg-green-400 animate-pulse-dot';
        statusBadge.className = 'px-2 py-1 text-xs rounded bg-green-500/20 text-green-400';
        statusBadge.textContent = 'online';
    } else {
        statusDot.className = 'w-3 h-3 rounded-full bg-gray-500';
        statusBadge.className = 'px-2 py-1 text-xs rounded bg-gray-600 text-gray-400';
        statusBadge.textContent = 'offline';
    }
}

function updateIncidentCount(count) {
    const incidentBadge = document.querySelector('[href="/incidents"] .bg-red-500');
    if (count > 0) {
        if (incidentBadge) {
            incidentBadge.textContent = count;
        }
        // Update stats card if present
        const statsCard = document.querySelector('[class*="text-red-400"]');
        if (statsCard && statsCard.closest('.bg-gray-800')) {
            statsCard.textContent = count;
        }
    } else if (incidentBadge) {
        incidentBadge.remove();
    }
}

// Initialize SSE if on a page that needs it
if (document.querySelector('[sse-connect]') || window.location.pathname === '/dashboard') {
    // SSE is handled by HTMX sse extension
    console.log('SSE managed by HTMX');
}

// Keyboard shortcuts
document.addEventListener('keydown', function(event) {
    // ESC to close modals
    if (event.key === 'Escape') {
        const modals = document.querySelectorAll('[id$="-modal"]:not(.hidden)');
        modals.forEach(modal => modal.classList.add('hidden'));
    }
});

// Close modal when clicking outside
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
        showToast('Copied to clipboard', 'success');
    }).catch(() => {
        showToast('Failed to copy', 'error');
    });
}
