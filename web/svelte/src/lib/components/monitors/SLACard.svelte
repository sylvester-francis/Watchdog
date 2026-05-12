<script lang="ts">
	import { TrendingUp, TrendingDown } from 'lucide-svelte';
	import { monitors as monitorsApi } from '$lib/api';
	import { uptimeColor } from '$lib/utils';
	import type { SLAData } from '$lib/types';
	import { onMount } from 'svelte';

	interface Props {
		monitorId: string;
	}

	let { monitorId }: Props = $props();

	type Period = '7d' | '30d' | '90d';
	const periods: { value: Period; label: string }[] = [
		{ value: '7d', label: '7D' },
		{ value: '30d', label: '30D' },
		{ value: '90d', label: '90D' }
	];

	let activePeriod = $state<Period>('30d');
	let sla = $state<SLAData | null>(null);
	let loading = $state(true);
	let failed = $state(false);

	async function fetchSLA(period: Period) {
		loading = true;
		try {
			const res = await monitorsApi.getMonitorSLA(monitorId, period);
			sla = res.data;
			failed = false;
		} catch {
			failed = true;
			sla = null;
		} finally {
			loading = false;
		}
	}

	function selectPeriod(period: Period) {
		activePeriod = period;
		fetchSLA(period);
	}

	onMount(() => {
		fetchSLA(activePeriod);
	});

	function marginColor(margin: number): string {
		return margin >= 0 ? 'text-emerald-400' : 'text-red-400';
	}
</script>

{#if failed && !loading}
	<!-- silently hide if no SLA data -->
{:else}
	<section>
		<div class="flex items-center justify-between border-b border-border pb-3">
			<h3 class="text-sm font-medium text-foreground">SLA Compliance</h3>
			<div class="flex items-center space-x-1">
				{#each periods as p}
					<button
						onclick={() => selectPeriod(p.value)}
						class="px-2.5 py-1 text-[10px] font-medium rounded transition-colors {activePeriod === p.value
							? 'bg-foreground/[0.08] text-foreground'
							: 'text-muted-foreground hover:text-foreground hover:bg-muted/50'}"
					>
						{p.label}
					</button>
				{/each}
			</div>
		</div>

		<div class="pt-4">
			{#if loading}
				<div class="animate-pulse">
					<div class="grid grid-cols-3 gap-4">
						{#each Array(3) as _}
							<div class="space-y-2">
								<div class="h-3 bg-muted/50 rounded w-1/2"></div>
								<div class="h-6 bg-muted/30 rounded w-2/3"></div>
							</div>
						{/each}
					</div>
				</div>
			{:else if sla}
				<div class="grid grid-cols-3 gap-4">
					<!-- Uptime -->
					<div class="space-y-1">
						<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Uptime</p>
						<p class="text-xl font-semibold font-mono {uptimeColor(sla.uptime_percent)}">{sla.uptime_percent.toFixed(2)}%</p>
					</div>

					<!-- Target -->
					<div class="space-y-1">
						<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Target</p>
						<p class="text-xl font-semibold font-mono text-foreground">{sla.sla_target.toFixed(2)}%</p>
					</div>

					<!-- Margin -->
					<div class="space-y-1">
						<p class="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Margin</p>
						<div class="flex items-center space-x-1">
							{#if sla.margin >= 0}
								<TrendingUp class="w-3.5 h-3.5 text-emerald-400" />
							{:else}
								<TrendingDown class="w-3.5 h-3.5 text-red-400" />
							{/if}
							<p class="text-xl font-semibold font-mono {marginColor(sla.margin)}">
								{sla.margin >= 0 ? '+' : ''}{sla.margin.toFixed(2)}%
							</p>
						</div>
					</div>
				</div>

				{#if sla.breached}
					<p class="mt-3 font-mono tabular-nums text-xs text-destructive">
						<span aria-hidden="true">●</span> SLA target breached — uptime is below {sla.sla_target}%
					</p>
				{/if}
			{/if}
		</div>
	</section>
{/if}
