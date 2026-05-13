<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Button, Skeleton } from '@sylvester-francis/watchdog-ui';
	import { goto } from '$app/navigation';
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
			agentsApi.listAgents().then((res) => { agentList = res.data ?? []; });
			monitorsApi.getDashboardStats().then((res) => { stats = res; });
		} else if (event === 'incident-count') {
			incidentsApi.listIncidents().then((res) => { incidentList = res.data ?? []; });
			monitorsApi.getDashboardStats().then((res) => { stats = res; });
		}
	}

	const sse = createSSE(handleSSEEvent);

	function handleAgentCreated() {
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
	<div class="animate-fade-in-up mx-auto max-w-[1080px] space-y-8 px-4 py-8 sm:px-6 sm:py-10">
		<div class="space-y-2">
			<Skeleton emphasis="tertiary" width="3rem" height="0.75rem" />
			<Skeleton emphasis="secondary" width="22rem" height="2rem" />
			<Skeleton emphasis="tertiary" width="18rem" height="0.875rem" />
		</div>
		<div class="grid grid-cols-2 gap-px border-y border-border bg-border sm:grid-cols-4">
			{#each Array(4) as _}
				<div class="bg-background p-4">
					<Skeleton emphasis="tertiary" width="3.5rem" height="0.625rem" />
					<div class="mt-2">
						<Skeleton emphasis="secondary" width="4rem" height="1.25rem" />
					</div>
				</div>
			{/each}
		</div>
	</div>
{:else}
	<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-6 sm:px-6 sm:py-10">
		<FleetBanner {stats} uptimePercent={uptimePercent()} />

		<div class="mt-8">
			<StatsGrid {stats} uptimePercent={uptimePercent()} />
		</div>

		<div class="mt-10 space-y-10">
			{#if services.length > 0}
				<MonitorTable monitors={services} title="Services" variant="service" />
			{/if}

			{#if infra.length > 0}
				<MonitorTable monitors={infra} title="Infrastructure" variant="infra" />
			{/if}

			{#if summaries.length === 0}
				<!-- Empty state -->
				<section>
					<div class="border-b border-border pb-3">
						<h3 class="text-sm font-medium text-foreground">Monitor Health</h3>
					</div>
					<div class="flex flex-col items-start gap-3 pt-6 sm:flex-row sm:items-center sm:justify-between sm:gap-6">
						<div>
							{#if stats.total_agents > 0}
								<p class="text-sm font-medium text-foreground">
									You have {stats.total_agents} agent{stats.total_agents > 1 ? 's' : ''} ready
								</p>
								<p class="mt-1 text-xs text-muted-foreground">Create a monitor to start tracking your services.</p>
							{:else}
								<p class="text-sm font-medium text-foreground">No monitors yet</p>
								<p class="mt-1 text-xs text-muted-foreground">Deploy an agent first, then create monitors to track your services.</p>
							{/if}
						</div>
						{#if stats.total_agents > 0}
							<Button variant="primary" size="sm" onclick={() => goto('/monitors')}>Create Monitor</Button>
						{:else}
							<Button variant="secondary" size="sm" onclick={() => { showAgentModal = true; }}>Deploy Agent</Button>
						{/if}
					</div>
				</section>
			{/if}

			<!-- Bottom Grid: Agents + Incidents -->
			<div class="grid grid-cols-1 gap-10 lg:grid-cols-2">
				<AgentCard agents={agentList} {stats} onCreateAgent={() => { showAgentModal = true; }} />
				<IncidentCard incidents={incidentList} monitors={monitorMap()} />
			</div>
		</div>
	</div>

	<NewAgentModal bind:open={showAgentModal} onClose={() => { showAgentModal = false; }} onCreated={handleAgentCreated} />
{/if}
