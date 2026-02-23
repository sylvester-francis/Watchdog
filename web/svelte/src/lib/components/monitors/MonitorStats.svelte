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

<div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
	<!-- Check Interval -->
	<div class="bg-card border border-border rounded-lg p-4">
		<div class="flex items-center space-x-2 mb-2">
			<div class="w-7 h-7 bg-muted/50 rounded-md flex items-center justify-center">
				<Clock class="w-3.5 h-3.5 text-muted-foreground" />
			</div>
			<span class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Interval</span>
		</div>
		<p class="text-lg font-semibold text-foreground font-mono">{monitor.interval_seconds}s</p>
	</div>

	<!-- Timeout -->
	<div class="bg-card border border-border rounded-lg p-4">
		<div class="flex items-center space-x-2 mb-2">
			<div class="w-7 h-7 bg-muted/50 rounded-md flex items-center justify-center">
				<Timer class="w-3.5 h-3.5 text-muted-foreground" />
			</div>
			<span class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Timeout</span>
		</div>
		<p class="text-lg font-semibold text-foreground font-mono">{monitor.timeout_seconds}s</p>
	</div>

	<!-- Uptime -->
	<div class="bg-card border border-border rounded-lg p-4">
		<div class="flex items-center space-x-2 mb-2">
			<div class="w-7 h-7 bg-muted/50 rounded-md flex items-center justify-center">
				<Activity class="w-3.5 h-3.5 text-muted-foreground" />
			</div>
			<span class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Uptime</span>
		</div>
		<p class="text-lg font-semibold font-mono {uptimeColorClass}">{uptimeDisplay}%</p>
	</div>

	<!-- Agent -->
	<div class="bg-card border border-border rounded-lg p-4">
		<div class="flex items-center space-x-2 mb-2">
			<div class="w-7 h-7 bg-muted/50 rounded-md flex items-center justify-center">
				<Server class="w-3.5 h-3.5 text-muted-foreground" />
			</div>
			<span class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Agent</span>
		</div>
		<p class="text-sm font-medium text-foreground truncate">{agentName}</p>
	</div>

	<!-- Type-specific extra cards -->
	{#if monitor.type === 'tls' && monitor.metadata?.cert_expiry_days}
		<div class="bg-card border border-border rounded-lg p-4">
			<div class="flex items-center space-x-2 mb-2">
				<div class="w-7 h-7 bg-muted/50 rounded-md flex items-center justify-center">
					<Shield class="w-3.5 h-3.5 text-muted-foreground" />
				</div>
				<span class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Cert Expiry</span>
			</div>
			<p class="text-lg font-semibold text-foreground font-mono">
				{monitor.metadata.cert_expiry_days}d
			</p>
		</div>
	{/if}

	{#if monitor.type === 'docker' && monitor.metadata?.container_name}
		<div class="bg-card border border-border rounded-lg p-4">
			<div class="flex items-center space-x-2 mb-2">
				<div class="w-7 h-7 bg-muted/50 rounded-md flex items-center justify-center">
					<Container class="w-3.5 h-3.5 text-muted-foreground" />
				</div>
				<span class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Container</span>
			</div>
			<p class="text-sm font-medium text-foreground truncate">{monitor.metadata.container_name}</p>
		</div>
	{/if}

	{#if monitor.type === 'database' && monitor.metadata?.db_type}
		<div class="bg-card border border-border rounded-lg p-4">
			<div class="flex items-center space-x-2 mb-2">
				<div class="w-7 h-7 bg-muted/50 rounded-md flex items-center justify-center">
					<Database class="w-3.5 h-3.5 text-muted-foreground" />
				</div>
				<span class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">DB Type</span>
			</div>
			<p class="text-sm font-medium text-foreground capitalize">{monitor.metadata.db_type}</p>
		</div>
	{/if}

	{#if monitor.type === 'system' && monitor.metadata?.metric_name}
		<div class="bg-card border border-border rounded-lg p-4">
			<div class="flex items-center space-x-2 mb-2">
				<div class="w-7 h-7 bg-muted/50 rounded-md flex items-center justify-center">
					<Cpu class="w-3.5 h-3.5 text-muted-foreground" />
				</div>
				<span class="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">Metric</span>
			</div>
			<p class="text-sm font-medium text-foreground capitalize">{monitor.metadata.metric_name}</p>
		</div>
	{/if}
</div>
