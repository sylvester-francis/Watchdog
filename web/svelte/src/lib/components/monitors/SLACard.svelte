<script lang="ts">
	import { Target, TrendingUp, TrendingDown } from 'lucide-svelte';
	import { getMonitorSLA } from '$lib/api/monitors';
	import type { SLAResponse } from '$lib/types';
	import { onMount } from 'svelte';

	interface Props {
		monitorId: string;
	}

	let { monitorId }: Props = $props();
	let sla = $state<SLAResponse | null>(null);
	let loading = $state(true);

	onMount(async () => {
		try {
			const res = await getMonitorSLA(monitorId, '30d');
			sla = res.data;
		} catch {
			// silently fail — monitor may not have heartbeats yet
		} finally {
			loading = false;
		}
	});

	let statusColor = $derived(() => {
		if (!sla) return 'text-muted-foreground';
		return sla.breached ? 'text-destructive' : 'text-green-500';
	});

	let marginColor = $derived(() => {
		if (!sla) return 'text-muted-foreground';
		return sla.margin >= 0 ? 'text-green-500' : 'text-destructive';
	});
</script>

{#if loading}
	<div class="bg-card border border-border rounded-lg p-4">
		<div class="animate-pulse space-y-2">
			<div class="h-3 bg-muted rounded w-1/4"></div>
			<div class="h-5 bg-muted rounded w-1/3"></div>
		</div>
	</div>
{:else if sla}
	<div class="bg-card border border-border rounded-lg p-5">
		<div class="flex items-center space-x-2 mb-4">
			<Target class="w-4 h-4 text-muted-foreground" />
			<h3 class="text-sm font-semibold text-foreground">SLA Compliance</h3>
			<span class="text-[10px] text-muted-foreground ml-auto">Last {sla.period}</span>
		</div>

		<div class="grid grid-cols-3 gap-4">
			<!-- Uptime -->
			<div class="space-y-1">
				<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Uptime</p>
				<p class="text-xl font-bold font-mono {statusColor()}">{sla.uptime_percent.toFixed(2)}%</p>
			</div>

			<!-- Target -->
			<div class="space-y-1">
				<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Target</p>
				<p class="text-xl font-bold font-mono text-foreground">{sla.sla_target.toFixed(2)}%</p>
			</div>

			<!-- Margin -->
			<div class="space-y-1">
				<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Margin</p>
				<div class="flex items-center space-x-1">
					{#if sla.margin >= 0}
						<TrendingUp class="w-3.5 h-3.5 text-green-500" />
					{:else}
						<TrendingDown class="w-3.5 h-3.5 text-destructive" />
					{/if}
					<p class="text-xl font-bold font-mono {marginColor()}">
						{sla.margin >= 0 ? '+' : ''}{sla.margin.toFixed(2)}%
					</p>
				</div>
			</div>
		</div>

		{#if sla.breached}
			<div class="mt-3 px-3 py-2 bg-destructive/10 border border-destructive/20 rounded-md">
				<p class="text-xs text-destructive font-medium">SLA target breached — uptime is below {sla.sla_target}%</p>
			</div>
		{/if}
	</div>
{/if}
