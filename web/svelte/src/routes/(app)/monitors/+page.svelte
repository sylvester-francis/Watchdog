<script lang="ts">
	import { onMount } from 'svelte';
	import {
		Plus,
		Search,
		Activity,
		Globe,
		HardDrive,
		MoreHorizontal,
		Eye,
		Trash2
	} from 'lucide-svelte';
	import { monitors as monitorsApi, agents as agentsApi } from '$lib/api';
	import { formatPercent, uptimeColor, isInfraMonitor } from '$lib/utils';
	import type { MonitorSummary, Agent, MonitorType } from '$lib/types';
	import UptimeChecks from '$lib/components/dashboard/UptimeChecks.svelte';
	import Sparkline from '$lib/components/dashboard/Sparkline.svelte';
	import CreateMonitorModal from '$lib/components/monitors/CreateMonitorModal.svelte';
	import { getToasts } from '$lib/stores/toast.svelte';

	const toast = getToasts();

	let summaries = $state<MonitorSummary[]>([]);
	let agentList = $state<Agent[]>([]);
	let loading = $state(true);
	let showCreateModal = $state(false);

	// Filters
	let activeFilter = $state<'all' | MonitorType>('all');
	let searchQuery = $state('');

	// Delete confirmation state
	let confirmDeleteId = $state<string | null>(null);
	let deleting = $state(false);

	// Actions dropdown state
	let openDropdownId = $state<string | null>(null);

	const filterTabs: { value: 'all' | MonitorType; label: string }[] = [
		{ value: 'all', label: 'All' },
		{ value: 'http', label: 'HTTP' },
		{ value: 'tcp', label: 'TCP' },
		{ value: 'ping', label: 'Ping' },
		{ value: 'dns', label: 'DNS' },
		{ value: 'tls', label: 'TLS' },
		{ value: 'docker', label: 'Docker' },
		{ value: 'database', label: 'Database' },
		{ value: 'system', label: 'System' }
	];

	// Filtered monitors
	let filtered = $derived.by(() => {
		let result = summaries;

		// Type filter
		if (activeFilter !== 'all') {
			result = result.filter((m) => m.type === activeFilter);
		}

		// Search filter
		if (searchQuery.trim()) {
			const q = searchQuery.trim().toLowerCase();
			result = result.filter(
				(m) => m.name.toLowerCase().includes(q) || m.target.toLowerCase().includes(q)
			);
		}

		return result;
	});

	let services = $derived(filtered.filter((m) => !isInfraMonitor(m.type)));
	let infra = $derived(filtered.filter((m) => isInfraMonitor(m.type)));

	function uptimePercent(m: MonitorSummary): number {
		if (m.total === 0) return 0;
		return (m.uptimeUp / m.total) * 100;
	}

	function sparkColor(status: string): string {
		if (status === 'up') return '#22c55e';
		if (status === 'down') return '#ef4444';
		return '#a1a1aa';
	}

	function lastLatency(m: MonitorSummary): string {
		if (!m.latencies || m.latencies.length === 0) return '--';
		return m.latencies[m.latencies.length - 1] + 'ms';
	}

	function checkResults(m: MonitorSummary): number[] {
		if (m.total === 0) return [];
		const results: number[] = [];
		const up = m.uptimeUp;
		const down = m.uptimeDown;
		for (let i = 0; i < up && results.length < 20; i++) results.push(1);
		for (let i = 0; i < down && results.length < 20; i++) results.push(0);
		return results;
	}

	function infraValue(m: MonitorSummary): string {
		if (m.type === 'docker') return m.status === 'up' ? 'Running' : 'Stopped';
		if (m.type === 'database' && m.latencies?.length > 0)
			return m.latencies[m.latencies.length - 1] + 'ms';
		if (m.type === 'system') return '--';
		return 'No data';
	}

	function infraValueClass(m: MonitorSummary): string {
		if (m.type === 'docker') return m.status === 'up' ? 'text-emerald-400' : 'text-red-400';
		return 'text-muted-foreground';
	}

	function statusDotClass(status: string): string {
		if (status === 'up') return 'bg-emerald-400 shadow-[0_0_6px_rgba(34,197,94,0.4)]';
		if (status === 'down') return 'bg-red-400 shadow-[0_0_6px_rgba(239,68,68,0.4)]';
		if (status === 'degraded') return 'bg-amber-400';
		return 'bg-muted-foreground/50';
	}

	function toggleDropdown(id: string) {
		openDropdownId = openDropdownId === id ? null : id;
	}

	function closeDropdown() {
		openDropdownId = null;
	}

	async function handleDelete(id: string) {
		if (confirmDeleteId !== id) {
			confirmDeleteId = id;
			return;
		}

		deleting = true;
		try {
			await monitorsApi.deleteMonitor(id);
			summaries = summaries.filter((m) => m.id !== id);
			confirmDeleteId = null;
			openDropdownId = null;
			toast.success('Monitor deleted');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete monitor');
		} finally {
			deleting = false;
		}
	}

	function cancelDelete() {
		confirmDeleteId = null;
	}

	async function loadData() {
		try {
			const [summaryRes, agentsRes] = await Promise.all([
				monitorsApi.getMonitorsSummary(),
				agentsApi.listAgents()
			]);
			summaries = summaryRes ?? [];
			agentList = agentsRes.data ?? [];
		} catch {
			// Keep defaults on error
		} finally {
			loading = false;
		}
	}

	function handleMonitorCreated() {
		loadData();
		toast.success('Monitor created');
	}

	// Close dropdown when clicking outside
	function handleWindowClick() {
		if (openDropdownId) {
			openDropdownId = null;
		}
	}

	onMount(() => {
		loadData();
	});
</script>

<svelte:head>
	<title>Monitors - WatchDog</title>
</svelte:head>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<svelte:window onclick={handleWindowClick} />

{#if loading}
	<!-- Skeleton loading state -->
	<div class="animate-fade-in-up space-y-4">
		<!-- Header skeleton -->
		<div class="flex items-center justify-between">
			<div class="h-7 w-32 bg-muted/50 rounded animate-pulse"></div>
			<div class="h-9 w-32 bg-muted/50 rounded-md animate-pulse"></div>
		</div>
		<!-- Filter bar skeleton -->
		<div class="flex items-center space-x-2">
			{#each Array(5) as _}
				<div class="h-7 w-16 bg-muted/50 rounded-md animate-pulse"></div>
			{/each}
		</div>
		<!-- Table skeleton -->
		<div class="bg-card border border-border rounded-lg">
			{#each Array(5) as _}
				<div class="flex items-center px-4 py-4 border-b border-border/20">
					<div class="w-2.5 h-2.5 bg-muted/50 rounded-full mr-3"></div>
					<div class="flex-1 space-y-1.5">
						<div class="h-4 w-40 bg-muted/50 rounded animate-pulse"></div>
						<div class="h-3 w-56 bg-muted/30 rounded animate-pulse"></div>
					</div>
					<div class="h-4 w-28 bg-muted/50 rounded animate-pulse hidden md:block"></div>
					<div class="h-4 w-14 bg-muted/50 rounded animate-pulse hidden md:block ml-4"></div>
				</div>
			{/each}
		</div>
	</div>
{:else}
	<div class="animate-fade-in-up">
		<!-- Page header -->
		<div class="flex items-center justify-between mb-5">
			<div>
				<h1 class="text-lg font-semibold text-foreground">Monitors</h1>
				<p class="text-xs text-muted-foreground mt-0.5">
					{summaries.length} monitor{summaries.length !== 1 ? 's' : ''} configured
				</p>
			</div>
			<button
				onclick={() => { showCreateModal = true; }}
				class="flex items-center space-x-1.5 px-3 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors"
			>
				<Plus class="w-3.5 h-3.5" />
				<span>New Monitor</span>
			</button>
		</div>

		<!-- Filter bar -->
		<div class="flex flex-col sm:flex-row items-start sm:items-center gap-3 mb-4">
			<!-- Type filter tabs -->
			<div class="flex items-center flex-wrap gap-1">
				{#each filterTabs as tab}
					<button
						onclick={() => { activeFilter = tab.value; }}
						class="px-2.5 py-1 text-xs rounded-md transition-colors {activeFilter === tab.value
							? 'bg-foreground/[0.08] text-foreground font-medium'
							: 'text-muted-foreground hover:text-foreground hover:bg-foreground/[0.04]'}"
					>
						{tab.label}
					</button>
				{/each}
			</div>

			<!-- Search input -->
			<div class="relative w-full sm:w-auto sm:ml-auto">
				<Search class="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground/50" />
				<input
					type="text"
					bind:value={searchQuery}
					placeholder="Search monitors..."
					class="w-full sm:w-56 pl-8 pr-3 py-1.5 bg-card border border-border rounded-md text-xs text-foreground placeholder-muted-foreground/50 focus:outline-none focus:ring-1 focus:ring-ring"
				/>
			</div>
		</div>

		{#if summaries.length === 0}
			<!-- Empty state: no monitors at all -->
			<div class="bg-card border border-border rounded-lg">
				<div class="p-12 text-center">
					<div class="w-12 h-12 bg-muted/50 rounded-lg flex items-center justify-center mx-auto mb-4">
						<Activity class="w-6 h-6 text-muted-foreground/40" />
					</div>
					<p class="text-sm font-medium text-foreground mb-1">No monitors yet</p>
					<p class="text-xs text-muted-foreground mb-4">
						Create a monitor to start tracking your services and infrastructure.
					</p>
					<button
						onclick={() => { showCreateModal = true; }}
						class="inline-flex items-center space-x-1.5 px-4 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors"
					>
						<Plus class="w-3.5 h-3.5" />
						<span>Create Monitor</span>
					</button>
				</div>
			</div>
		{:else if filtered.length === 0}
			<!-- Empty state: filters returned nothing -->
			<div class="bg-card border border-border rounded-lg">
				<div class="p-8 text-center">
					<div class="w-10 h-10 bg-muted/50 rounded-lg flex items-center justify-center mx-auto mb-3">
						<Search class="w-5 h-5 text-muted-foreground/40" />
					</div>
					<p class="text-sm font-medium text-foreground mb-1">No matches</p>
					<p class="text-xs text-muted-foreground">
						No monitors match the current filter. Try adjusting your search or filter.
					</p>
				</div>
			</div>
		{:else}
			<!-- Services section -->
			{#if services.length > 0}
				<div class="bg-card border border-border rounded-lg mb-4">
					<div class="px-4 py-3 border-b border-border flex items-center space-x-2">
						<Globe class="w-4 h-4 text-muted-foreground" />
						<h2 class="text-sm font-medium text-foreground">Services</h2>
						<span class="text-[10px] text-muted-foreground font-mono">{services.length}</span>
					</div>

					<!-- Column headers -->
					<div class="hidden sm:flex items-center px-4 py-2 border-b border-border/30 text-[9px] font-medium text-muted-foreground uppercase tracking-wider">
						<div class="w-5 shrink-0"></div>
						<div class="flex-1 min-w-0 ml-2">Service</div>
						<div class="w-36 shrink-0 text-center hidden md:block">Uptime (24h)</div>
						<div class="w-14 shrink-0 text-right hidden md:block ml-2">Uptime</div>
						<div class="w-16 shrink-0 text-right hidden md:block ml-3">Latency</div>
						<div class="w-14 shrink-0 text-right hidden md:block ml-2">Response</div>
						<div class="w-12 shrink-0 text-right hidden md:block ml-2">Interval</div>
						<div class="w-8 shrink-0"></div>
					</div>

					<!-- Rows -->
					<div class="divide-y divide-border/20">
						{#each services as m (m.id)}
							<div class="flex items-center px-4 py-3.5 hover:bg-card-elevated transition-colors group relative">
								<!-- Status dot -->
								<div class="w-5 shrink-0 flex justify-center">
									<div
										class="w-2 h-2 rounded-full {statusDotClass(m.status)}"
										aria-label="Status: {m.status}"
									></div>
								</div>

								<!-- Name + type + target (clickable link) -->
								<a href="/monitors/{m.id}" class="flex-1 min-w-0 ml-2">
									<div class="flex items-center space-x-2">
										<span class="text-sm text-foreground truncate group-hover:text-accent transition-colors">{m.name}</span>
										<span class="text-[9px] text-muted-foreground font-mono uppercase shrink-0 px-1.5 py-0.5 rounded bg-muted/50">{m.type}</span>
									</div>
									<p class="text-[10px] text-muted-foreground font-mono truncate mt-0.5 hidden sm:block">{m.target}</p>
								</a>

								<!-- Uptime checks -->
								<div class="w-36 shrink-0 hidden md:flex items-center justify-center mx-2">
									{#if m.total > 0}
										<UptimeChecks checkResults={checkResults(m)} />
									{:else}
										<span class="text-[9px] text-muted-foreground">No data</span>
									{/if}
								</div>

								<!-- Uptime % -->
								<div class="w-14 shrink-0 hidden md:flex items-center justify-end ml-2">
									<span class="text-xs font-mono font-medium {uptimeColor(uptimePercent(m))}">{formatPercent(uptimePercent(m))}%</span>
								</div>

								<!-- Sparkline -->
								<div class="w-16 shrink-0 hidden md:flex items-center justify-end ml-3">
									{#if m.latencies?.length > 0}
										<Sparkline values={m.latencies} color={sparkColor(m.status)} />
									{:else}
										<span class="text-[9px] text-muted-foreground">No data</span>
									{/if}
								</div>

								<!-- Response time -->
								<div class="w-14 shrink-0 hidden md:flex items-center justify-end ml-2">
									<span class="text-xs font-mono text-muted-foreground">{lastLatency(m)}</span>
								</div>

								<!-- Interval -->
								<div class="w-12 shrink-0 hidden md:flex items-center justify-end ml-2">
									<span class="text-[10px] font-mono text-muted-foreground/70">--</span>
								</div>

								<!-- Actions dropdown -->
								<div class="w-8 shrink-0 flex justify-end relative">
									<button
										onclick={(e) => { e.stopPropagation(); toggleDropdown(m.id); }}
										class="p-1 rounded hover:bg-muted/50 text-muted-foreground/40 hover:text-muted-foreground transition-colors"
										aria-label="Actions"
									>
										<MoreHorizontal class="w-4 h-4" />
									</button>

									{#if openDropdownId === m.id}
										<!-- svelte-ignore a11y_no_static_element_interactions -->
										<div
											class="absolute right-0 top-8 z-20 w-40 bg-card border border-border rounded-md shadow-lg py-1"
											onclick={(e) => e.stopPropagation()}
										>
											<a
												href="/monitors/{m.id}"
												class="flex items-center space-x-2 px-3 py-1.5 text-xs text-foreground hover:bg-muted/50 transition-colors"
											>
												<Eye class="w-3.5 h-3.5 text-muted-foreground" />
												<span>View Details</span>
											</a>
											{#if confirmDeleteId === m.id}
												<div class="px-3 py-1.5">
													<p class="text-[10px] text-red-400 mb-1.5">Delete this monitor?</p>
													<div class="flex items-center space-x-1.5">
														<button
															onclick={() => handleDelete(m.id)}
															disabled={deleting}
															class="px-2 py-1 text-[10px] bg-red-500/20 text-red-400 hover:bg-red-500/30 rounded transition-colors disabled:opacity-50"
														>
															{deleting ? 'Deleting...' : 'Confirm'}
														</button>
														<button
															onclick={cancelDelete}
															class="px-2 py-1 text-[10px] text-muted-foreground hover:text-foreground transition-colors"
														>
															Cancel
														</button>
													</div>
												</div>
											{:else}
												<button
													onclick={() => handleDelete(m.id)}
													class="flex items-center space-x-2 px-3 py-1.5 text-xs text-red-400 hover:bg-red-500/10 transition-colors w-full text-left"
												>
													<Trash2 class="w-3.5 h-3.5" />
													<span>Delete</span>
												</button>
											{/if}
										</div>
									{/if}
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Infrastructure section -->
			{#if infra.length > 0}
				<div class="bg-card border border-border rounded-lg mb-4">
					<div class="px-4 py-3 border-b border-border flex items-center space-x-2">
						<HardDrive class="w-4 h-4 text-muted-foreground" />
						<h2 class="text-sm font-medium text-foreground">Infrastructure</h2>
						<span class="text-[10px] text-muted-foreground font-mono">{infra.length}</span>
					</div>

					<!-- Column headers -->
					<div class="hidden sm:flex items-center px-4 py-2 border-b border-border/30 text-[9px] font-medium text-muted-foreground uppercase tracking-wider">
						<div class="w-5 shrink-0"></div>
						<div class="flex-1 min-w-0 ml-2">Service</div>
						<div class="w-36 shrink-0 text-center hidden md:block">Health (24h)</div>
						<div class="w-14 shrink-0 text-right hidden md:block ml-2">Uptime</div>
						<div class="w-20 shrink-0 text-right hidden md:block ml-3">Value</div>
						<div class="w-12 shrink-0 text-right hidden md:block ml-2">Interval</div>
						<div class="w-8 shrink-0"></div>
					</div>

					<!-- Rows -->
					<div class="divide-y divide-border/20">
						{#each infra as m (m.id)}
							<div class="flex items-center px-4 py-3.5 hover:bg-card-elevated transition-colors group relative">
								<!-- Status dot -->
								<div class="w-5 shrink-0 flex justify-center">
									<div
										class="w-2 h-2 rounded-full {statusDotClass(m.status)}"
										aria-label="Status: {m.status}"
									></div>
								</div>

								<!-- Name + type + target -->
								<a href="/monitors/{m.id}" class="flex-1 min-w-0 ml-2">
									<div class="flex items-center space-x-2">
										<span class="text-sm text-foreground truncate group-hover:text-accent transition-colors">{m.name}</span>
										<span class="text-[9px] text-muted-foreground font-mono uppercase shrink-0 px-1.5 py-0.5 rounded bg-muted/50">{m.type}</span>
									</div>
									<p class="text-[10px] text-muted-foreground font-mono truncate mt-0.5 hidden sm:block">{m.target}</p>
								</a>

								<!-- Health checks -->
								<div class="w-36 shrink-0 hidden md:flex items-center justify-center mx-2">
									{#if m.total > 0}
										<UptimeChecks checkResults={checkResults(m)} />
									{:else}
										<span class="text-[9px] text-muted-foreground">No data</span>
									{/if}
								</div>

								<!-- Uptime % -->
								<div class="w-14 shrink-0 hidden md:flex items-center justify-end ml-2">
									<span class="text-xs font-mono font-medium {uptimeColor(uptimePercent(m))}">{formatPercent(uptimePercent(m))}%</span>
								</div>

								<!-- Value column -->
								<div class="w-20 shrink-0 hidden md:flex items-center justify-end ml-3">
									<span class="text-xs font-mono {infraValueClass(m)}">{infraValue(m)}</span>
								</div>

								<!-- Interval -->
								<div class="w-12 shrink-0 hidden md:flex items-center justify-end ml-2">
									<span class="text-[10px] font-mono text-muted-foreground/70">--</span>
								</div>

								<!-- Actions dropdown -->
								<div class="w-8 shrink-0 flex justify-end relative">
									<button
										onclick={(e) => { e.stopPropagation(); toggleDropdown(m.id); }}
										class="p-1 rounded hover:bg-muted/50 text-muted-foreground/40 hover:text-muted-foreground transition-colors"
										aria-label="Actions"
									>
										<MoreHorizontal class="w-4 h-4" />
									</button>

									{#if openDropdownId === m.id}
										<!-- svelte-ignore a11y_no_static_element_interactions -->
										<div
											class="absolute right-0 top-8 z-20 w-40 bg-card border border-border rounded-md shadow-lg py-1"
											onclick={(e) => e.stopPropagation()}
										>
											<a
												href="/monitors/{m.id}"
												class="flex items-center space-x-2 px-3 py-1.5 text-xs text-foreground hover:bg-muted/50 transition-colors"
											>
												<Eye class="w-3.5 h-3.5 text-muted-foreground" />
												<span>View Details</span>
											</a>
											{#if confirmDeleteId === m.id}
												<div class="px-3 py-1.5">
													<p class="text-[10px] text-red-400 mb-1.5">Delete this monitor?</p>
													<div class="flex items-center space-x-1.5">
														<button
															onclick={() => handleDelete(m.id)}
															disabled={deleting}
															class="px-2 py-1 text-[10px] bg-red-500/20 text-red-400 hover:bg-red-500/30 rounded transition-colors disabled:opacity-50"
														>
															{deleting ? 'Deleting...' : 'Confirm'}
														</button>
														<button
															onclick={cancelDelete}
															class="px-2 py-1 text-[10px] text-muted-foreground hover:text-foreground transition-colors"
														>
															Cancel
														</button>
													</div>
												</div>
											{:else}
												<button
													onclick={() => handleDelete(m.id)}
													class="flex items-center space-x-2 px-3 py-1.5 text-xs text-red-400 hover:bg-red-500/10 transition-colors w-full text-left"
												>
													<Trash2 class="w-3.5 h-3.5" />
													<span>Delete</span>
												</button>
											{/if}
										</div>
									{/if}
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/if}
		{/if}
	</div>

	<CreateMonitorModal
		bind:open={showCreateModal}
		agents={agentList}
		onClose={() => { showCreateModal = false; }}
		onCreated={handleMonitorCreated}
	/>
{/if}
