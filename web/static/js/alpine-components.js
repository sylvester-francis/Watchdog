// WatchDog Alpine.js Components (CSP-safe)
// Registers all components via alpine:init event, which fires right before
// Alpine initializes — so Alpine global is available at that point.
//
// IMPORTANT: Alpine CSP build cannot evaluate inline expressions.
// All x-data must reference registered component names.
// All event handlers (@click, @input, etc.) must reference method names.
// All x-text, :class, etc. must reference properties or getters.

document.addEventListener('alpine:init', () => {

// 1. baseLayout — body element in layouts/base.html
// Controls sidebar visibility and command palette toggle
Alpine.data('baseLayout', () => ({
    sidebarOpen: false,
    commandPaletteOpen: false,
    toggleSidebar() { this.sidebarOpen = !this.sidebarOpen; },
    closeSidebar() { this.sidebarOpen = false; },
    toggleCommandPalette() { this.commandPaletteOpen = !this.commandPaletteOpen; },
    openCommandPalette() { this.commandPaletteOpen = true; },
    closeCommandPalette() { this.commandPaletteOpen = false; },
    // Modal dispatch helpers (CSP build can't evaluate $dispatch inline)
    openMonitorModal() { window.dispatchEvent(new CustomEvent('open-monitor-modal')); },
    openAgentModal() { window.dispatchEvent(new CustomEvent('open-agent-modal')); },
    openChannelModal() { window.dispatchEvent(new CustomEvent('open-channel-modal')); },
    openTokenModal() { window.dispatchEvent(new CustomEvent('open-token-modal')); },
    openPageModal() { window.dispatchEvent(new CustomEvent('open-page-modal')); },
    get sidebarClass() {
        return this.sidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0';
    },
}));

// 2. commandPalette — partials/command_palette.html
// Cmd+K search palette with keyboard navigation
Alpine.data('commandPalette', () => ({
    query: '',
    selectedIndex: 0,
    results: [],
    allItems: [
        { id: 'nav-dashboard', label: 'Dashboard', url: '/dashboard', icon: 'layout-dashboard', type: 'Page' },
        { id: 'nav-monitors', label: 'Monitors', url: '/monitors', icon: 'activity', type: 'Page' },
        { id: 'nav-incidents', label: 'Incidents', url: '/incidents', icon: 'alert-triangle', type: 'Page' },
        { id: 'nav-new-monitor', label: 'New Monitor', url: '/monitors/new', icon: 'plus-circle', type: 'Action' }
    ],
    init() {
        // Parse server-injected monitor data
        var monitorData = document.getElementById('monitor-data');
        if (monitorData) {
            try {
                var monitors = JSON.parse(monitorData.textContent);
                for (var i = 0; i < monitors.length; i++) {
                    var m = monitors[i];
                    this.allItems.push({
                        id: 'mon-' + (m.ID || m.id),
                        label: m.Name || m.name,
                        url: '/monitors/' + (m.ID || m.id),
                        icon: 'monitor',
                        type: (m.Type || m.type || '').toUpperCase()
                    });
                }
            } catch(e) {
                // Silently ignore parse errors
            }
        }
        this.results = this.allItems.slice(0, 8);

        // Focus search input when palette opens (replaces x-effect)
        this.$watch('commandPaletteOpen', (open) => {
            if (open) {
                this.$nextTick(() => {
                    if (this.$refs.searchInput) {
                        this.$refs.searchInput.focus();
                        this.$refs.searchInput.value = '';
                    }
                    this.query = '';
                    this.search();
                });
            }
        });

        // Re-run search when query changes (CSP build can't use @input="search()")
        this.$watch('query', () => this.search());

        // Highlight selected result via DOM (CSP build can't evaluate ternaries)
        this.$watch('selectedIndex', () => this.updateHighlight());
        this.$watch('results', () => this.$nextTick(() => this.updateHighlight()));
    },
    selectItem(event) {
        var items = Array.from(this.$el.querySelectorAll('[data-cp-item]'));
        var idx = items.indexOf(event.currentTarget);
        if (idx !== -1) this.selectedIndex = idx;
    },
    updateHighlight() {
        var items = this.$el.querySelectorAll('[data-cp-item]');
        for (var i = 0; i < items.length; i++) {
            if (i === this.selectedIndex) {
                items[i].classList.add('bg-accent/10', 'text-foreground');
                items[i].classList.remove('text-muted-foreground');
            } else {
                items[i].classList.remove('bg-accent/10', 'text-foreground');
                items[i].classList.add('text-muted-foreground');
            }
        }
    },
    search() {
        this.selectedIndex = 0;
        if (!this.query) {
            this.results = this.allItems.slice(0, 8);
            return;
        }
        var q = this.query.toLowerCase();
        this.results = this.allItems
            .filter(function(item) {
                return item.label.toLowerCase().indexOf(q) !== -1;
            })
            .slice(0, 8);
    },
    onKeydown(event) {
        if (event.key === 'ArrowDown') {
            event.preventDefault();
            this.selectedIndex = Math.min(this.selectedIndex + 1, this.results.length - 1);
        } else if (event.key === 'ArrowUp') {
            event.preventDefault();
            this.selectedIndex = Math.max(this.selectedIndex - 1, 0);
        } else if (event.key === 'Enter' && this.results.length > 0) {
            event.preventDefault();
            window.location.href = this.results[this.selectedIndex].url;
        }
    },
    get noResults() {
        return this.results.length === 0 && this.query.length > 0;
    },
}));

// 3. mobileNav — nav in pages/landing.html
Alpine.data('mobileNav', () => ({
    mobileOpen: false,
    mobileClosed: true,
    toggle() { this.mobileOpen = !this.mobileOpen; this.mobileClosed = !this.mobileOpen; },
    close() { this.mobileOpen = false; this.mobileClosed = true; },
}));

// 4. dashboardMockup — landing page hero animated demo
// Classes pre-computed because CSP build can't evaluate ternaries in templates
Alpine.data('dashboardMockup', () => ({
    stats: { monitors: 12, healthy: 11 },
    services: [
        { name: 'PostgreSQL', type: 'database', latency: '2ms', dotClass: 'bg-emerald-400 animate-pulse-dot', latencyClass: 'text-muted-foreground' },
        { name: 'Redis Cache', type: 'database', latency: '1ms', dotClass: 'bg-emerald-400 animate-pulse-dot', latencyClass: 'text-muted-foreground' },
        { name: 'API Gateway', type: 'http', latency: '45ms', dotClass: 'bg-emerald-400 animate-pulse-dot', latencyClass: 'text-muted-foreground' },
        { name: 'nginx-proxy', type: 'docker', latency: '0ms', dotClass: 'bg-emerald-400 animate-pulse-dot', latencyClass: 'text-muted-foreground' },
        { name: 'Vault', type: 'http', latency: 'timeout', dotClass: 'bg-red-400', latencyClass: 'text-red-400' },
    ],
    _vaultDown: true,
    _latencies: ['38ms', '42ms', '51ms', '45ms', '33ms', '47ms'],
    _tick: 0,
    init() {
        var self = this;
        setInterval(function() {
            self._tick++;
            // Cycle a healthy service's latency
            var idx = self._tick % 4; // cycle through first 4 services
            var latIdx = self._tick % self._latencies.length;
            var svcType = self.services[idx].type;
            if (svcType === 'database') {
                self.services[idx].latency = (1 + (self._tick % 3)) + 'ms';
            } else if (svcType === 'docker') {
                self.services[idx].latency = '0ms';
            } else {
                self.services[idx].latency = self._latencies[latIdx];
            }
            // Toggle Vault between timeout and recovery
            if (self._tick % 3 === 0) {
                self._vaultDown = !self._vaultDown;
                if (self._vaultDown) {
                    self.services[4].latency = 'timeout';
                    self.services[4].dotClass = 'bg-red-400';
                    self.services[4].latencyClass = 'text-red-400';
                    self.stats.healthy = 11;
                } else {
                    self.services[4].latency = '210ms';
                    self.services[4].dotClass = 'bg-emerald-400 animate-pulse-dot';
                    self.services[4].latencyClass = 'text-muted-foreground';
                    self.stats.healthy = 12;
                }
            }
        }, 2000);
    },
    bars: [
        { cls: 'bg-emerald-500/40', style: 'height: 35%; animation-delay: 0s;' },
        { cls: 'bg-emerald-500/40', style: 'height: 42%; animation-delay: 0.05s;' },
        { cls: 'bg-emerald-500/40', style: 'height: 38%; animation-delay: 0.1s;' },
        { cls: 'bg-emerald-500/40', style: 'height: 55%; animation-delay: 0.15s;' },
        { cls: 'bg-emerald-500/40', style: 'height: 40%; animation-delay: 0.2s;' },
        { cls: 'bg-emerald-500/40', style: 'height: 62%; animation-delay: 0.25s;' },
        { cls: 'bg-emerald-500/40', style: 'height: 45%; animation-delay: 0.3s;' },
        { cls: 'bg-red-500/60', style: 'height: 90%; animation-delay: 0.35s;' },
        { cls: 'bg-emerald-500/40', style: 'height: 48%; animation-delay: 0.4s;' },
        { cls: 'bg-emerald-500/40', style: 'height: 35%; animation-delay: 0.45s;' },
        { cls: 'bg-emerald-500/40', style: 'height: 42%; animation-delay: 0.5s;' },
        { cls: 'bg-emerald-500/40', style: 'height: 38%; animation-delay: 0.55s;' },
        { cls: 'bg-emerald-500/40', style: 'height: 44%; animation-delay: 0.6s;' },
        { cls: 'bg-emerald-500/40', style: 'height: 50%; animation-delay: 0.65s;' },
        { cls: 'bg-emerald-500/40', style: 'height: 35%; animation-delay: 0.7s;' },
    ],
}));

// 4b. copyInstall — landing page terminal copy button
Alpine.data('copyInstall', () => ({
    copied: false,
    notCopied: true,
    copy() {
        var self = this;
        navigator.clipboard.writeText('curl -sSL https://usewatchdog.dev/install | sh').then(function() {
            self.copied = true;
            self.notCopied = false;
            setTimeout(function() { self.copied = false; self.notCopied = true; }, 2000);
        });
    },
}));

// 5. monitorFilter — pages/monitors.html
// Uses JS-driven filtering (not x-show) for CSP compatibility
Alpine.data('monitorFilter', () => ({
    search: '',
    filterType: 'all',
    totalCount: 0,
    httpCount: 0,
    tcpCount: 0,
    pingCount: 0,
    dnsCount: 0,
    tlsCount: 0,
    dockerCount: 0,
    databaseCount: 0,
    systemCount: 0,
    visibleCount: 0,
    init() {
        this.countTypes();
        this.$watch('search', () => this.filterRows());
        this.$watch('filterType', () => this.filterRows());
    },
    _allRows() {
        return document.querySelectorAll('#monitors-table tr[data-type], #infra-monitors-table tr[data-type]');
    },
    countTypes() {
        var rows = this._allRows();
        var counts = { http: 0, tcp: 0, ping: 0, dns: 0, tls: 0, docker: 0, database: 0, system: 0 };
        rows.forEach(function(row) {
            var type = row.dataset.type;
            if (counts[type] !== undefined) counts[type]++;
        });
        this.totalCount = rows.length;
        this.visibleCount = rows.length;
        this.httpCount = counts.http;
        this.tcpCount = counts.tcp;
        this.pingCount = counts.ping;
        this.dnsCount = counts.dns;
        this.tlsCount = counts.tls;
        this.dockerCount = counts.docker;
        this.databaseCount = counts.database;
        this.systemCount = counts.system;
    },
    setFilterAll() { this.filterType = 'all'; },
    setFilterHttp() { this.filterType = 'http'; },
    setFilterTcp() { this.filterType = 'tcp'; },
    setFilterPing() { this.filterType = 'ping'; },
    setFilterDns() { this.filterType = 'dns'; },
    setFilterTls() { this.filterType = 'tls'; },
    setFilterDocker() { this.filterType = 'docker'; },
    setFilterDatabase() { this.filterType = 'database'; },
    setFilterSystem() { this.filterType = 'system'; },
    // CSP-safe count labels (replaces inline x-text expressions)
    get totalCountLabel() { return '(' + this.totalCount + ')'; },
    get httpCountLabel() { return this.httpCount > 0 ? '(' + this.httpCount + ')' : ''; },
    get tcpCountLabel() { return this.tcpCount > 0 ? '(' + this.tcpCount + ')' : ''; },
    get pingCountLabel() { return this.pingCount > 0 ? '(' + this.pingCount + ')' : ''; },
    get dnsCountLabel() { return this.dnsCount > 0 ? '(' + this.dnsCount + ')' : ''; },
    get tlsCountLabel() { return this.tlsCount > 0 ? '(' + this.tlsCount + ')' : ''; },
    get dockerCountLabel() { return this.dockerCount > 0 ? '(' + this.dockerCount + ')' : ''; },
    get databaseCountLabel() { return this.databaseCount > 0 ? '(' + this.databaseCount + ')' : ''; },
    get systemCountLabel() { return this.systemCount > 0 ? '(' + this.systemCount + ')' : ''; },
    _filterClass(type) {
        return this.filterType === type
            ? 'bg-muted text-foreground shadow-sm'
            : 'text-muted-foreground hover:text-foreground hover:bg-muted/50';
    },
    get filterAllClass() { return this._filterClass('all'); },
    get filterHttpClass() { return this._filterClass('http'); },
    get filterTcpClass() { return this._filterClass('tcp'); },
    get filterPingClass() { return this._filterClass('ping'); },
    get filterDnsClass() { return this._filterClass('dns'); },
    get filterTlsClass() { return this._filterClass('tls'); },
    get filterDockerClass() { return this._filterClass('docker'); },
    get filterDatabaseClass() { return this._filterClass('database'); },
    get filterSystemClass() { return this._filterClass('system'); },
    get showNoResults() {
        return this.visibleCount === 0 && this.totalCount > 0;
    },
    filterRows() {
        var rows = this._allRows();
        var ft = this.filterType;
        var q = this.search.toLowerCase();
        var visible = 0;
        var serviceVisible = 0;
        var infraVisible = 0;
        rows.forEach(function(row) {
            var type = row.dataset.type;
            var name = row.dataset.name || '';
            var target = row.dataset.target || '';
            var matchType = ft === 'all' || ft === type;
            var matchSearch = q === '' || name.indexOf(q) !== -1 || target.indexOf(q) !== -1;
            var show = matchType && matchSearch;
            row.style.display = show ? '' : 'none';
            if (show) {
                visible++;
                var inInfra = row.closest('#infra-monitors-table');
                if (inInfra) { infraVisible++; } else { serviceVisible++; }
            }
        });
        this.visibleCount = visible;
        var svc = document.getElementById('services-section');
        var infra = document.getElementById('infra-section');
        if (svc) svc.style.display = serviceVisible > 0 ? '' : 'none';
        if (infra) infra.style.display = infraVisible > 0 ? '' : 'none';
    },
}));

// 6. channelSelector — settings.html alert channel modal
// NOTE: Alpine CSP build breaks @change on <select>, x-show reactivity,
// x-model, and getters. We bypass Alpine entirely for DOM updates:
// vanilla addEventListener + direct style.display manipulation.
Alpine.data('channelSelector', () => ({
    channelType: 'discord',
    show: false,
    _bound: false,
    init() {
        var self = this;
        var sel = this.$el.querySelector('select[name="type"]');
        if (sel && !this._bound) {
            this._bound = true;
            sel.addEventListener('change', function() {
                self.channelType = sel.value;
                self._syncSections();
            });
        }
    },
    open() {
        this.show = true;
        this.channelType = 'discord';
        var sel = this.$el.querySelector('select[name="type"]');
        if (sel) sel.value = 'discord';
        this._syncSections();
    },
    close() { this.show = false; },
    _syncSections() {
        var t = this.channelType;
        var root = this.$el;
        var sections = root.querySelectorAll('[data-section]');
        for (var i = 0; i < sections.length; i++) {
            var s = sections[i];
            var types = s.getAttribute('data-section').split(',');
            s.style.display = types.indexOf(t) >= 0 ? '' : 'none';
        }
        // Update webhook placeholder based on type
        var webhookInput = root.querySelector('input[name="webhook_url"]');
        if (webhookInput) {
            if (t === 'slack') {
                webhookInput.placeholder = 'https://hooks.slack.com/services/...';
            } else {
                webhookInput.placeholder = 'https://discord.com/api/webhooks/...';
            }
        }
    },
}));

// 7. planEditor — system.html per-row plan editor
Alpine.data('planEditor', () => ({
    editing: false,
    toggle() { this.editing = !this.editing; },
    submitPlan() {
        var form = this.$el.closest('form');
        if (form) form.requestSubmit();
    },
}));

// 8. incidentManager — pages/incidents.html
Alpine.data('incidentManager', () => ({
    view: 'table',
    selectedIncidents: [],
    selectAll: false,
    showTableView() { this.view = 'table'; },
    showTimelineView() { this.view = 'timeline'; },
    get isTableView() { return this.view === 'table'; },
    get isTimelineView() { return this.view === 'timeline'; },
    get hasSelection() { return this.selectedIncidents.length > 0; },
    get selectionCount() { return this.selectedIncidents.length + ' selected'; },
    clearSelection() { this.selectedIncidents = []; this.selectAll = false; },
    toggleSelectAll() {
        if (this.selectAll) {
            var ids = [];
            document.querySelectorAll('[data-incident-id]').forEach(function(el) {
                ids.push(el.dataset.incidentId);
            });
            this.selectedIncidents = ids;
        } else {
            this.selectedIncidents = [];
        }
    },
    bulkAck() {
        if (this.selectedIncidents.length === 0) return;
        if (!confirm('Acknowledge ' + this.selectedIncidents.length + ' incident(s)?')) return;
        this.selectedIncidents.forEach(function(id) {
            fetch('/incidents/' + id + '/ack', { method: 'POST' });
        });
        setTimeout(function() { window.location.reload(); }, 500);
    },
    bulkResolve() {
        if (this.selectedIncidents.length === 0) return;
        if (!confirm('Resolve ' + this.selectedIncidents.length + ' incident(s)?')) return;
        var filter = new URLSearchParams(window.location.search).get('status') || 'active';
        this.selectedIncidents.forEach(function(id) {
            fetch('/incidents/' + id + '/resolve?filter=' + filter, { method: 'POST' });
        });
        setTimeout(function() { window.location.reload(); }, 500);
    },
    get tableViewClass() {
        return this.view === 'table'
            ? 'bg-muted text-foreground shadow-sm'
            : 'text-muted-foreground hover:text-foreground';
    },
    get timelineViewClass() {
        return this.view === 'timeline'
            ? 'bg-muted text-foreground shadow-sm'
            : 'text-muted-foreground hover:text-foreground';
    },
}));

// 9. dropdown — generic kebab/action menu (replaces inline x-data="{ open: false }")
Alpine.data('dropdown', () => ({
    open: false,
    toggle() { this.open = !this.open; },
    close() { this.open = false; },
}));

// 10. monitorModal — monitors.html new monitor dialog
Alpine.data('monitorModal', () => ({
    show: false,
    _bound: false,
    init() {
        // Auto-open when navigating to /monitors/new
        if (document.getElementById('auto-open-modal')) {
            this.$nextTick(() => { this.show = true; });
        }
        // Bind type-switching on select change
        var self = this;
        var sel = this.$el.querySelector('select[name="type"]');
        if (sel && !this._bound) {
            this._bound = true;
            sel.addEventListener('change', function() {
                self._syncTypeFields(sel.value);
            });
        }
    },
    open() {
        this.show = true;
        // Reset type fields when opening
        var sel = this.$el.querySelector('select[name="type"]');
        if (sel) this._syncTypeFields(sel.value);
    },
    close() { this.show = false; },
    _syncTypeFields(type) {
        var root = this.$el;
        // Show/hide conditional fields
        var fields = root.querySelectorAll('[data-monitor-type-field]');
        for (var i = 0; i < fields.length; i++) {
            var f = fields[i];
            f.style.display = f.getAttribute('data-monitor-type-field') === type ? '' : 'none';
        }
        // Update target placeholder and hint
        var targetInput = root.querySelector('#monitor-target-input');
        var targetHint = root.querySelector('#monitor-target-hint');
        var placeholders = {
            http: 'https://example.com/health',
            tcp: 'host:port (e.g. db.internal:5432)',
            ping: 'hostname or IP address',
            dns: 'example.com',
            tls: 'example.com',
            docker: 'container-name or container-id',
            database: 'host:port (e.g. db.internal:5432)',
            system: 'cpu:90 or memory:85 or disk:90:/'
        };
        var hints = {
            http: 'Full URL including protocol (https://)',
            tcp: 'Host and port separated by colon',
            ping: 'Hostname or IP to check reachability',
            dns: 'Domain name to resolve',
            tls: 'Domain name to check certificate',
            docker: 'Container name or ID (agent must have Docker socket access)',
            database: 'Host:port of the database server',
            system: 'Format: metric:threshold (disk also needs path: disk:90:/)'
        };
        if (targetInput) targetInput.placeholder = placeholders[type] || 'target';
        if (targetHint) targetHint.textContent = hints[type] || '';
    },
}));

// 11. agentModal — dashboard.html new agent dialog
Alpine.data('agentModal', () => ({
    show: false,
    open() { this.show = true; },
    close() { this.show = false; },
}));

// 12. tokenModal — settings.html new API token dialog
Alpine.data('tokenModal', () => ({
    show: false,
    open() { this.show = true; },
    close() { this.show = false; },
}));

// 13. statusPageModal — status_pages.html new page dialog
Alpine.data('statusPageModal', () => ({
    show: false,
    open() { this.show = true; },
    close() { this.show = false; },
}));


}); // end alpine:init
