<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { base } from '$app/paths';
	import { ChevronRight } from 'lucide-svelte';
	import { monitors as monitorsApi, agents as agentsApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast';
	import type { Monitor, Agent } from '$lib/types';
	import MonitorHeader from '$lib/components/monitors/MonitorHeader.svelte';
	import MonitorStats from '$lib/components/monitors/MonitorStats.svelte';
	import LatencyChart from '$lib/components/monitors/LatencyChart.svelte';
	import RecentChecks from '$lib/components/monitors/RecentChecks.svelte';
	import DangerZone from '$lib/components/monitors/DangerZone.svelte';

	const toast = getToasts();

	let monitor = $state<Monitor | null>(null);
	let agentName = $state('Unknown');
	let uptimePercent = $state(0);
	let uptimeUp = $state(0);
	let uptimeDown = $state(0);
	let loading = $state(true);
	let error = $state('');

	let monitorId = $derived(page.params.id ?? '');

	async function loadData() {
		loading = true;
		error = '';

		try {
			const [monitorRes, agentsRes] = await Promise.all([
				monitorsApi.getMonitor(monitorId),
				agentsApi.listAgents()
			]);

			monitor = monitorRes.data;
			uptimeUp = monitorRes.heartbeats?.uptime_up ?? 0;
			uptimeDown = monitorRes.heartbeats?.uptime_down ?? 0;
			const total = monitorRes.heartbeats?.total ?? 0;
			uptimePercent = total > 0 ? (uptimeUp / total) * 100 : 0;

			// Find matching agent name
			const agents: Agent[] = agentsRes.data ?? [];
			const matchedAgent = agents.find((a) => a.id === monitor?.agent_id);
			agentName = matchedAgent?.name ?? 'Unknown';
		} catch (err) {
			const msg = err instanceof Error ? err.message : 'Failed to load monitor';
			error = msg;
			toast.error(msg);
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		loadData();
	});
</script>

<svelte:head>
	<title>{monitor?.name ?? 'Monitor'} - WatchDog</title>
</svelte:head>

{#if loading}
	<!-- Skeleton loading state -->
	<div class="animate-fade-in-up space-y-4">
		<!-- Breadcrumb skeleton -->
		<div class="h-4 w-48 bg-muted/50 rounded animate-pulse"></div>

		<!-- Header skeleton -->
		<div class="flex items-center space-x-3">
			<div class="w-3 h-3 bg-muted/50 rounded-full animate-pulse"></div>
			<div class="space-y-2">
				<div class="h-6 w-56 bg-muted/50 rounded animate-pulse"></div>
				<div class="h-3 w-72 bg-muted/30 rounded animate-pulse"></div>
			</div>
		</div>

		<!-- Stats grid skeleton -->
		<div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
			{#each Array(4) as _}
				<div class="bg-card border border-border rounded-lg h-24 animate-pulse"></div>
			{/each}
		</div>

		<!-- Chart skeleton -->
		<div class="bg-card border border-border rounded-lg h-64 animate-pulse"></div>

		<!-- Table skeleton -->
		<div class="bg-card border border-border rounded-lg">
			{#each Array(5) as _}
				<div class="flex items-center px-4 py-3 border-b border-border/20">
					<div class="h-3 w-20 bg-muted/50 rounded animate-pulse"></div>
					<div class="h-3 w-12 bg-muted/50 rounded animate-pulse ml-auto"></div>
					<div class="h-3 w-14 bg-muted/50 rounded animate-pulse ml-4"></div>
				</div>
			{/each}
		</div>
	</div>
{:else if error && !monitor}
	<!-- Error state -->
	<div class="animate-fade-in-up">
		<div class="bg-card border border-border rounded-lg p-8 text-center">
			<p class="text-sm text-foreground font-medium mb-1">Failed to load monitor</p>
			<p class="text-xs text-muted-foreground mb-4">{error}</p>
			<a
				href="{base}/monitors"
				class="inline-flex items-center px-4 py-2 bg-accent text-white hover:bg-accent/90 text-xs font-medium rounded-md transition-colors"
			>
				Back to Monitors
			</a>
		</div>
	</div>
{:else if monitor}
	<div class="animate-fade-in-up space-y-5">
		<!-- Breadcrumb -->
		<nav class="flex items-center space-x-1.5 text-xs">
			<a href="{base}/monitors" class="text-muted-foreground hover:text-foreground transition-colors">
				Monitors
			</a>
			<ChevronRight class="w-3 h-3 text-muted-foreground/50" />
			<span class="text-foreground font-medium truncate max-w-[200px]">{monitor.name}</span>
		</nav>

		<!-- Header -->
		<MonitorHeader {monitor} />

		<!-- Stats Cards -->
		<MonitorStats {monitor} {uptimePercent} {agentName} />

		<!-- Latency Chart -->
		<LatencyChart {monitorId} />

		<!-- Uptime Bar -->
		{#if uptimeUp > 0 || uptimeDown > 0}
			<div class="bg-card border border-border rounded-lg p-4">
				<h2 class="text-sm font-medium text-foreground mb-3">Uptime Summary</h2>
				<div class="w-full h-3 rounded-full overflow-hidden bg-muted/30 flex">
					{#if uptimeUp > 0}
						<div
							class="h-full bg-emerald-400 transition-all"
							style="width: {(uptimeUp / (uptimeUp + uptimeDown)) * 100}%"
						></div>
					{/if}
					{#if uptimeDown > 0}
						<div
							class="h-full bg-red-400 transition-all"
							style="width: {(uptimeDown / (uptimeUp + uptimeDown)) * 100}%"
						></div>
					{/if}
				</div>
				<p class="text-[10px] text-muted-foreground mt-2 font-mono">
					{uptimeUp} successful / {uptimeDown} failed
				</p>
			</div>
		{/if}

		<!-- Recent Checks -->
		<RecentChecks {monitorId} />

		<!-- Danger Zone -->
		<DangerZone {monitorId} />
	</div>
{/if}
