<script lang="ts">
	import { Clock, Timer, Activity, Server, Shield, Container, Database, Cpu } from 'lucide-svelte';
	import { formatPercent, uptimeColor } from '$lib/utils';
	import type { Monitor } from '$lib/types';

	interface Props {
		monitor: Monitor;
		uptimePercent: number;
		agentName: string;
	}

	let { monitor, uptimePercent, agentName }: Props = $props();

	let uptimeDisplay = $derived(formatPercent(uptimePercent));
	let uptimeColorClass = $derived(uptimeColor(uptimePercent));
</script>

<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
	<!-- Check Interval -->
	<div class="bg-card border border-border rounded-lg p-4">
		<div class="flex items-center space-x-1.5 mb-1">
			<Clock class="w-3 h-3 text-muted-foreground" />
			<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Interval</p>
		</div>
		<p class="text-lg font-semibold text-foreground font-mono">{monitor.interval_seconds}s</p>
	</div>

	<!-- Timeout -->
	<div class="bg-card border border-border rounded-lg p-4">
		<div class="flex items-center space-x-1.5 mb-1">
			<Timer class="w-3 h-3 text-muted-foreground" />
			<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Timeout</p>
		</div>
		<p class="text-lg font-semibold text-foreground font-mono">{monitor.timeout_seconds}s</p>
	</div>

	<!-- Uptime -->
	<div class="bg-card border border-border rounded-lg p-4">
		<div class="flex items-center space-x-1.5 mb-1">
			<Activity class="w-3 h-3 text-muted-foreground" />
			<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Uptime</p>
		</div>
		<p class="text-lg font-semibold font-mono {uptimeColorClass}">{uptimeDisplay}%</p>
	</div>

	<!-- Agent -->
	<div class="bg-card border border-border rounded-lg p-4">
		<div class="flex items-center space-x-1.5 mb-1">
			<Server class="w-3 h-3 text-muted-foreground" />
			<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Agent</p>
		</div>
		<p class="text-sm font-medium text-foreground truncate">{agentName}</p>
	</div>

	<!-- Type-specific extra cards -->
	{#if monitor.type === 'tls' && monitor.metadata?.cert_expiry_days}
		<div class="bg-card border border-border rounded-lg p-4">
			<div class="flex items-center space-x-1.5 mb-1">
				<Shield class="w-3 h-3 text-muted-foreground" />
				<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Cert Expiry</p>
			</div>
			<p class="text-lg font-semibold text-foreground font-mono">
				{monitor.metadata.cert_expiry_days}d
			</p>
		</div>
	{/if}

	{#if monitor.type === 'docker' && monitor.metadata?.container_name}
		<div class="bg-card border border-border rounded-lg p-4">
			<div class="flex items-center space-x-1.5 mb-1">
				<Container class="w-3 h-3 text-muted-foreground" />
				<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Container</p>
			</div>
			<p class="text-sm font-medium text-foreground truncate">{monitor.metadata.container_name}</p>
		</div>
	{/if}

	{#if monitor.type === 'database' && monitor.metadata?.db_type}
		<div class="bg-card border border-border rounded-lg p-4">
			<div class="flex items-center space-x-1.5 mb-1">
				<Database class="w-3 h-3 text-muted-foreground" />
				<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">DB Type</p>
			</div>
			<p class="text-sm font-medium text-foreground capitalize">{monitor.metadata.db_type}</p>
		</div>
	{/if}

	{#if monitor.type === 'system' && monitor.metadata?.metric_name}
		<div class="bg-card border border-border rounded-lg p-4">
			<div class="flex items-center space-x-1.5 mb-1">
				<Cpu class="w-3 h-3 text-muted-foreground" />
				<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Metric</p>
			</div>
			<p class="text-sm font-medium text-foreground capitalize">{monitor.metadata.metric_name}</p>
		</div>
	{/if}
</div>
