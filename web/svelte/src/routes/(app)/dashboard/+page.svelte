<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Globe, HardDrive } from 'lucide-svelte';
	import { monitors as monitorsApi, agents as agentsApi, incidents as incidentsApi } from '$lib/api';
	import { createSSE } from '$lib/stores/sse';
	import { isInfraMonitor } from '$lib/utils';
	import type { DashboardStats, MonitorSummary, Agent, Incident } from '$lib/types';
	import FleetBanner from '$lib/components/dashboard/FleetBanner.svelte';
	import StatsGrid from '$lib/components/dashboard/StatsGrid.svelte';
	import MonitorTable from '$lib/components/dashboard/MonitorTable.svelte';
	import AgentCard from '$lib/components/dashboard/AgentCard.svelte';
	import IncidentCard from '$lib/components/dashboard/IncidentCard.svelte';
	import NewAgentModal from '$lib/components/dashboard/NewAgentModal.svelte';

	let stats = $state<DashboardStats>({
		total_monitors: 0,
		monitors_up: 0,
		monitors_down: 0,
		active_incidents: 0,
		total_agents: 0,
		online_agents: 0
	});
	let summaries = $state<MonitorSummary[]>([]);
	let agentList = $state<Agent[]>([]);
	let incidentList = $state<Incident[]>([]);
	let loading = $state(true);
	let showAgentModal = $state(false);

	let services = $derived(summaries.filter((m) => !isInfraMonitor(m.type)));
	let infra = $derived(summaries.filter((m) => isInfraMonitor(m.type)));

	let uptimePercent = $derived(() => {
		const totalUp = summaries.reduce((acc, m) => acc + m.uptimeUp, 0);
		const totalChecks = summaries.reduce((acc, m) => acc + m.total, 0);
		if (totalChecks === 0) return 100;
		return (totalUp / totalChecks) * 100;
	});

	let monitorMap = $derived(() => {
		const map = new Map<string, MonitorSummary>();
		for (const m of summaries) {
			map.set(m.id, m);
		}
		return map;
	});

	async function loadData() {
		try {
			const [statsRes, summaryRes, agentsRes, incidentsRes] = await Promise.all([
				monitorsApi.getDashboardStats(),
				monitorsApi.getMonitorsSummary(),
				agentsApi.listAgents(),
				incidentsApi.listIncidents()
			]);
			stats = statsRes;
			summaries = summaryRes ?? [];
			agentList = agentsRes.data ?? [];
			incidentList = incidentsRes.data ?? [];
		} catch {
			// Keep defaults on error
		} finally {
			loading = false;
		}
	}

	function handleSSEEvent(event: string, data: unknown) {
		if (event === 'agent-status') {
			// Refresh agents + stats
			agentsApi.listAgents().then((res) => { agentList = res.data ?? []; });
			monitorsApi.getDashboardStats().then((res) => { stats = res; });
		} else if (event === 'incident-count') {
			// Refresh incidents + stats
			incidentsApi.listIncidents().then((res) => { incidentList = res.data ?? []; });
			monitorsApi.getDashboardStats().then((res) => { stats = res; });
		}
	}

	const sse = createSSE(handleSSEEvent);

	function handleAgentCreated() {
		// Refresh agents and stats after creation
		agentsApi.listAgents().then((res) => { agentList = res.data ?? []; });
		monitorsApi.getDashboardStats().then((res) => { stats = res; });
	}

	onMount(() => {
		loadData();
		sse.connect();
	});

	onDestroy(() => {
		sse.disconnect();
	});
</script>

<svelte:head>
	<title>Dashboard - WatchDog</title>
</svelte:head>

{#if loading}
	<div class="animate-fade-in-up space-y-4">
		<!-- Skeleton fleet banner -->
		<div class="bg-card border border-border rounded-lg h-16 animate-pulse"></div>
		<!-- Skeleton stats grid -->
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
			{#each Array(4) as _}
				<div class="bg-card border border-border rounded-lg h-24 animate-pulse"></div>
			{/each}
		</div>
	</div>
{:else}
	<div class="animate-fade-in-up">
		<FleetBanner {stats} uptimePercent={uptimePercent()} />
		<StatsGrid {stats} uptimePercent={uptimePercent()} />

		{#if services.length > 0}
			<MonitorTable monitors={services} title="Services" icon={Globe} variant="service" />
		{/if}

		{#if infra.length > 0}
			<MonitorTable monitors={infra} title="Infrastructure" icon={HardDrive} variant="infra" />
		{/if}

		{#if summaries.length === 0}
			<!-- Empty state for no monitors -->
			<div class="bg-card border border-border rounded-lg mb-4">
				<div class="px-4 py-3 border-b border-border">
					<h2 class="text-sm font-medium text-foreground">Monitor Health</h2>
				</div>
				<div class="p-8 text-center">
					<div class="w-10 h-10 bg-muted/50 rounded-lg flex items-center justify-center mx-auto mb-3">
						<svg class="w-5 h-5 text-muted-foreground/40" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
							<path d="M22 12h-4l-3 9L9 3l-3 9H2" />
						</svg>
					</div>
					{#if stats.total_agents > 0}
						<p class="text-sm text-foreground font-medium mb-1">You have {stats.total_agents} agent{stats.total_agents > 1 ? 's' : ''} ready</p>
						<p class="text-xs text-muted-foreground mb-3">Create a monitor to start tracking your services.</p>
					{:else}
						<p class="text-sm text-foreground font-medium mb-1">No monitors yet</p>
						<p class="text-xs text-muted-foreground mb-1">Deploy an agent first, then create monitors to track your services.</p>
					{/if}
				</div>
			</div>
		{/if}

		<!-- Bottom Grid: Agents + Incidents -->
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
			<AgentCard agents={agentList} {stats} onCreateAgent={() => { showAgentModal = true; }} />
			<IncidentCard incidents={incidentList} monitors={monitorMap()} />
		</div>
	</div>

	<NewAgentModal bind:open={showAgentModal} onClose={() => { showAgentModal = false; }} onCreated={handleAgentCreated} />
{/if}
