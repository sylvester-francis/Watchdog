<script lang="ts">
	import { onMount } from 'svelte';
	import { AlertTriangle, CheckCircle2, ShieldAlert } from 'lucide-svelte';
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
			<Skeleton emphasis="secondary" width="8rem" height="1.75rem" />
		</div>
		<!-- Stats pills skeleton -->
		<div class="flex items-center gap-3">
			{#each Array(3) as _}
				<Skeleton emphasis="secondary" width="9rem" height="2.25rem" />
			{/each}
		</div>
		<!-- Filter tabs skeleton -->
		<div class="flex items-center gap-1">
			{#each Array(3) as _}
				<Skeleton emphasis="secondary" width="5rem" height="1.75rem" />
			{/each}
		</div>
		<!-- Table skeleton -->
		<div class="bg-card border border-border rounded-lg">
			{#each Array(5) as _}
				<div class="flex items-center px-4 py-4 border-b border-border/20">
					<div class="mr-4">
						<Skeleton emphasis="secondary" width="5rem" height="1.25rem" />
					</div>
					<div class="flex-1 space-y-1.5">
						<Skeleton emphasis="secondary" width="10rem" height="1rem" />
					</div>
					<div class="hidden md:block ml-4">
						<Skeleton emphasis="secondary" width="4rem" height="1rem" />
					</div>
					<div class="hidden md:block ml-4">
						<Skeleton emphasis="secondary" width="4rem" height="1rem" />
					</div>
					<div class="ml-4">
						<Skeleton emphasis="secondary" width="6rem" height="1.5rem" />
					</div>
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

		<div class="mb-4">
			<Tabs
				options={tabs as Array<{ value: string; label: string }>}
				value={activeTab}
				variant="pill"
				onchange={(v) => handleTabChange(v as FilterTab)}
			/>
		</div>

		<!-- Incidents table -->
		{#if filteredIncidents.length > 0}
			<div class="bg-card border border-border rounded-lg overflow-x-auto">
				<table class="w-full">
					<thead>
						<tr class="border-b border-border">
							<th class="px-4 py-3 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Status</th>
							<th class="px-4 py-3 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Monitor</th>
							<th class="px-4 py-3 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden lg:table-cell">Target</th>
							<th class="px-4 py-3 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden md:table-cell">Started</th>
							<th class="px-4 py-3 text-left text-[10px] font-medium text-muted-foreground uppercase tracking-wider hidden md:table-cell">
								{activeTab === 'resolved' ? 'TTR' : 'Duration'}
							</th>
							<th class="px-4 py-3 text-right text-[10px] font-medium text-muted-foreground uppercase tracking-wider">Actions</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border/50">
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

	<InvestigationDrawer
		open={!!selectedIncidentId}
		loading={investigationLoading}
		{investigation}
		onClose={closeInvestigation}
	/>
{/if}
