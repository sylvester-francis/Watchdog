<script lang="ts">
	import type { Span } from '$lib/types';
	import { serviceBarClass, serviceChipClass } from '$lib/utils/serviceColors';

	interface Props {
		spans: Span[];
		selectedSpanId: string | null;
		onSelect: (spanId: string) => void;
	}

	let { spans, selectedSpanId, onSelect }: Props = $props();

	// Build a parent → children tree from the flat span list. Spans with
	// no parent_span_id (or one we can't resolve, e.g. spans whose parent
	// got dropped at decode time) are treated as roots.
	type TreeNode = {
		span: Span;
		children: TreeNode[];
		depth: number;
	};

	let tree = $derived.by(() => buildTree(spans));
	let flat = $derived.by(() => flatten(tree));

	function buildTree(input: Span[]): TreeNode[] {
		const byID = new Map<string, TreeNode>();
		for (const s of input) {
			byID.set(s.span_id, { span: s, children: [], depth: 0 });
		}
		const roots: TreeNode[] = [];
		for (const node of byID.values()) {
			const parentID = node.span.parent_span_id;
			if (parentID && byID.has(parentID)) {
				const parent = byID.get(parentID)!;
				parent.children.push(node);
				node.depth = parent.depth + 1;
			} else {
				roots.push(node);
			}
		}
		// Sort each level by start_time so the waterfall reads
		// chronologically within siblings.
		const sortByStart = (a: TreeNode, b: TreeNode) =>
			new Date(a.span.start_time).getTime() - new Date(b.span.start_time).getTime();
		const recurseSort = (nodes: TreeNode[]) => {
			nodes.sort(sortByStart);
			for (const n of nodes) recurseSort(n.children);
		};
		recurseSort(roots);
		// Re-set depth after sorting (depth is set by parent walk; sort
		// doesn't change it, but recompute to be safe in case the tree
		// gets edited later).
		const setDepth = (nodes: TreeNode[], d: number) => {
			for (const n of nodes) {
				n.depth = d;
				setDepth(n.children, d + 1);
			}
		};
		setDepth(roots, 0);
		return roots;
	}

	function flatten(nodes: TreeNode[]): TreeNode[] {
		const out: TreeNode[] = [];
		const walk = (ns: TreeNode[]) => {
			for (const n of ns) {
				out.push(n);
				if (n.children.length > 0) walk(n.children);
			}
		};
		walk(nodes);
		return out;
	}

	// Trace-wide time bounds drive bar positioning.
	let traceStart = $derived.by(() => {
		if (spans.length === 0) return 0;
		return Math.min(...spans.map((s) => new Date(s.start_time).getTime()));
	});
	let traceEnd = $derived.by(() => {
		if (spans.length === 0) return 0;
		return Math.max(...spans.map((s) => new Date(s.end_time).getTime()));
	});
	let totalMs = $derived(Math.max(traceEnd - traceStart, 1));

	function barLeftPct(s: Span): number {
		const start = new Date(s.start_time).getTime() - traceStart;
		return Math.max(0, (start / totalMs) * 100);
	}

	function barWidthPct(s: Span): number {
		const widthMs = s.duration_ns / 1_000_000;
		const pct = (widthMs / totalMs) * 100;
		// Always show at least a 1px-equivalent sliver so very fast spans
		// stay clickable / visible.
		return Math.max(pct, 0.4);
	}

	function formatDuration(ns: number): string {
		if (ns < 1_000) return `${ns}ns`;
		if (ns < 1_000_000) return `${(ns / 1_000).toFixed(1)}µs`;
		if (ns < 1_000_000_000) return `${(ns / 1_000_000).toFixed(1)}ms`;
		return `${(ns / 1_000_000_000).toFixed(2)}s`;
	}

	// Time-ruler tick marks at 0%, 25%, 50%, 75%, 100% of trace duration.
	let rulerTicks = $derived.by(() => {
		const ticks: { pct: number; label: string }[] = [];
		for (let i = 0; i <= 4; i++) {
			const pct = i * 25;
			const ms = (totalMs * i) / 4;
			ticks.push({ pct, label: i === 0 ? '0' : formatDuration(ms * 1_000_000) });
		}
		return ticks;
	});
</script>

<div class="bg-card border border-border rounded-lg overflow-hidden">
	<!-- Time-axis ruler (sticky to the top of the waterfall section) -->
	<div class="sticky top-0 z-10 bg-card border-b border-border/30 flex items-center text-[10px] text-muted-foreground/70 font-mono">
		<div class="w-72 shrink-0 px-3 py-2 border-r border-border/30 uppercase tracking-wider">
			Span
		</div>
		<div class="flex-1 relative h-7">
			{#each rulerTicks as tick}
				<div
					class="absolute top-0 bottom-0 border-l border-border/40 flex items-center pl-1 -translate-x-px"
					style="left: {tick.pct}%"
				>
					<span class="text-muted-foreground/70 tabular-nums">{tick.label}</span>
				</div>
			{/each}
		</div>
		<div class="w-24 shrink-0 px-3 py-2 border-l border-border/30 text-right uppercase tracking-wider">
			Duration
		</div>
	</div>

	<!-- Span rows -->
	<div class="divide-y divide-border/10 font-mono">
		{#each flat as node (node.span.span_id)}
			{@const s = node.span}
			{@const isError = s.status_code === 2}
			{@const isSelected = selectedSpanId === s.span_id}
			<button
				type="button"
				onclick={() => onSelect(s.span_id)}
				class="w-full flex items-center text-left transition-colors group {isSelected
					? 'bg-foreground/[0.06]'
					: 'hover:bg-foreground/[0.03]'}"
			>
				<!-- Span name + indent -->
				<div class="w-72 shrink-0 px-3 py-2 border-r border-border/20 flex items-center gap-2 min-w-0">
					<div style="width: {node.depth * 12}px" class="shrink-0"></div>
					<!-- Service chip -->
					<span
						class="shrink-0 inline-flex items-center px-1.5 py-0.5 rounded text-[9px] uppercase tracking-wider {serviceChipClass(s.service_name)}"
						title={s.service_name}
					>
						{s.service_name.length > 8 ? s.service_name.slice(0, 8) + '…' : s.service_name}
					</span>
					<span class="text-xs text-foreground truncate">{s.name}</span>
				</div>

				<!-- Time-aligned bar track -->
				<div class="flex-1 relative h-8 self-stretch">
					<!-- gridlines fade -->
					{#each rulerTicks as tick}
						<div
							class="absolute top-0 bottom-0 border-l border-border/15"
							style="left: {tick.pct}%"
						></div>
					{/each}
					<!-- bar -->
					<div
						class="absolute top-1/2 -translate-y-1/2 h-3 rounded-sm {serviceBarClass(s.service_name)} {isError
							? 'ring-1 ring-inset ring-red-400/80'
							: ''}"
						style="left: {barLeftPct(s)}%; width: {barWidthPct(s)}%"
					></div>
					{#if isError}
						<span
							class="absolute top-1/2 -translate-y-1/2 w-1.5 h-1.5 rounded-full bg-red-400 shadow-[0_0_4px_rgba(239,68,68,0.6)]"
							style="left: calc({barLeftPct(s) + barWidthPct(s)}% + 4px)"
							aria-hidden="true"
						></span>
					{/if}
				</div>

				<!-- Duration -->
				<div class="w-24 shrink-0 px-3 py-2 border-l border-border/20 text-right tabular-nums text-xs {isError
					? 'text-red-400'
					: s.duration_ns >= 1_000_000_000
						? 'text-amber-400'
						: 'text-muted-foreground'}">
					{formatDuration(s.duration_ns)}
				</div>
			</button>
		{/each}
	</div>
</div>
