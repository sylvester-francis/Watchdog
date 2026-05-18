<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { ChevronRight } from 'lucide-svelte';
	import { Skeleton } from '@sylvester-francis/watchdog-ui';
	import { monitors as monitorsApi, agents as agentsApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast.svelte';
	import type { Monitor, Agent } from '$lib/types';
	import MonitorHeader from '$lib/components/monitors/MonitorHeader.svelte';
	import MonitorStats from '$lib/components/monitors/MonitorStats.svelte';
	import LatencyChart from '$lib/components/monitors/LatencyChart.svelte';
	import LatencyTrend from '$lib/components/monitors/LatencyTrend.svelte';
	import MetricChart from '$lib/components/monitors/MetricChart.svelte';
	import StatusChart from '$lib/components/monitors/StatusChart.svelte';
	import CertExpiryChart from '$lib/components/monitors/CertExpiryChart.svelte';
	import PortScanResults from '$lib/components/monitors/PortScanResults.svelte';
	import SNMPResults from '$lib/components/monitors/SNMPResults.svelte';
	import DeviceHealthCard from '$lib/components/monitors/DeviceHealthCard.svelte';
	import SNMPValueChart from '$lib/components/monitors/SNMPValueChart.svelte';
	import RecentChecks from '$lib/components/monitors/RecentChecks.svelte';
	import DangerZone from '$lib/components/monitors/DangerZone.svelte';
	import CertDetailsCard from '$lib/components/monitors/CertDetailsCard.svelte';
	import SLACard from '$lib/components/monitors/SLACard.svelte';
	import EditMonitorModal from '$lib/components/monitors/EditMonitorModal.svelte';

	const toast = getToasts();

	let monitor = $state<Monitor | null>(null);
	let agents = $state<Agent[]>([]);
	let agentName = $state('Unknown');
	let uptimePercent = $state(0);
	let uptimeUp = $state(0);
	let uptimeDown = $state(0);
	let loading = $state(true);
	let error = $state('');
	let editOpen = $state(false);

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

			// Store agents and find matching agent name
			agents = agentsRes.data ?? [];
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
	<div class="animate-fade-in-up mx-auto max-w-[1080px] space-y-8 px-4 py-8 sm:px-6 sm:py-10">
		<Skeleton emphasis="secondary" width="12rem" height="1rem" />

		<div class="space-y-2">
			<Skeleton emphasis="tertiary" width="9rem" height="0.75rem" />
			<Skeleton emphasis="secondary" width="16rem" height="2rem" />
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

		<Skeleton variant="chart" emphasis="secondary" height="16rem" />
	</div>
{:else if error && !monitor}
	<!-- Error state -->
	<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-6 sm:px-6 sm:py-10">
		<p class="text-sm font-medium text-foreground">Failed to load monitor</p>
		<p class="mt-1 font-mono tabular-nums text-xs text-destructive">{error}</p>
		<a
			href="/monitors"
			class="mt-4 inline-block text-sm text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline"
		>
			← Back to Monitors
		</a>
	</div>
{:else if monitor}
	<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-6 sm:px-6 sm:py-10">
		<!-- Breadcrumb -->
		<nav class="flex items-center gap-1.5 text-xs">
			<a href="/monitors" class="text-muted-foreground transition-colors hover:text-foreground">
				Monitors
			</a>
			<ChevronRight class="h-3 w-3 text-muted-foreground/40" />
			<span class="truncate max-w-[200px] font-mono tabular-nums text-foreground">{monitor.name}</span>
		</nav>

		<!-- Header -->
		<div class="mt-6">
			<MonitorHeader {monitor} onEdit={() => editOpen = true} />
		</div>

		<!-- Stats Row (hairline-separated columns) -->
		<div class="mt-8">
			<MonitorStats {monitor} {uptimePercent} {agentName} />
		</div>

		<!-- Charts and sections -->
		<div class="mt-10 space-y-10">
			{#if monitor.type === 'tls'}
				<CertExpiryChart {monitorId} />
				<CertDetailsCard {monitorId} />
				<LatencyChart {monitorId} />
				<StatusChart {monitorId} />
			{:else if monitor.type === 'system'}
				<MetricChart {monitorId} target={monitor.target} />
				<StatusChart {monitorId} />
			{:else if monitor.type === 'port_scan'}
				<PortScanResults metadata={monitor.metadata} />
				<StatusChart {monitorId} />
			{:else if monitor.type === 'snmp'}
				{#if monitor.metadata?.template_id}
					<DeviceHealthCard metadata={monitor.metadata} templateId={monitor.metadata.template_id} />
				{/if}
				<SNMPResults metadata={monitor.metadata} />
				<SNMPValueChart {monitorId} />
				<LatencyChart {monitorId} />
				<StatusChart {monitorId} />
			{:else if monitor.type === 'service' || monitor.type === 'docker'}
				<StatusChart {monitorId} />
			{:else}
				<LatencyChart {monitorId} />
				<StatusChart {monitorId} />
			{/if}

			{#if monitor.metadata?.sla_target_percent || monitor.sla_target_percent}
				<SLACard {monitorId} />
			{/if}

			<!-- Uptime Bar (hidden for port_scan — redundant with StatusChart) -->
			{#if monitor.type !== 'port_scan' && (uptimeUp > 0 || uptimeDown > 0)}
				<section>
					<div class="border-b border-border pb-3">
						<h3 class="text-sm font-medium text-foreground">Uptime (Last 20 checks)</h3>
					</div>
					<div class="pt-4">
						<div class="mb-3 flex items-center gap-4 font-mono tabular-nums text-[10px] text-muted-foreground">
							<div class="flex items-center gap-1.5">
								<span class="inline-block h-1.5 w-1.5 rounded-full bg-success"></span>
								{uptimeUp} success
							</div>
							<div class="flex items-center gap-1.5">
								<span class="inline-block h-1.5 w-1.5 rounded-full bg-destructive"></span>
								{uptimeDown} failed
							</div>
						</div>
						<div class="flex h-2 w-full gap-px overflow-hidden bg-muted">
							{#if uptimeUp > 0}
								<div
									class="h-full bg-success"
									style="width: {(uptimeUp / (uptimeUp + uptimeDown)) * 100}%"
								></div>
							{/if}
							{#if uptimeDown > 0}
								<div
									class="h-full bg-destructive"
									style="width: {(uptimeDown / (uptimeUp + uptimeDown)) * 100}%"
								></div>
							{/if}
						</div>
					</div>
				</section>
			{/if}

			<RecentChecks {monitorId} monitorType={monitor.type} />

			<LatencyTrend {monitorId} />

			<DangerZone {monitorId} />
		</div>
	</div>

	<!-- Edit Modal -->
	<EditMonitorModal
		bind:open={editOpen}
		{monitor}
		{agents}
		onClose={() => editOpen = false}
		onUpdated={() => { editOpen = false; loadData(); toast.success('Monitor updated'); }}
	/>
{/if}
