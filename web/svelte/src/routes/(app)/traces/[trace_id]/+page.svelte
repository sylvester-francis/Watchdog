<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { ArrowLeft, AlertCircle, Copy, Check } from 'lucide-svelte';
	import { traces as tracesApi } from '$lib/api';
	import type { Span } from '$lib/types';
	import Waterfall from '$lib/components/traces/Waterfall.svelte';
	import SpanDetailRail from '$lib/components/traces/SpanDetailRail.svelte';

	let traceId = $derived(page.params.trace_id ?? '');

	let spans = $state<Span[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let selectedSpanId = $state<string | null>(null);
	let copiedTraceId = $state(false);
	let copyTimer: ReturnType<typeof setTimeout> | null = null;

	let selectedSpan = $derived.by(() => {
		if (!selectedSpanId) return null;
		return spans.find((s) => s.span_id === selectedSpanId) ?? null;
	});

	let traceStart = $derived.by(() => {
		if (spans.length === 0) return 0;
		return Math.min(...spans.map((s) => new Date(s.start_time).getTime()));
	});
	let traceEnd = $derived.by(() => {
		if (spans.length === 0) return 0;
		return Math.max(...spans.map((s) => new Date(s.end_time).getTime()));
	});
	let totalDurationNs = $derived(Math.max((traceEnd - traceStart) * 1_000_000, 0));
	let serviceCount = $derived(new Set(spans.map((s) => s.service_name)).size);
	let errorCount = $derived(spans.filter((s) => s.status_code === 2).length);

	let rootSpan = $derived.by(() => {
		if (spans.length === 0) return null;
		// A trace's root is the span with no resolvable parent. If the
		// parent_span_id is set but its span isn't in this trace's set
		// (orphan after dropping), treat as root too.
		const ids = new Set(spans.map((s) => s.span_id));
		return (
			spans.find((s) => !s.parent_span_id || !ids.has(s.parent_span_id)) ?? spans[0]
		);
	});

	async function loadTrace() {
		loading = true;
		loadError = null;
		try {
			const resp = await tracesApi.getTrace(traceId);
			spans = resp.data ?? [];
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load trace';
		} finally {
			loading = false;
		}
	}

	function handleSpanSelect(spanId: string) {
		selectedSpanId = selectedSpanId === spanId ? null : spanId;
	}

	function closeRail() {
		selectedSpanId = null;
	}

	async function copyTraceID() {
		try {
			await navigator.clipboard.writeText(traceId);
			copiedTraceId = true;
			if (copyTimer) clearTimeout(copyTimer);
			copyTimer = setTimeout(() => (copiedTraceId = false), 1000);
		} catch {
			// noop
		}
	}

	function formatDuration(ns: number): string {
		if (ns < 1_000) return `${ns}ns`;
		if (ns < 1_000_000) return `${(ns / 1_000).toFixed(1)}µs`;
		if (ns < 1_000_000_000) return `${(ns / 1_000_000).toFixed(1)}ms`;
		return `${(ns / 1_000_000_000).toFixed(2)}s`;
	}

	function formatTimeRange(startMs: number, endMs: number): string {
		const fmt = (ms: number) => {
			const d = new Date(ms);
			const hh = String(d.getHours()).padStart(2, '0');
			const mm = String(d.getMinutes()).padStart(2, '0');
			const ss = String(d.getSeconds()).padStart(2, '0');
			const ml = String(d.getMilliseconds()).padStart(3, '0');
			return `${hh}:${mm}:${ss}.${ml}`;
		};
		return `${fmt(startMs)} → ${fmt(endMs)}`;
	}

	onMount(loadTrace);
</script>

<svelte:head>
	<title>Trace {traceId.slice(0, 8)} - WatchDog</title>
</svelte:head>

<div class="animate-fade-in-up">
	<a href="/traces" class="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors mb-4">
		<ArrowLeft class="w-3.5 h-3.5" />
		<span>traces</span>
	</a>

	{#if loading}
		<div class="space-y-4">
			<div class="h-8 w-64 bg-muted/50 rounded animate-pulse"></div>
			<div class="h-5 w-96 bg-muted/30 rounded animate-pulse"></div>
			<div class="bg-card border border-border rounded-lg overflow-hidden">
				{#each Array(8) as _}
					<div class="flex items-center px-4 py-2.5 border-b border-border/20">
						<div class="w-44 h-3 bg-muted/40 rounded animate-pulse"></div>
						<div class="ml-auto w-72 h-3 bg-muted/30 rounded animate-pulse"></div>
						<div class="ml-4 w-16 h-3 bg-muted/30 rounded animate-pulse"></div>
					</div>
				{/each}
			</div>
		</div>
	{:else if loadError}
		<div class="bg-card border border-border rounded-lg">
			<div class="p-12 text-center">
				<div class="w-12 h-12 bg-muted/50 rounded-lg flex items-center justify-center mx-auto mb-4">
					<AlertCircle class="w-6 h-6 text-red-400/70" />
				</div>
				<p class="text-sm font-medium text-foreground mb-1">Couldn't load trace</p>
				<p class="text-xs text-muted-foreground mb-4 font-mono">{loadError}</p>
				<button
					onclick={loadTrace}
					class="inline-flex items-center space-x-1.5 px-4 py-2 bg-muted/50 hover:bg-muted text-xs font-medium rounded-md transition-colors text-foreground"
				>
					Try again
				</button>
			</div>
		</div>
	{:else if spans.length === 0}
		<div class="bg-card border border-border rounded-lg">
			<div class="p-12 text-center">
				<p class="text-sm text-muted-foreground">Trace not found.</p>
			</div>
		</div>
	{:else}
		<div class="grid grid-cols-1 lg:grid-cols-[1fr_minmax(320px,420px)] gap-4">
			<div class="min-w-0">
				<!-- Header strip -->
				<div class="mb-4">
					<h1 class="text-lg font-semibold text-foreground font-mono">
						{rootSpan?.name ?? 'Trace'}
					</h1>
					<div class="flex items-center gap-1.5 text-[11px] text-muted-foreground/80 font-mono mt-0.5">
						<span class="tabular-nums">{traceId}</span>
						<button
							onclick={copyTraceID}
							class="text-muted-foreground/60 hover:text-foreground transition-colors"
							aria-label="Copy trace_id"
						>
							{#if copiedTraceId}
								<Check class="w-3 h-3 text-emerald-400" />
							{:else}
								<Copy class="w-3 h-3" />
							{/if}
						</button>
					</div>
					<div class="mt-3 flex flex-wrap items-center gap-x-5 gap-y-1 text-xs font-mono">
						<span class="tabular-nums text-foreground">{formatDuration(totalDurationNs)}</span>
						<span class="text-muted-foreground"><span class="tabular-nums">{spans.length}</span> spans</span>
						<span class="text-muted-foreground"><span class="tabular-nums">{serviceCount}</span> service{serviceCount === 1 ? '' : 's'}</span>
						{#if errorCount > 0}
							<span class="text-red-400">● <span class="tabular-nums">{errorCount}</span> error{errorCount === 1 ? '' : 's'}</span>
						{/if}
						<span class="text-muted-foreground/70 ml-auto tabular-nums">
							{formatTimeRange(traceStart, traceEnd)}
						</span>
					</div>
				</div>

				<Waterfall
					{spans}
					{selectedSpanId}
					onSelect={handleSpanSelect}
				/>
			</div>

			{#if selectedSpan}
				<div class="lg:sticky lg:top-4 self-start">
					<SpanDetailRail span={selectedSpan} {traceStart} onClose={closeRail} />
				</div>
			{/if}
		</div>
	{/if}
</div>
