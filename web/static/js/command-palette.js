// WatchDog Command Palette (Alpine.js component)
// Usage: <div x-data="commandPalette()">

function commandPalette() {
    return {
        query: '',
        selectedIndex: 0,
        results: [],
        allItems: [
            { id: 'nav-dashboard', label: 'Dashboard', url: '/dashboard', icon: 'layout-dashboard', type: 'Page' },
            { id: 'nav-monitors', label: 'Monitors', url: '/monitors', icon: 'activity', type: 'Page' },
            { id: 'nav-incidents', label: 'Incidents', url: '/incidents', icon: 'alert-triangle', type: 'Page' },
            { id: 'nav-new-monitor', label: 'New Monitor', url: '/monitors/new', icon: 'plus-circle', type: 'Action' }
        ],

        init: function() {
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
        },

        search: function() {
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

        onKeydown: function(event) {
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
        }
    };
}
