<script lang="ts">
	interface Props {
		open: number;
		acknowledged: number;
		resolved: number;
	}

	let { open, acknowledged, resolved }: Props = $props();
</script>

<div class="flex items-center space-x-3 flex-wrap">
	<!-- Open pill -->
	{#if open > 0}
		<div class="flex items-center space-x-1.5 px-2.5 py-1 rounded-md bg-red-500/10 border border-red-500/20 border-l-[3px] border-l-red-500">
			<div class="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse"></div>
			<span class="text-xs font-medium text-red-400 font-mono">{open} open</span>
		</div>
	{/if}

	<!-- Acknowledged pill -->
	{#if acknowledged > 0}
		<div class="flex items-center space-x-1.5 px-2.5 py-1 rounded-md bg-yellow-500/10 border border-yellow-500/20 border-l-[3px] border-l-amber-500">
			<div class="w-1.5 h-1.5 rounded-full bg-yellow-400"></div>
			<span class="text-xs font-medium text-yellow-400 font-mono">{acknowledged} acked</span>
		</div>
	{/if}

	<!-- Resolved pill -->
	{#if resolved > 0}
		<div class="flex items-center space-x-1.5 px-2.5 py-1 rounded-md bg-emerald-500/10 border border-emerald-500/20 border-l-[3px] border-l-emerald-500">
			<div class="w-1.5 h-1.5 rounded-full bg-emerald-400"></div>
			<span class="text-xs font-medium text-emerald-400 font-mono">{resolved} resolved</span>
		</div>
	{/if}
</div>
