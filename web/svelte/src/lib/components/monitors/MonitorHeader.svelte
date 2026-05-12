<script lang="ts">
	import type { Monitor } from '$lib/types';

	interface Props {
		monitor: Monitor;
		onEdit?: () => void;
	}

	let { monitor, onEdit }: Props = $props();

	function statusPipClass(status: string): string {
		if (status === 'up') return 'bg-success';
		if (status === 'down') return 'bg-destructive';
		if (status === 'degraded') return 'bg-warning';
		return 'bg-muted-foreground/50';
	}

	function statusLabel(status: string): string {
		if (status === 'up') return 'Operational';
		if (status === 'down') return 'Down';
		if (status === 'degraded') return 'Degraded';
		return 'Pending';
	}
</script>

<header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between sm:gap-4">
	<div class="min-w-0">
		<div class="flex items-center gap-2 font-mono tabular-nums text-xs text-muted-foreground">
			<span class="inline-block h-1.5 w-1.5 rounded-full {statusPipClass(monitor.status)}" aria-label="Status: {monitor.status}"></span>
			<span class="uppercase tracking-wider">{statusLabel(monitor.status)}</span>
			<span class="text-muted-foreground/40">·</span>
			<span class="uppercase tracking-wider">{monitor.type}</span>
		</div>
		<h1 class="mt-1.5 truncate text-2xl font-medium text-foreground sm:text-3xl">
			{monitor.name}
		</h1>
		<div class="mt-1 truncate font-mono tabular-nums text-sm text-muted-foreground">
			{monitor.target}
		</div>
	</div>
	{#if onEdit}
		<button
			onclick={onEdit}
			class="shrink-0 self-start text-sm text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline sm:self-auto"
		>
			Edit
		</button>
	{/if}
</header>
