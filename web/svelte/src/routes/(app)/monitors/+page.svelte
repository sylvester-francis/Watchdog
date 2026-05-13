<script lang="ts">
	import { onMount } from 'svelte';
	import { Plus, Search, MoreHorizontal } from 'lucide-svelte';
	import { monitors as monitorsApi, agents as agentsApi } from '$lib/api';
	import { formatPercent, uptimeColor, isInfraMonitor } from '$lib/utils';
	import type { MonitorSummary, Agent, MonitorType } from '$lib/types';
	import UptimeChecks from '$lib/components/dashboard/UptimeChecks.svelte';
	import { Button, Skeleton, Sparkline } from '@sylvester-francis/watchdog-ui';
	import CreateMonitorModal from '$lib/components/monitors/CreateMonitorModal/index.svelte';
	import ConfirmModal from '$lib/components/ConfirmModal.svelte';
	import { getToasts } from '$lib/stores/toast.svelte';

	const toast = getToasts();

	let summaries = $state<MonitorSummary[]>([]);
	let agentList = $state<Agent[]>([]);
	let loading = $state(true);
	let showCreateModal = $state(false);

	// Filters
	let activeFilter = $state<'all' | MonitorType>('all');
	let searchQuery = $state('');

	// Delete confirmation modal
	let deleting = $state(false);
	let confirmModal = $state<{
		open: boolean;
		monitorId: string;
		monitorName: string;
	}>({ open: false, monitorId: '', monitorName: '' });

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
		{ value: 'system', label: 'System' },
		{ value: 'service', label: 'Service' },
		{ value: 'port_scan', label: 'Port Scan' }
	];

	// Filtered monitors
	let filtered = $derived.by(() => {
		let result = summaries;
		if (activeFilter !== 'all') {
			result = result.filter((m) => m.type === activeFilter);
		}
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
		if (!m.latencies || m.latencies.length === 0) {
			return m.status === 'up' ? 'OK' : '--';
		}
		return m.latencies[m.latencies.length - 1] + 'ms';
	}

	function formatInterval(seconds: number): string {
		if (!seconds) return '--';
		if (seconds < 60) return `${seconds}s`;
		return `${Math.floor(seconds / 60)}m`;
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

	function parseMetricValue(msg: string | undefined): string | null {
		if (!msg) return null;
		const match = msg.match(/([\d.]+)%/);
		return match ? `${parseFloat(match[1]).toFixed(1)}%` : null;
	}

	function infraValue(m: MonitorSummary): string {
		if (m.type === 'docker') return m.status === 'up' ? 'Running' : 'Stopped';
		if (m.type === 'service') return m.status === 'up' ? 'Running' : 'Stopped';
		if (m.type === 'port_scan') {
			if (m.latest_value) return m.latest_value;
			return m.status === 'up' ? 'Clean' : 'Drift';
		}
		if (m.type === 'database' && m.latencies?.length > 0)
			return m.latencies[m.latencies.length - 1] + 'ms';
		if (m.type === 'system') {
			const val = parseMetricValue(m.latest_value);
			if (val) return val;
			return m.status === 'up' ? 'OK' : 'Error';
		}
		if (m.latencies?.length > 0) return m.latencies[m.latencies.length - 1] + 'ms';
		return m.status === 'up' ? 'OK' : 'No data';
	}

	function infraValueClass(m: MonitorSummary): string {
		if (m.type === 'docker' || m.type === 'service')
			return m.status === 'up' ? 'text-success' : 'text-destructive';
		if (m.type === 'port_scan' || m.type === 'system')
			return m.status === 'up' ? 'text-success' : 'text-destructive';
		return 'text-muted-foreground';
	}

	function statusPipClass(status: string): string {
		if (status === 'up') return 'bg-success';
		if (status === 'down') return 'bg-destructive';
		if (status === 'warn') return 'bg-warning';
		return 'bg-muted-foreground/50';
	}

	function toggleDropdown(id: string) {
		openDropdownId = openDropdownId === id ? null : id;
	}

	function handleDelete(id: string) {
		const monitor = summaries.find((m) => m.id === id);
		openDropdownId = null;
		confirmModal = { open: true, monitorId: id, monitorName: monitor?.name ?? 'this monitor' };
	}

	async function executeDelete() {
		deleting = true;
		try {
			await monitorsApi.deleteMonitor(confirmModal.monitorId);
			summaries = summaries.filter((m) => m.id !== confirmModal.monitorId);
			confirmModal = { open: false, monitorId: '', monitorName: '' };
			toast.success('Monitor deleted');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete monitor');
		} finally {
			deleting = false;
		}
	}

	function closeConfirmModal() {
		if (!deleting) confirmModal = { open: false, monitorId: '', monitorName: '' };
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

	async function handleMonitorCreated() {
		await loadData();
		toast.success('Monitor created');
	}

	function handleWindowClick() {
		if (openDropdownId) openDropdownId = null;
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
	<div class="animate-fade-in-up mx-auto max-w-[1080px] space-y-8 px-4 py-8 sm:px-6 sm:py-10">
		<div class="space-y-2">
			<Skeleton emphasis="tertiary" width="6rem" height="0.75rem" />
			<Skeleton emphasis="secondary" width="14rem" height="2rem" />
			<Skeleton emphasis="tertiary" width="10rem" height="0.875rem" />
		</div>
		<div class="flex flex-wrap items-center gap-2">
			{#each Array(6) as _}
				<Skeleton emphasis="tertiary" width="4rem" height="1.5rem" />
			{/each}
		</div>
		<div class="space-y-2">
			{#each Array(5) as _}
				<Skeleton emphasis="tertiary" width="100%" height="3rem" />
			{/each}
		</div>
	</div>
{:else}
	<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-6 sm:px-6 sm:py-10">
		<!-- Page header -->
		<header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between sm:gap-4">
			<div class="min-w-0">
				<div class="flex items-center gap-2 font-mono tabular-nums text-xs text-muted-foreground">
					<span class="uppercase tracking-wider">Monitors</span>
				</div>
				<h1 class="mt-1.5 text-xl font-medium text-foreground sm:text-2xl md:text-3xl">
					{summaries.length} monitor{summaries.length !== 1 ? 's' : ''} configured
				</h1>
			</div>
			<Button variant="primary" size="sm" onclick={() => { showCreateModal = true; }}>
				<span class="flex items-center gap-1.5">
					<Plus class="h-3.5 w-3.5" />
					<span>New Monitor</span>
				</span>
			</Button>
		</header>

		<!-- Filter bar -->
		<div class="mt-8 flex flex-col items-start gap-3 sm:flex-row sm:items-center">
			<!-- Type filter tabs -->
			<div class="flex flex-wrap items-center gap-1">
				{#each filterTabs as tab}
					<button
						onclick={() => { activeFilter = tab.value; }}
						class="px-2.5 py-1 text-xs transition-colors {activeFilter === tab.value
							? 'font-medium text-foreground'
							: 'text-muted-foreground hover:text-foreground'}"
					>
						{tab.label}
					</button>
				{/each}
			</div>

			<!-- Search input -->
			<div class="relative w-full sm:ml-auto sm:w-auto">
				<Search class="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground/50" />
				<input
					type="text"
					bind:value={searchQuery}
					placeholder="Search monitors..."
					class="w-full border border-border bg-background pl-8 pr-3 py-1.5 text-xs text-foreground placeholder:text-muted-foreground/50 focus:border-foreground/30 focus:outline-none focus:ring-0 sm:w-56"
				/>
			</div>
		</div>

		<div class="mt-6">
			{#if summaries.length === 0}
				<!-- Empty state: no monitors at all -->
				<section>
					<div class="border-b border-border pb-3">
						<h3 class="text-sm font-medium text-foreground">No monitors yet</h3>
					</div>
					<div class="flex flex-col items-start gap-3 pt-6 sm:flex-row sm:items-center sm:justify-between sm:gap-6">
						<p class="text-xs text-muted-foreground">Create a monitor to start tracking your services and infrastructure.</p>
						<Button variant="primary" size="sm" onclick={() => { showCreateModal = true; }}>
							<span class="flex items-center gap-1.5">
								<Plus class="h-3.5 w-3.5" />
								<span>Create Monitor</span>
							</span>
						</Button>
					</div>
				</section>
			{:else if filtered.length === 0}
				<!-- Empty state: filters returned nothing -->
				<section>
					<div class="border-b border-border pb-3">
						<h3 class="text-sm font-medium text-foreground">No matches</h3>
					</div>
					<div class="flex flex-col items-start gap-3 pt-4 sm:flex-row sm:items-center sm:justify-between sm:gap-6">
						<p class="text-xs text-muted-foreground">No monitors match the current filter. Try adjusting your search or filter.</p>
						<button
							onclick={() => { activeFilter = 'all'; searchQuery = ''; }}
							class="text-xs text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline"
						>
							Clear filters
						</button>
					</div>
				</section>
			{:else}
				<div class="space-y-10">
					<!-- Services section -->
					{#if services.length > 0}
						<section>
							<div class="flex items-baseline gap-2 border-b border-border pb-3">
								<h3 class="text-sm font-medium text-foreground">Services</h3>
								<span class="font-mono tabular-nums text-[11px] text-muted-foreground">{services.length}</span>
							</div>

							<!-- Column headers -->
							<div class="mt-3 hidden items-center pb-2 text-[9px] font-medium uppercase tracking-wider text-muted-foreground sm:flex">
								<div class="w-4 shrink-0"></div>
								<div class="ml-2 min-w-0 flex-1">Service</div>
								<div class="hidden w-36 shrink-0 text-center md:block">Uptime (24h)</div>
								<div class="ml-2 hidden w-14 shrink-0 text-right md:block">Uptime</div>
								<div class="ml-3 hidden w-16 shrink-0 text-right md:block">Latency</div>
								<div class="ml-2 hidden w-14 shrink-0 text-right md:block">Response</div>
								<div class="ml-2 hidden w-12 shrink-0 text-right md:block">Interval</div>
								<div class="w-8 shrink-0"></div>
							</div>

							<!-- Rows -->
							<div class="divide-y divide-border/40">
								{#each services as m (m.id)}
									<div class="group relative flex items-center py-3 transition-colors hover:bg-muted/30">
										<!-- Status pip -->
										<div class="flex w-4 shrink-0 justify-center">
											<span class="inline-block h-1.5 w-1.5 rounded-full {statusPipClass(m.status)}" aria-label="Status: {m.status}"></span>
										</div>

										<a href="/monitors/{m.id}" class="ml-2 min-w-0 flex-1">
											<div class="flex items-center gap-2">
												<span class="truncate text-sm text-foreground transition-colors group-hover:text-accent">{m.name}</span>
												<span class="font-mono tabular-nums text-[10px] uppercase tracking-wider text-muted-foreground">{m.type}</span>
											</div>
											<p class="mt-0.5 hidden truncate font-mono tabular-nums text-[10px] text-muted-foreground sm:block">{m.target}</p>
										</a>

										<div class="mx-2 hidden w-36 shrink-0 items-center justify-center md:flex">
											{#if m.total > 0}
												<UptimeChecks checkResults={checkResults(m)} />
											{:else}
												<span class="text-[9px] text-muted-foreground">No data</span>
											{/if}
										</div>

										<div class="ml-2 hidden w-14 shrink-0 items-center justify-end md:flex">
											<span class="font-mono tabular-nums text-xs {uptimeColor(uptimePercent(m))}">{formatPercent(uptimePercent(m))}%</span>
										</div>

										<div class="ml-3 hidden w-16 shrink-0 items-center justify-end md:flex">
											{#if m.latencies?.length > 0}
												<Sparkline data={m.latencies} color={sparkColor(m.status)} />
											{:else}
												<span class="text-[9px] text-muted-foreground">No data</span>
											{/if}
										</div>

										<div class="ml-2 hidden w-14 shrink-0 items-center justify-end md:flex">
											<span class="font-mono tabular-nums text-xs text-muted-foreground">{lastLatency(m)}</span>
										</div>

										<div class="ml-2 hidden w-12 shrink-0 items-center justify-end md:flex">
											<span class="font-mono tabular-nums text-[10px] text-muted-foreground/70">{formatInterval(m.interval_seconds)}</span>
										</div>

										<!-- Actions dropdown -->
										<div class="relative flex w-8 shrink-0 justify-end">
											<button
												onclick={(e) => { e.stopPropagation(); toggleDropdown(m.id); }}
												class="p-1 text-muted-foreground/40 transition-colors hover:text-foreground"
												aria-label="Actions"
											>
												<MoreHorizontal class="h-4 w-4" />
											</button>

											{#if openDropdownId === m.id}
												<div
													class="absolute right-0 top-8 z-20 w-40 border border-border bg-background py-1 shadow-lg"
													onclick={(e) => e.stopPropagation()}
													onkeydown={(e) => e.stopPropagation()}
													role="menu"
													tabindex="-1"
												>
													<a
														href="/monitors/{m.id}"
														class="block px-3 py-1.5 text-xs text-foreground transition-colors hover:bg-muted/40"
													>
														View Details
													</a>
													<button
														onclick={() => handleDelete(m.id)}
														class="block w-full px-3 py-1.5 text-left text-xs text-destructive transition-colors hover:bg-destructive/10"
													>
														Delete
													</button>
												</div>
											{/if}
										</div>
									</div>
								{/each}
							</div>
						</section>
					{/if}

					<!-- Infrastructure section -->
					{#if infra.length > 0}
						<section>
							<div class="flex items-baseline gap-2 border-b border-border pb-3">
								<h3 class="text-sm font-medium text-foreground">Infrastructure</h3>
								<span class="font-mono tabular-nums text-[11px] text-muted-foreground">{infra.length}</span>
							</div>

							<!-- Column headers -->
							<div class="mt-3 hidden items-center pb-2 text-[9px] font-medium uppercase tracking-wider text-muted-foreground sm:flex">
								<div class="w-4 shrink-0"></div>
								<div class="ml-2 min-w-0 flex-1">Service</div>
								<div class="hidden w-36 shrink-0 text-center md:block">Health (24h)</div>
								<div class="ml-2 hidden w-14 shrink-0 text-right md:block">Uptime</div>
								<div class="ml-3 hidden w-20 shrink-0 text-right md:block">Value</div>
								<div class="ml-2 hidden w-12 shrink-0 text-right md:block">Interval</div>
								<div class="w-8 shrink-0"></div>
							</div>

							<!-- Rows -->
							<div class="divide-y divide-border/40">
								{#each infra as m (m.id)}
									<div class="group relative flex items-center py-3 transition-colors hover:bg-muted/30">
										<div class="flex w-4 shrink-0 justify-center">
											<span class="inline-block h-1.5 w-1.5 rounded-full {statusPipClass(m.status)}" aria-label="Status: {m.status}"></span>
										</div>

										<a href="/monitors/{m.id}" class="ml-2 min-w-0 flex-1">
											<div class="flex items-center gap-2">
												<span class="truncate text-sm text-foreground transition-colors group-hover:text-accent">{m.name}</span>
												<span class="font-mono tabular-nums text-[10px] uppercase tracking-wider text-muted-foreground">{m.type}</span>
											</div>
											<p class="mt-0.5 hidden truncate font-mono tabular-nums text-[10px] text-muted-foreground sm:block">{m.target}</p>
										</a>

										<div class="mx-2 hidden w-36 shrink-0 items-center justify-center md:flex">
											{#if m.total > 0}
												<UptimeChecks checkResults={checkResults(m)} />
											{:else}
												<span class="text-[9px] text-muted-foreground">No data</span>
											{/if}
										</div>

										<div class="ml-2 hidden w-14 shrink-0 items-center justify-end md:flex">
											<span class="font-mono tabular-nums text-xs {uptimeColor(uptimePercent(m))}">{formatPercent(uptimePercent(m))}%</span>
										</div>

										<div class="ml-3 hidden w-20 shrink-0 items-center justify-end md:flex">
											<span class="font-mono tabular-nums text-xs {infraValueClass(m)}">{infraValue(m)}</span>
										</div>

										<div class="ml-2 hidden w-12 shrink-0 items-center justify-end md:flex">
											<span class="font-mono tabular-nums text-[10px] text-muted-foreground/70">{formatInterval(m.interval_seconds)}</span>
										</div>

										<div class="relative flex w-8 shrink-0 justify-end">
											<button
												onclick={(e) => { e.stopPropagation(); toggleDropdown(m.id); }}
												class="p-1 text-muted-foreground/40 transition-colors hover:text-foreground"
												aria-label="Actions"
											>
												<MoreHorizontal class="h-4 w-4" />
											</button>

											{#if openDropdownId === m.id}
												<div
													class="absolute right-0 top-8 z-20 w-40 border border-border bg-background py-1 shadow-lg"
													onclick={(e) => e.stopPropagation()}
													onkeydown={(e) => e.stopPropagation()}
													role="menu"
													tabindex="-1"
												>
													<a
														href="/monitors/{m.id}"
														class="block px-3 py-1.5 text-xs text-foreground transition-colors hover:bg-muted/40"
													>
														View Details
													</a>
													<button
														onclick={() => handleDelete(m.id)}
														class="block w-full px-3 py-1.5 text-left text-xs text-destructive transition-colors hover:bg-destructive/10"
													>
														Delete
													</button>
												</div>
											{/if}
										</div>
									</div>
								{/each}
							</div>
						</section>
					{/if}
				</div>
			{/if}
		</div>
	</div>

	<CreateMonitorModal
		bind:open={showCreateModal}
		agents={agentList}
		onClose={() => { showCreateModal = false; }}
		onCreated={handleMonitorCreated}
	/>

	<ConfirmModal
		open={confirmModal.open}
		title="Delete Monitor"
		message="Are you sure you want to delete &quot;{confirmModal.monitorName}&quot;? All heartbeat data and incident history will be permanently removed."
		confirmLabel="Delete"
		variant="danger"
		loading={deleting}
		onConfirm={executeDelete}
		onCancel={closeConfirmModal}
	/>
{/if}
