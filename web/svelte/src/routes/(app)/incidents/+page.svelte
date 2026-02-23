<script lang="ts">
	import { onMount } from 'svelte';
	import { AlertTriangle, CheckCircle2, ShieldAlert } from 'lucide-svelte';
	import { incidents as incidentsApi, monitors as monitorsApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast';
	import type { Incident, MonitorSummary, IncidentStatus } from '$lib/types';
	import IncidentStats from '$lib/components/incidents/IncidentStats.svelte';
	import IncidentRow from '$lib/components/incidents/IncidentRow.svelte';

	const toast = getToasts();

	type FilterTab = 'active' | 'resolved' | 'all';

	let allIncidents = $state<Incident[]>([]);
	let filteredIncidents = $state<Incident[]>([]);
	let monitorMap = $state<Map<string, MonitorSummary>>(new Map());
	let loading = $state(true);
	let activeTab = $state<FilterTab>('active');

	// Stats computed from all incidents (not just filtered)
	let openCount = $derived(allIncidents.filter((i) => i.status === 'open').length);
	let acknowledgedCount = $derived(allIncidents.filter((i) => i.status === 'acknowledged').length);
	let resolvedCount = $derived(allIncidents.filter((i) => i.status === 'resolved').length);

	// Tab counts from all incidents
	let activeCount = $derived(openCount + acknowledgedCount);

	const tabs: { value: FilterTab; label: string }[] = [
		{ value: 'active', label: 'Active' },
		{ value: 'resolved', label: 'Resolved' },
		{ value: 'all', label: 'All' }
	];

	function tabCount(tab: FilterTab): number {
		switch (tab) {
			case 'active':
				return activeCount;
			case 'resolved':
				return resolvedCount;
			case 'all':
				return allIncidents.length;
		}
	}

	function statusParam(tab: FilterTab): string | undefined {
		switch (tab) {
			case 'active':
				return 'active';
			case 'resolved':
				return 'resolved';
			case 'all':
				return undefined;
		}
	}

	function emptyMessage(tab: FilterTab): { title: string; subtitle: string } {
		switch (tab) {
			case 'active':
				return { title: 'No active incidents', subtitle: 'All systems operational.' };
			case 'resolved':
				return { title: 'No resolved incidents yet', subtitle: 'Resolved incidents will appear here.' };
			case 'all':
				return { title: 'No incidents recorded yet', subtitle: 'Incidents will appear here when monitors detect failures.' };
		}
	}

	function emptyIcon(tab: FilterTab) {
		switch (tab) {
			case 'active':
				return CheckCircle2;
			case 'resolved':
				return ShieldAlert;
			case 'all':
				return AlertTriangle;
		}
	}

	async function fetchIncidents(tab: FilterTab) {
		try {
			const res = await incidentsApi.listIncidents(statusParam(tab));
			filteredIncidents = res.data ?? [];
		} catch {
			toast.error('Failed to load incidents');
		}
	}

	async function loadData() {
		try {
			const [incidentsRes, allRes, summaryRes] = await Promise.all([
				incidentsApi.listIncidents(statusParam(activeTab)),
				incidentsApi.listIncidents(),
				monitorsApi.getMonitorsSummary()
			]);

			filteredIncidents = incidentsRes.data ?? [];
			allIncidents = allRes.data ?? [];

			const map = new Map<string, MonitorSummary>();
			for (const m of summaryRes ?? []) {
				map.set(m.id, m);
			}
			monitorMap = map;
		} catch {
			toast.error('Failed to load incidents');
		} finally {
			loading = false;
		}
	}

	async function handleTabChange(tab: FilterTab) {
		activeTab = tab;
		await fetchIncidents(tab);
	}

	async function handleAcknowledge(id: string): Promise<void> {
		try {
			await incidentsApi.acknowledgeIncident(id);
			toast.success('Incident acknowledged');

			// Update locally in both lists
			const now = new Date().toISOString();
			filteredIncidents = filteredIncidents.map((i) =>
				i.id === id ? { ...i, status: 'acknowledged' as IncidentStatus, acknowledged_at: now } : i
			);
			allIncidents = allIncidents.map((i) =>
				i.id === id ? { ...i, status: 'acknowledged' as IncidentStatus, acknowledged_at: now } : i
			);
		} catch {
			toast.error('Failed to acknowledge incident');
		}
	}

	async function handleResolve(id: string): Promise<void> {
		try {
			await incidentsApi.resolveIncident(id);
			toast.success('Incident resolved');

			// Update locally in both lists
			const now = new Date().toISOString();
			const incident = allIncidents.find((i) => i.id === id);
			const startedAt = incident ? new Date(incident.started_at).getTime() : Date.now();
			const ttr = Math.floor((Date.now() - startedAt) / 1000);

			filteredIncidents = filteredIncidents.map((i) =>
				i.id === id
					? { ...i, status: 'resolved' as IncidentStatus, resolved_at: now, ttr_seconds: ttr }
					: i
			);
			allIncidents = allIncidents.map((i) =>
				i.id === id
					? { ...i, status: 'resolved' as IncidentStatus, resolved_at: now, ttr_seconds: ttr }
					: i
			);

			// If on "active" tab, remove resolved incidents from filtered view
			if (activeTab === 'active') {
				filteredIncidents = filteredIncidents.filter((i) => i.status !== 'resolved');
			}
		} catch {
			toast.error('Failed to resolve incident');
		}
	}

	onMount(() => {
		loadData();
	});
</script>

<svelte:head>
	<title>Incidents - WatchDog</title>
</svelte:head>

{#if loading}
	<!-- Skeleton loading state -->
	<div class="animate-fade-in-up space-y-4">
		<!-- Header skeleton -->
		<div class="flex items-center justify-between">
			<div class="h-7 w-32 bg-muted/50 rounded animate-pulse"></div>
		</div>
		<!-- Stats pills skeleton -->
		<div class="flex items-center gap-3">
			{#each Array(3) as _}
				<div class="h-9 w-36 bg-muted/50 rounded-lg animate-pulse"></div>
			{/each}
		</div>
		<!-- Filter tabs skeleton -->
		<div class="flex items-center gap-1">
			{#each Array(3) as _}
				<div class="h-7 w-20 bg-muted/50 rounded-md animate-pulse"></div>
			{/each}
		</div>
		<!-- Table skeleton -->
		<div class="bg-card border border-border rounded-lg">
			{#each Array(5) as _}
				<div class="flex items-center px-4 py-4 border-b border-border/20">
					<div class="h-5 w-20 bg-muted/50 rounded animate-pulse mr-4"></div>
					<div class="flex-1 space-y-1.5">
						<div class="h-4 w-40 bg-muted/50 rounded animate-pulse"></div>
					</div>
					<div class="h-4 w-16 bg-muted/50 rounded animate-pulse hidden md:block ml-4"></div>
					<div class="h-4 w-16 bg-muted/50 rounded animate-pulse hidden md:block ml-4"></div>
					<div class="h-6 w-24 bg-muted/50 rounded animate-pulse ml-4"></div>
				</div>
			{/each}
		</div>
	</div>
{:else}
	<div class="animate-fade-in-up">
		<!-- Page header -->
		<div class="mb-5">
			<h1 class="text-lg font-semibold text-foreground">Incidents</h1>
			<p class="text-xs text-muted-foreground mt-0.5">
				{allIncidents.length} incident{allIncidents.length !== 1 ? 's' : ''} total
			</p>
		</div>

		<!-- Stats pills -->
		<div class="mb-4">
			<IncidentStats open={openCount} acknowledged={acknowledgedCount} resolved={resolvedCount} />
		</div>

		<!-- Filter tabs -->
		<div class="flex items-center gap-1 mb-4">
			{#each tabs as tab}
				<button
					onclick={() => handleTabChange(tab.value)}
					class="flex items-center space-x-1.5 px-2.5 py-1 text-xs rounded-md transition-colors {activeTab === tab.value
						? 'bg-foreground/[0.08] text-foreground font-medium'
						: 'text-muted-foreground hover:text-foreground hover:bg-foreground/[0.04]'}"
				>
					<span>{tab.label}</span>
					<span class="text-[10px] font-mono {activeTab === tab.value ? 'text-foreground/60' : 'text-muted-foreground/60'}">{tabCount(tab.value)}</span>
				</button>
			{/each}
		</div>

		<!-- Incidents table -->
		{#if filteredIncidents.length > 0}
			<div class="bg-card border border-border rounded-lg overflow-x-auto">
				<table class="w-full">
					<thead>
						<tr class="border-b border-border/30">
							<th class="px-4 py-2 text-left text-[9px] font-medium text-muted-foreground uppercase tracking-wider">Status</th>
							<th class="px-4 py-2 text-left text-[9px] font-medium text-muted-foreground uppercase tracking-wider">Monitor</th>
							<th class="px-4 py-2 text-left text-[9px] font-medium text-muted-foreground uppercase tracking-wider hidden lg:table-cell">Target</th>
							<th class="px-4 py-2 text-left text-[9px] font-medium text-muted-foreground uppercase tracking-wider hidden md:table-cell">Started</th>
							<th class="px-4 py-2 text-left text-[9px] font-medium text-muted-foreground uppercase tracking-wider hidden md:table-cell">
								{activeTab === 'resolved' ? 'TTR' : 'Duration'}
							</th>
							<th class="px-4 py-2 text-left text-[9px] font-medium text-muted-foreground uppercase tracking-wider">Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each filteredIncidents as incident (incident.id)}
							<IncidentRow
								{incident}
								monitor={monitorMap.get(incident.monitor_id)}
								onAcknowledge={handleAcknowledge}
								onResolve={handleResolve}
							/>
						{/each}
					</tbody>
				</table>
			</div>
		{:else}
			<!-- Empty state -->
			{@const msg = emptyMessage(activeTab)}
			{@const Icon = emptyIcon(activeTab)}
			<div class="bg-card border border-border rounded-lg">
				<div class="p-12 text-center">
					<div class="w-12 h-12 bg-muted/50 rounded-lg flex items-center justify-center mx-auto mb-4">
						<Icon class="w-6 h-6 text-muted-foreground/40" />
					</div>
					<p class="text-sm font-medium text-foreground mb-1">{msg.title}</p>
					<p class="text-xs text-muted-foreground">{msg.subtitle}</p>
				</div>
			</div>
		{/if}
	</div>
{/if}
