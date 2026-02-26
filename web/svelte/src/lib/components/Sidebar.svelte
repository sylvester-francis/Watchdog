<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { getAuth } from '$lib/stores/auth.svelte';
	import { LayoutDashboard, Activity, AlertTriangle, Globe, Settings, Monitor, ShieldCheck, ShieldAlert, MessageCircle, X, Search } from 'lucide-svelte';

	const auth = getAuth();
	let mobileOpen = $state(false);
	let isMac = $state(false);

	function toggleMobile() {
		mobileOpen = !mobileOpen;
	}

	function closeMobile() {
		mobileOpen = false;
	}

	onMount(() => {
		isMac = navigator.platform?.includes('Mac') ?? false;
		function handleToggle() { toggleMobile(); }
		window.addEventListener('toggle-sidebar', handleToggle);
		return () => window.removeEventListener('toggle-sidebar', handleToggle);
	});

	const navItems = [
		{ href: `/dashboard`, label: 'Dashboard', icon: LayoutDashboard, group: 'Monitoring', shortcut: 'G D' },
		{ href: `/monitors`, label: 'Monitors', icon: Activity, group: 'Monitoring', shortcut: 'G M' },
		{ href: `/incidents`, label: 'Incidents', icon: AlertTriangle, group: 'Monitoring', shortcut: 'G I' },
		{ href: `/status-pages`, label: 'Status Pages', icon: Globe, group: 'Monitoring', shortcut: 'G P' },
	];

	const systemItems = [
		{ href: `/settings`, label: 'Settings', icon: Settings },
		{ href: `/system`, label: 'System', icon: Monitor },
		{ href: `/security`, label: 'Security', icon: ShieldAlert },
	];

	function isActive(href: string): boolean {
		return page.url.pathname === href || page.url.pathname.startsWith(href + '/');
	}
</script>

<!-- Mobile overlay -->
{#if mobileOpen}
	<div
		class="lg:hidden fixed inset-0 bg-black/50 z-40"
		onclick={closeMobile}
		onkeydown={(e) => e.key === 'Escape' && closeMobile()}
		role="button"
		tabindex="-1"
		aria-label="Close navigation"
	></div>
{/if}

<aside class="fixed left-0 top-0 h-full w-64 bg-card border-r border-border z-50 flex flex-col transition-transform duration-200
	{mobileOpen ? 'translate-x-0' : '-translate-x-full'} lg:translate-x-0">

	<!-- Logo -->
	<div class="px-5 h-14 flex items-center border-b border-border">
		<a href="/dashboard" class="flex items-center space-x-3" title="WatchDog v0.1.0-beta">
			<div class="w-8 h-8 bg-accent rounded-lg flex items-center justify-center">
				<ShieldCheck class="w-4 h-4 text-white" />
			</div>
			<span class="text-base font-semibold text-foreground tracking-tight">WatchDog</span>
		</a>
		<button class="lg:hidden ml-auto p-1.5 rounded-md text-muted-foreground hover:text-foreground" onclick={closeMobile}>
			<X class="w-4 h-4" />
		</button>
	</div>

	<!-- Search shortcut -->
	<button
		onclick={() => window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', metaKey: true }))}
		class="mx-3 mt-3 mb-1 flex items-center space-x-2 px-3 py-1.5 rounded-md border border-border/50 text-xs text-muted-foreground hover:text-foreground hover:border-border transition-colors cursor-pointer"
	>
		<Search class="w-3.5 h-3.5" />
		<span class="flex-1 text-left">Search...</span>
		<kbd class="hidden sm:inline-flex px-1.5 py-0.5 rounded bg-muted text-[10px] font-mono">{isMac ? '\u2318' : 'Ctrl+'}K</kbd>
	</button>

	<!-- Navigation -->
	<nav class="flex-1 px-3 py-4 space-y-0.5" aria-label="Main navigation">
		<p class="text-[9px] uppercase tracking-wider text-muted-foreground/40 px-3 mb-1">Monitoring</p>

		{#each navItems as item}
			<a
				href={item.href}
				onclick={closeMobile}
				class="group flex items-center space-x-3 px-3 py-2 rounded-md text-sm sidebar-link
					{isActive(item.href) ? 'bg-foreground/[0.08] text-foreground font-medium' : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'}"
			>
				<item.icon class="w-4 h-4" />
				<span class="flex-1">{item.label}</span>
				{#if item.shortcut}
					<kbd class="hidden lg:inline-flex px-1 py-0.5 rounded bg-muted/50 text-[9px] font-mono text-muted-foreground/40 group-hover:text-muted-foreground/60 transition-colors">{item.shortcut}</kbd>
				{/if}
			</a>
		{/each}

		<div class="pt-4 mt-2 border-t border-border/50"></div>
		<p class="text-[9px] uppercase tracking-wider text-muted-foreground/40 px-3 mb-1 mt-2">System</p>

		{#each systemItems as item}
			<a
				href={item.href}
				onclick={closeMobile}
				class="group flex items-center space-x-3 px-3 py-2 rounded-md text-sm sidebar-link
					{isActive(item.href) ? 'bg-foreground/[0.08] text-foreground font-medium' : 'text-muted-foreground hover:text-foreground hover:bg-muted/50'}"
			>
				<item.icon class="w-4 h-4" />
				<span>{item.label}</span>
			</a>
		{/each}
	</nav>

	<!-- Footer -->
	<div class="px-4 h-12 flex items-center border-t border-border/50">
		<div class="flex items-center justify-between w-full">
			{#if auth.user?.plan}
				<span class="text-xs px-2 py-0.5 rounded-md font-medium cursor-default bg-muted text-muted-foreground">
					{auth.user.plan === 'beta' ? 'Beta' : auth.user.plan}
				</span>
			{:else}
				<span></span>
			{/if}
			<a href="https://discord.gg/PPPjZDVS" target="_blank" rel="noopener noreferrer"
				class="p-1.5 rounded-md text-muted-foreground/40 hover:text-muted-foreground hover:bg-muted/50 transition-smooth"
				title="Feedback & Community"
				aria-label="Feedback and Community on Discord">
				<MessageCircle class="w-4 h-4" />
			</a>
		</div>
	</div>
</aside>
