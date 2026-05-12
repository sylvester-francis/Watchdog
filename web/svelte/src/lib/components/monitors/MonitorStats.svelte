<script lang="ts">
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

	let isPortScan = $derived(monitor.type === 'port_scan');
	let scannedCount = $derived(monitor.metadata?.scanned_count ?? '0');
	let openCount = $derived(monitor.metadata?.open_count ?? '0');
	let hasDrift = $derived(
		(monitor.metadata?.missing_ports?.split(',').filter(Boolean).length ?? 0) > 0 ||
			(monitor.metadata?.unexpected_ports?.split(',').filter(Boolean).length ?? 0) > 0
	);

	let tlsDays = $derived(
		monitor.type === 'tls' && monitor.metadata?.cert_expiry_days
			? parseInt(monitor.metadata.cert_expiry_days)
			: null
	);
	let tlsColorClass = $derived(
		tlsDays === null
			? ''
			: tlsDays < 14
				? 'text-destructive'
				: tlsDays < 30
					? 'text-warning'
					: 'text-success'
	);

	const cellClass = 'flex flex-col bg-background px-4 py-3.5';
	const labelClass = 'text-[11px] font-medium uppercase tracking-wider text-muted-foreground';
	const valueClass = 'mt-1 font-mono tabular-nums text-lg text-foreground';
</script>

<section class="grid grid-cols-2 gap-px overflow-hidden border-y border-border bg-border sm:grid-cols-3 md:grid-cols-4">
	{#if isPortScan}
		<div class={cellClass}>
			<div class={labelClass}>Scanned</div>
			<div class={valueClass}>{scannedCount}</div>
		</div>
		<div class={cellClass}>
			<div class={labelClass}>Open</div>
			<div class="{valueClass} text-success">{openCount}</div>
		</div>
		<div class={cellClass}>
			<div class={labelClass}>Compliance</div>
			<div class="{valueClass} {hasDrift ? 'text-destructive' : 'text-success'}">
				{hasDrift ? 'Drift' : 'Clean'}
			</div>
		</div>
		<div class={cellClass}>
			<div class={labelClass}>Agent</div>
			<div class="mt-1 truncate text-sm text-foreground">{agentName}</div>
		</div>
	{:else}
		<div class={cellClass}>
			<div class={labelClass}>Interval</div>
			<div class={valueClass}>{monitor.interval_seconds}s</div>
		</div>
		<div class={cellClass}>
			<div class={labelClass}>Timeout</div>
			<div class={valueClass}>{monitor.timeout_seconds}s</div>
		</div>
		<div class={cellClass}>
			<div class={labelClass}>Uptime</div>
			<div class="{valueClass} {uptimeColorClass}">{uptimeDisplay}%</div>
		</div>
		<div class={cellClass}>
			<div class={labelClass}>Agent</div>
			<div class="mt-1 truncate text-sm text-foreground">{agentName}</div>
		</div>
	{/if}

	{#if tlsDays !== null}
		<div class={cellClass}>
			<div class={labelClass}>Cert Expiry</div>
			<div class="{valueClass} {tlsColorClass}">{tlsDays}d</div>
		</div>
	{/if}

	{#if monitor.type === 'tls' && monitor.metadata?.cert_issuer}
		<div class={cellClass}>
			<div class={labelClass}>Issuer</div>
			<div class="mt-1 truncate text-sm text-foreground">{monitor.metadata.cert_issuer}</div>
		</div>
	{/if}

	{#if monitor.type === 'docker' && monitor.metadata?.container_name}
		<div class={cellClass}>
			<div class={labelClass}>Container</div>
			<div class="mt-1 truncate font-mono tabular-nums text-sm text-foreground">
				{monitor.metadata.container_name}
			</div>
		</div>
	{/if}

	{#if monitor.type === 'database' && monitor.metadata?.db_type}
		<div class={cellClass}>
			<div class={labelClass}>DB Type</div>
			<div class="mt-1 truncate text-sm capitalize text-foreground">{monitor.metadata.db_type}</div>
		</div>
	{/if}

	{#if monitor.type === 'system' && monitor.metadata?.metric_name}
		<div class={cellClass}>
			<div class={labelClass}>Metric</div>
			<div class="mt-1 truncate text-sm capitalize text-foreground">
				{monitor.metadata.metric_name}
			</div>
		</div>
	{/if}

	{#if monitor.type === 'service'}
		<div class={cellClass}>
			<div class={labelClass}>Service</div>
			<div class="mt-1 truncate font-mono tabular-nums text-sm text-foreground">{monitor.target}</div>
		</div>
	{/if}
</section>
