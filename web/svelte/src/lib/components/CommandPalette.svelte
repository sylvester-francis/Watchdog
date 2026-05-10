<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { monitors as monitorsApi } from '$lib/api';
	import { Search, LayoutDashboard, Activity, AlertTriangle, Globe, Settings, Monitor, ShieldAlert, PlusCircle } from 'lucide-svelte';

	let open = $state(false);
	let query = $state('');
	let selectedIndex = $state(0);
	let inputEl = $state<HTMLInputElement | null>(null);
	let pendingG = $state(false);
	let gTimer: ReturnType<typeof setTimeout> | null = null;

	interface PaletteItem {
		id: string;
		label: string;
		url: string;
		type: string;
		icon?: typeof LayoutDashboard;
	}

	const staticItems: PaletteItem[] = [
		{ id: 'nav-dashboard', label: 'Dashboard', url: '/dashboard', type: 'Page', icon: LayoutDashboard },
		{ id: 'nav-monitors', label: 'Monitors', url: '/monitors', type: 'Page', icon: Activity },
		{ id: 'nav-incidents', label: 'Incidents', url: '/incidents', type: 'Page', icon: AlertTriangle },
		{ id: 'nav-status-pages', label: 'Status Pages', url: '/status-pages', type: 'Page', icon: Globe },
		{ id: 'nav-settings', label: 'Settings', url: '/settings', type: 'Page', icon: Settings },
		{ id: 'nav-system', label: 'System', url: '/system', type: 'Page', icon: Monitor },
		{ id: 'nav-security', label: 'Security', url: '/security', type: 'Page', icon: ShieldAlert },
		{ id: 'act-new-monitor', label: 'New Monitor', url: '/monitors?new=1', type: 'Action', icon: PlusCircle },
	];

	let monitorItems = $state<PaletteItem[]>([]);
	let allItems = $derived([...staticItems, ...monitorItems]);

	let results = $derived.by(() => {
		if (!query) return allItems.slice(0, 10);
		const q = query.toLowerCase();
		return allItems
			.filter((item) => item.label.toLowerCase().includes(q) || item.type.toLowerCase().includes(q))
			.slice(0, 10);
	});

	function openPalette() {
		open = true;
		query = '';
		selectedIndex = 0;
		loadMonitors();
		requestAnimationFrame(() => {
			inputEl?.focus();
		});
	}

	function closePalette() {
		open = false;
	}

	function selectItem(index: number) {
		const item = results[index];
		if (item) {
			closePalette();
			goto(item.url);
		}
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			selectedIndex = Math.min(selectedIndex + 1, results.length - 1);
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			selectedIndex = Math.max(selectedIndex - 1, 0);
		} else if (e.key === 'Enter' && results.length > 0) {
			e.preventDefault();
			selectItem(selectedIndex);
		}
	}

	async function loadMonitors() {
		if (monitorItems.length > 0) return;
		try {
			const res = await monitorsApi.listMonitors();
			const monitors = res?.data ?? [];
			monitorItems = monitors.map((m) => ({
				id: `mon-${m.id}`,
				label: m.name,
				url: `/monitors/${m.id}`,
				type: (m.type || '').toUpperCase(),
			}));
		} catch {
			// Silently ignore
		}
	}

	function isInputFocused(): boolean {
		const tag = document.activeElement?.tagName;
		return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' ||
			(document.activeElement as HTMLElement)?.isContentEditable === true;
	}

	function handleGlobalKeydown(e: KeyboardEvent) {
		// Cmd/Ctrl+K — open command palette
		if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
			e.preventDefault();
			if (open) closePalette();
			else openPalette();
			return;
		}

		// Escape — close palette
		if (e.key === 'Escape' && open) {
			closePalette();
			return;
		}

		// Skip keyboard nav shortcuts when typing in inputs
		if (isInputFocused() || open) return;

		// Two-key navigation: G then D/M/I/S/P
		if (e.key === 'g' && !e.metaKey && !e.ctrlKey) {
			pendingG = true;
			if (gTimer) clearTimeout(gTimer);
			gTimer = setTimeout(() => { pendingG = false; }, 500);
			return;
		}

		if (pendingG) {
			pendingG = false;
			if (gTimer) clearTimeout(gTimer);
			switch (e.key) {
				case 'd': goto('/dashboard'); break;
				case 'm': goto('/monitors'); break;
				case 'i': goto('/incidents'); break;
				case 's': goto('/settings'); break;
				case 'p': goto('/status-pages'); break;
			}
		}
	}

	onMount(() => {
		window.addEventListener('keydown', handleGlobalKeydown);
	});

	onDestroy(() => {
		window.removeEventListener('keydown', handleGlobalKeydown);
		if (gTimer) clearTimeout(gTimer);
	});
</script>

{#if open}
	<!-- Backdrop -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-[100] bg-black/60 backdrop-blur-sm"
		onclick={closePalette}
		onkeydown={(e) => e.key === 'Escape' && closePalette()}
	></div>

	<!-- Palette -->
	<div class="fixed inset-0 z-[101] flex items-start justify-center pt-[20vh]">
		<div class="w-full max-w-lg bg-card border border-border rounded-xl shadow-2xl overflow-hidden">
			<!-- Search input -->
			<div class="flex items-center px-4 border-b border-border">
				<Search class="w-4 h-4 text-muted-foreground shrink-0" />
				<input
					bind:this={inputEl}
					bind:value={query}
					onkeydown={onKeydown}
					type="text"
					placeholder="Search pages and monitors..."
					class="w-full bg-transparent px-3 py-3 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none"
				/>
				<kbd class="shrink-0 px-1.5 py-0.5 rounded bg-muted text-muted-foreground text-[10px] font-mono">ESC</kbd>
			</div>

			<!-- Results -->
			<div class="max-h-72 overflow-y-auto p-2">
				{#if results.length === 0 && query.length > 0}
					<div class="px-3 py-8 text-center text-sm text-muted-foreground">
						No results found
					</div>
				{:else}
					{#each results as item, index}
						<button
							class="w-full flex items-center space-x-3 px-3 py-2.5 rounded-lg text-sm transition-colors cursor-pointer text-left
								{index === selectedIndex ? 'bg-foreground/[0.08] text-foreground' : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'}"
							onmouseenter={() => selectedIndex = index}
							onclick={() => selectItem(index)}
						>
							{#if item.icon}
								<item.icon class="w-4 h-4 shrink-0" />
							{:else}
								<Activity class="w-4 h-4 shrink-0" />
							{/if}
							<span class="flex-1 truncate">{item.label}</span>
							<span class="shrink-0 text-[10px] uppercase tracking-wider px-1.5 py-0.5 rounded bg-muted text-muted-foreground font-medium">
								{item.type}
							</span>
						</button>
					{/each}
				{/if}
			</div>

			<!-- Footer -->
			<div class="px-4 py-2 border-t border-border flex items-center space-x-4 text-[10px] text-muted-foreground">
				<span class="flex items-center space-x-1">
					<kbd class="px-1 py-0.5 rounded bg-muted font-mono">&uarr;&darr;</kbd>
					<span>Navigate</span>
				</span>
				<span class="flex items-center space-x-1">
					<kbd class="px-1 py-0.5 rounded bg-muted font-mono">&crarr;</kbd>
					<span>Open</span>
				</span>
				<span class="flex items-center space-x-1">
					<kbd class="px-1 py-0.5 rounded bg-muted font-mono">ESC</kbd>
					<span>Close</span>
				</span>
			</div>
		</div>
	</div>
{/if}
