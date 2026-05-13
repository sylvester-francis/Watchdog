<script lang="ts">
	interface Props {
		open: number;
		acknowledged: number;
		resolved: number;
	}

	let { open, acknowledged, resolved }: Props = $props();

	const cellClass = 'flex flex-col bg-background px-4 py-3.5';
	const labelClass = 'text-[11px] font-medium uppercase tracking-wider text-muted-foreground';
	const valueClass = 'mt-1 font-mono tabular-nums text-lg';
</script>

<section class="grid grid-cols-3 gap-px overflow-hidden border-y border-border bg-border">
	<div class={cellClass}>
		<div class={labelClass}>Open</div>
		<div class="{valueClass} {open > 0 ? 'text-destructive' : 'text-foreground'}">{open}</div>
	</div>
	<div class={cellClass}>
		<div class={labelClass}>Acknowledged</div>
		<div class="{valueClass} {acknowledged > 0 ? 'text-warning' : 'text-foreground'}">{acknowledged}</div>
	</div>
	<div class={cellClass}>
		<div class={labelClass}>Resolved</div>
		<div class="{valueClass} text-foreground">{resolved}</div>
	</div>
</section>
