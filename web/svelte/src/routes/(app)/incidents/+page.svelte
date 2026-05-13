<script lang="ts">
	import { onMount } from 'svelte';
	import { Skeleton, Tabs } from '@sylvester-francis/watchdog-ui';
	import { incidents as incidentsApi, monitors as monitorsApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast.svelte';
	import type { Incident, MonitorSummary, IncidentStatus, IncidentInvestigation } from '$lib/types';
	import IncidentStats from '$lib/components/incidents/IncidentStats.svelte';
	import IncidentRow from '$lib/components/incidents/IncidentRow.svelte';
	import InvestigationDrawer from '$lib/components/incidents/InvestigationDrawer.svelte';

	const toast = getToasts();

	type FilterTab = 'active' | 'resolved' | 'all';

	let allIncidents = $state<Incident[]>([]);
	let filteredIncidents = $state<Incident[]>([]);
	let monitorMap = $state<Map<string, MonitorSummary>>(new Map());
	let loading = $state(true);
	let activeTab = $state<FilterTab>('active');

	// Investigation panel state
	let selectedIncidentId = $state<string | null>(null);
	let investigation = $state<IncidentInvestigation | null>(null);
	let investigationLoading = $state(false);

	async function handleInvestigate(id: string) {
		selectedIncidentId = id;
		investigationLoading = true;
		investigation = null;
		try {
			const res = await incidentsApi.getIncidentInvestigation(id);
			investigation = res.data;
		} catch {
			toast.error('Failed to load investigation');
			selectedIncidentId = null;
		} finally {
			investigationLoading = false;
		}
	}

	function closeInvestigation() {
		selectedIncidentId = null;
		investigation = null;
	}

	// Stats computed from all incidents (not just filtered)
	let openCount = $derived(allIncidents.filter((i) => i.status === 'open').length);
	let acknowledgedCount = $derived(allIncidents.filter((i) => i.status === 'acknowledged').length);
	let resolvedCount = $derived(allIncidents.filter((i) => i.status === 'resolved').length);

	// Tab counts from all incidents
	let activeCount = $derived(openCount + acknowledgedCount);

	const tabs = $derived([
		{ value: 'active' as const, label: `Active ${activeCount}` },
		{ value: 'resolved' as const, label: `Resolved ${resolvedCount}` },
		{ value: 'all' as const, label: `All ${allIncidents.length}` }
	]);

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
	<div class="animate-fade-in-up mx-auto max-w-[1080px] space-y-8 px-4 py-8 sm:px-6 sm:py-10">
		<div class="space-y-2">
			<Skeleton emphasis="tertiary" width="6rem" height="0.75rem" />
			<Skeleton emphasis="secondary" width="14rem" height="2rem" />
			<Skeleton emphasis="tertiary" width="10rem" height="0.875rem" />
		</div>
		<div class="grid grid-cols-1 gap-px border-y border-border bg-border sm:grid-cols-3">
			{#each Array(3) as _}
				<div class="bg-background p-4">
					<Skeleton emphasis="tertiary" width="4rem" height="0.625rem" />
					<div class="mt-2">
						<Skeleton emphasis="secondary" width="3rem" height="1.25rem" />
					</div>
				</div>
			{/each}
		</div>
		<div class="space-y-2">
			{#each Array(5) as _}
				<Skeleton emphasis="tertiary" width="100%" height="2.5rem" />
			{/each}
		</div>
	</div>
{:else}
	<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-6 sm:px-6 sm:py-10">
		<!-- Page header -->
		<header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between sm:gap-4">
			<div class="min-w-0">
				<div class="flex items-center gap-2 font-mono tabular-nums text-xs text-muted-foreground">
					<span class="uppercase tracking-wider">Incidents</span>
				</div>
				<h1 class="mt-1.5 text-xl font-medium text-foreground sm:text-2xl md:text-3xl">
					{allIncidents.length} incident{allIncidents.length !== 1 ? 's' : ''} total
				</h1>
			</div>
		</header>

		<!-- Stats -->
		<div class="mt-8">
			<IncidentStats open={openCount} acknowledged={acknowledgedCount} resolved={resolvedCount} />
		</div>

		<!-- Tabs -->
		<div class="mt-8">
			<Tabs
				options={tabs as Array<{ value: string; label: string }>}
				value={activeTab}
				variant="pill"
				onchange={(v) => handleTabChange(v as FilterTab)}
			/>
		</div>

		<!-- Incidents table -->
		<div class="mt-6">
			{#if filteredIncidents.length > 0}
				<div class="overflow-x-auto">
					<table class="w-full">
						<thead>
							<tr class="border-b border-border">
								<th class="py-2.5 pl-1 pr-4 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Status</th>
								<th class="px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Monitor</th>
								<th class="hidden px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground lg:table-cell">Target</th>
								<th class="hidden px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground md:table-cell">Started</th>
								<th class="hidden px-4 py-2.5 text-left text-[10px] font-medium uppercase tracking-wider text-muted-foreground md:table-cell">
									{activeTab === 'resolved' ? 'TTR' : 'Duration'}
								</th>
								<th class="px-1 py-2.5 pr-1 text-right text-[10px] font-medium uppercase tracking-wider text-muted-foreground sm:px-4">Actions</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border/40">
							{#each filteredIncidents as incident (incident.id)}
								<IncidentRow
									{incident}
									monitor={monitorMap.get(incident.monitor_id)}
									onAcknowledge={handleAcknowledge}
									onResolve={handleResolve}
									onInvestigate={handleInvestigate}
								/>
							{/each}
						</tbody>
					</table>
				</div>
			{:else}
				<!-- Empty state -->
				{@const msg = emptyMessage(activeTab)}
				<section>
					<div class="border-b border-border pb-3">
						<h3 class="text-sm font-medium text-foreground">{msg.title}</h3>
					</div>
					<p class="pt-4 text-xs text-muted-foreground">{msg.subtitle}</p>
				</section>
			{/if}
		</div>
	</div>

	<InvestigationDrawer
		open={!!selectedIncidentId}
		loading={investigationLoading}
		{investigation}
		onClose={closeInvestigation}
	/>
{/if}
