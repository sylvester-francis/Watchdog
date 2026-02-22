// WatchDog Charts & Animation Helpers
// Dependencies: Chart.js 4.x

// Chart.js global defaults for dark theme
if (typeof Chart !== 'undefined') {
    Chart.defaults.color = '#a1a1aa';
    Chart.defaults.borderColor = '#27272a';
    Chart.defaults.font.family = "'Inter', system-ui, sans-serif";
    Chart.defaults.font.size = 11;
    Chart.defaults.responsive = true;
    Chart.defaults.maintainAspectRatio = false;
}

// Sparkline factory — inline mini chart with no axes, labels, or legend
function createSparkline(canvasId, dataPoints, color) {
    color = color || '#3b82f6';
    var ctx = document.getElementById(canvasId);
    if (!ctx || !dataPoints || dataPoints.length === 0) return null;
    return new Chart(ctx, {
        type: 'line',
        data: {
            labels: dataPoints.map(function(_, i) { return i; }),
            datasets: [{
                data: dataPoints,
                borderColor: color,
                backgroundColor: color + '20',
                borderWidth: 1.5,
                fill: true,
                tension: 0.4,
                pointRadius: 0,
                pointHitRadius: 0
            }]
        },
        options: {
            plugins: { legend: { display: false }, tooltip: { enabled: false } },
            scales: { x: { display: false }, y: { display: false } },
            animation: { duration: 800 }
        }
    });
}

// Uptime bar builder (Better Stack style) — proportional colored segments
function createUptimeBar(containerId, uptimeUp, uptimeDown, uptimeTotal) {
    var container = document.getElementById(containerId);
    if (!container || uptimeTotal === 0) return;
    container.innerHTML = '';
    container.className = 'flex gap-px h-full w-full rounded-full overflow-hidden bg-muted';

    if (uptimeUp > 0) {
        var upBar = document.createElement('div');
        upBar.style.width = ((uptimeUp / uptimeTotal) * 100) + '%';
        upBar.className = 'h-full bg-emerald-500 rounded-sm';
        upBar.title = uptimeUp + ' up';
        container.appendChild(upBar);
    }
    if (uptimeDown > 0) {
        var downBar = document.createElement('div');
        downBar.style.width = ((uptimeDown / uptimeTotal) * 100) + '%';
        downBar.className = 'h-full bg-red-500 rounded-sm';
        downBar.title = uptimeDown + ' down';
        container.appendChild(downBar);
    }
    var unknown = uptimeTotal - uptimeUp - uptimeDown;
    if (unknown > 0) {
        var unkBar = document.createElement('div');
        unkBar.style.width = ((unknown / uptimeTotal) * 100) + '%';
        unkBar.className = 'h-full bg-zinc-700 rounded-sm';
        unkBar.title = unknown + ' unknown';
        container.appendChild(unkBar);
    }
}

// Full latency line chart
function createLatencyChart(canvasId, labels, data) {
    var ctx = document.getElementById(canvasId);
    if (!ctx) return null;
    return new Chart(ctx, {
        type: 'line',
        data: {
            labels: labels,
            datasets: [{
                label: 'Latency (ms)',
                data: data,
                borderColor: '#3b82f6',
                backgroundColor: 'rgba(59, 130, 246, 0.1)',
                fill: true,
                tension: 0.3,
                borderWidth: 2,
                pointRadius: 2,
                pointBackgroundColor: '#3b82f6'
            }]
        },
        options: {
            plugins: {
                legend: { display: false },
                tooltip: {
                    backgroundColor: '#111113',
                    borderColor: '#27272a',
                    borderWidth: 1,
                    titleFont: { family: "'Inter'" },
                    bodyFont: { family: "'JetBrains Mono'", size: 12 },
                    callbacks: {
                        label: function(ctx) { return ctx.parsed.y + 'ms'; }
                    }
                }
            },
            scales: {
                x: {
                    grid: { color: '#27272a' },
                    ticks: { maxTicksLimit: 8 }
                },
                y: {
                    grid: { color: '#27272a' },
                    ticks: { callback: function(v) { return v + 'ms'; } },
                    beginAtZero: true
                }
            }
        }
    });
}

// System metric usage chart (CPU/memory/disk %) with threshold line
function createMetricChart(canvasId, labels, data, threshold, metricName) {
    var ctx = document.getElementById(canvasId);
    if (!ctx) return null;

    var datasets = [{
        label: metricName + ' usage (%)',
        data: data,
        borderColor: '#3b82f6',
        backgroundColor: 'rgba(59, 130, 246, 0.1)',
        fill: true,
        tension: 0.3,
        borderWidth: 2,
        pointRadius: 2,
        pointBackgroundColor: '#3b82f6'
    }];

    if (threshold > 0) {
        datasets.push({
            label: 'Threshold (' + threshold + '%)',
            data: data.map(function() { return threshold; }),
            borderColor: '#ef4444',
            borderWidth: 1.5,
            borderDash: [5, 3],
            pointRadius: 0,
            fill: false
        });
    }

    return new Chart(ctx, {
        type: 'line',
        data: { labels: labels, datasets: datasets },
        options: {
            plugins: {
                legend: {
                    display: threshold > 0,
                    labels: { boxWidth: 12, padding: 16, font: { size: 10 } }
                },
                tooltip: {
                    backgroundColor: '#111113',
                    borderColor: '#27272a',
                    borderWidth: 1,
                    titleFont: { family: "'Inter'" },
                    bodyFont: { family: "'JetBrains Mono'", size: 12 },
                    callbacks: {
                        label: function(ctx) { return ctx.parsed.y.toFixed(1) + '%'; }
                    }
                }
            },
            scales: {
                x: {
                    grid: { color: '#27272a' },
                    ticks: { maxTicksLimit: 8 }
                },
                y: {
                    grid: { color: '#27272a' },
                    ticks: { callback: function(v) { return v + '%'; } },
                    beginAtZero: true,
                    max: 100
                }
            }
        }
    });
}

// Count-up animation for stat numbers
function animateCounter(element, target, duration) {
    duration = duration || 1200;
    var startTime = performance.now();
    function update(currentTime) {
        var elapsed = currentTime - startTime;
        var progress = Math.min(elapsed / duration, 1);
        var eased = 1 - Math.pow(1 - progress, 3); // ease-out cubic
        element.textContent = Math.floor(target * eased);
        if (progress < 1) requestAnimationFrame(update);
    }
    requestAnimationFrame(update);
}

// Intersection Observer for scroll reveal animations
document.addEventListener('DOMContentLoaded', function() {
    // Scroll reveals
    var observer = new IntersectionObserver(function(entries) {
        entries.forEach(function(entry) {
            if (entry.isIntersecting) {
                entry.target.classList.add('visible');
                observer.unobserve(entry.target);
            }
        });
    }, { threshold: 0.1 });

    document.querySelectorAll('.reveal').forEach(function(el) {
        observer.observe(el);
    });

    // Initialize animated counters
    document.querySelectorAll('[data-count-target]').forEach(function(el) {
        var target = parseInt(el.dataset.countTarget, 10);
        if (!isNaN(target)) {
            animateCounter(el, target);
        }
    });

    // Initialize Lucide icons if available
    if (typeof lucide !== 'undefined') {
        lucide.createIcons();
    }
});
