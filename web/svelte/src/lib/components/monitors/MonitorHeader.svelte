<script lang="ts">
	import { Pencil } from 'lucide-svelte';
	import type { Monitor } from '$lib/types';
	import Pill from '$lib/ui/Pill.svelte';
	import Button from '$lib/ui/Button.svelte';

	interface Props {
		monitor: Monitor;
		onEdit?: () => void;
	}

	let { monitor, onEdit }: Props = $props();

	function statusDotClass(status: string): string {
		if (status === 'up') return 'bg-emerald-400 shadow-[0_0_8px_rgba(34,197,94,0.5)]';
		if (status === 'down') return 'bg-red-400 shadow-[0_0_8px_rgba(239,68,68,0.5)]';
		if (status === 'degraded') return 'bg-amber-400 shadow-[0_0_8px_rgba(245,158,11,0.5)]';
		return 'bg-muted-foreground/50';
	}

	function statusLabel(status: string): string {
		if (status === 'up') return 'Operational';
		if (status === 'down') return 'Down';
		if (status === 'degraded') return 'Degraded';
		return 'Pending';
	}

	function statusTextClass(status: string): string {
		if (status === 'up') return 'text-emerald-400';
		if (status === 'down') return 'text-red-400';
		if (status === 'degraded') return 'text-amber-400';
		return 'text-muted-foreground';
	}
</script>

<div class="bg-card border border-border rounded-lg p-5">
	<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
		<div class="flex items-center space-x-4">
			<div class="w-10 h-10 rounded-lg {monitor.status === 'up' ? 'bg-emerald-500/10' : monitor.status === 'down' ? 'bg-red-500/10' : 'bg-muted/50'} flex items-center justify-center">
				<div class="w-3 h-3 rounded-full {statusDotClass(monitor.status)}" aria-label="Status: {monitor.status}"></div>
			</div>
			<div>
				<h2 class="text-lg font-semibold text-foreground">{monitor.name}</h2>
				<div class="flex items-center space-x-3 mt-1">
					<Pill tone="neutral">
					<span class="uppercase font-mono">{monitor.type}</span>
				</Pill>
					<span class="text-xs text-muted-foreground font-mono">{monitor.target}</span>
				</div>
			</div>
		</div>
		<div class="flex items-center space-x-3">
			<span class="text-sm font-medium {statusTextClass(monitor.status)}">{statusLabel(monitor.status)}</span>
			{#if onEdit}
				<Button variant="secondary" size="sm" onclick={onEdit}>
					<span class="flex items-center gap-1.5">
						<Pencil class="w-3.5 h-3.5" />
						<span>Edit</span>
					</span>
				</Button>
			{/if}
		</div>
	</div>
</div>
