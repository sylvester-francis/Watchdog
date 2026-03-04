<script lang="ts">
	import { Clock } from 'lucide-svelte';
	import type { TimelineEvent } from '$lib/types';

	interface Props {
		events: TimelineEvent[];
	}

	let { events }: Props = $props();

	function formatTime(iso: string): string {
		const d = new Date(iso);
		return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' });
	}

	function formatDate(iso: string): string {
		const d = new Date(iso);
		return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
	}

	function severityColor(severity: string): string {
		switch (severity) {
			case 'error': return 'border-red-500 bg-red-500/10';
			case 'warning': return 'border-yellow-500 bg-yellow-500/10';
			case 'info': return 'border-emerald-500 bg-emerald-500/10';
			default: return 'border-border bg-muted/50';
		}
	}

	function severityDotColor(severity: string): string {
		switch (severity) {
			case 'error': return 'bg-red-400';
			case 'warning': return 'bg-yellow-400';
			case 'info': return 'bg-emerald-400';
			default: return 'bg-muted-foreground';
		}
	}
</script>

<div class="bg-card border border-border rounded-lg">
	<div class="px-5 py-3.5 border-b border-border flex items-center space-x-2">
		<Clock class="w-4 h-4 text-muted-foreground" />
		<h3 class="text-sm font-medium text-foreground">Timeline</h3>
		<span class="text-xs text-muted-foreground">({events.length} events)</span>
	</div>

	<div class="p-5">
		{#if events.length === 0}
			<p class="text-xs text-muted-foreground text-center py-4">No timeline events</p>
		{:else}
			<div class="relative">
				<!-- Vertical line -->
				<div class="absolute left-[7px] top-2 bottom-2 w-px bg-border"></div>

				<div class="space-y-3">
					{#each events as event}
						<div class="flex items-start space-x-3 relative">
							<!-- Dot -->
							<div class="w-[15px] h-[15px] rounded-full border-2 flex-shrink-0 mt-0.5 {severityColor(event.severity)} flex items-center justify-center z-10">
								<div class="w-1.5 h-1.5 rounded-full {severityDotColor(event.severity)}"></div>
							</div>

							<!-- Content -->
							<div class="flex-1 min-w-0">
								<p class="text-xs text-foreground">{event.description}</p>
								<p class="text-[10px] text-muted-foreground font-mono mt-0.5">
									{formatDate(event.time)} {formatTime(event.time)}
								</p>
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	</div>
</div>
