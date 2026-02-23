<script lang="ts">
	import type { Monitor } from '$lib/types';

	interface Props {
		monitor: Monitor;
	}

	let { monitor }: Props = $props();

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

<div class="flex items-start justify-between">
	<div class="flex items-start space-x-3">
		<div class="mt-1.5">
			<div class="w-3 h-3 rounded-full {statusDotClass(monitor.status)}"></div>
		</div>
		<div>
			<div class="flex items-center space-x-2.5">
				<h1 class="text-xl font-semibold text-foreground">{monitor.name}</h1>
				<span class="text-[9px] font-mono uppercase tracking-wider px-2 py-0.5 rounded bg-muted/50 text-muted-foreground">
					{monitor.type}
				</span>
				<span class="text-xs font-medium {statusTextClass(monitor.status)}">
					{statusLabel(monitor.status)}
				</span>
			</div>
			<p class="text-xs text-muted-foreground font-mono mt-1">{monitor.target}</p>
		</div>
	</div>
</div>
