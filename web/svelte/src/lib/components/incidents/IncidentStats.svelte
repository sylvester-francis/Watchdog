<script lang="ts">
	import { AlertCircle, Clock, CheckCircle2 } from 'lucide-svelte';

	interface Props {
		open: number;
		acknowledged: number;
		resolved: number;
	}

	let { open, acknowledged, resolved }: Props = $props();
</script>

<div class="flex items-center gap-3 flex-wrap">
	<!-- Open pill -->
	<div class="flex items-center space-x-2 px-3 py-1.5 bg-card border border-border rounded-lg">
		<div class="flex items-center space-x-1.5">
			<AlertCircle class="w-3.5 h-3.5 text-red-400" />
			<span class="text-xs text-muted-foreground">Open</span>
		</div>
		<span class="inline-flex items-center space-x-1.5 text-xs font-mono font-medium
			{open > 0 ? 'text-red-400' : 'text-muted-foreground'}">
			{#if open > 0}
				<span class="relative flex h-2 w-2">
					<span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75"></span>
					<span class="relative inline-flex rounded-full h-2 w-2 bg-red-400"></span>
				</span>
			{/if}
			<span>{open}</span>
		</span>
	</div>

	<!-- Acknowledged pill -->
	<div class="flex items-center space-x-2 px-3 py-1.5 bg-card border border-border rounded-lg">
		<div class="flex items-center space-x-1.5">
			<Clock class="w-3.5 h-3.5 text-yellow-400" />
			<span class="text-xs text-muted-foreground">Acknowledged</span>
		</div>
		<span class="text-xs font-mono font-medium {acknowledged > 0 ? 'text-yellow-400' : 'text-muted-foreground'}">
			{acknowledged}
		</span>
	</div>

	<!-- Resolved pill -->
	<div class="flex items-center space-x-2 px-3 py-1.5 bg-card border border-border rounded-lg">
		<div class="flex items-center space-x-1.5">
			<CheckCircle2 class="w-3.5 h-3.5 text-emerald-400" />
			<span class="text-xs text-muted-foreground">Resolved</span>
		</div>
		<span class="text-xs font-mono font-medium {resolved > 0 ? 'text-emerald-400' : 'text-muted-foreground'}">
			{resolved}
		</span>
	</div>
</div>
