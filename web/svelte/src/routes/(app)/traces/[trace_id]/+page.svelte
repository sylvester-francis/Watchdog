<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { ArrowLeft, Copy, Check, RefreshCw } from 'lucide-svelte';
	import { Button } from '@sylvester-francis/watchdog-ui';
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
		const ids = new Set(spans.map((s) => s.span_id));
		return spans.find((s) => !s.parent_span_id || !ids.has(s.parent_span_id)) ?? spans[0];
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

<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-6 sm:px-6 sm:py-10">
	<a href="/traces" class="inline-flex items-center gap-1.5 text-xs text-muted-foreground transition-colors hover:text-foreground">
		<ArrowLeft class="h-3.5 w-3.5" />
		<span>Back to traces</span>
	</a>

	{#if loading}
		<div class="mt-6 animate-pulse space-y-4">
			<div class="h-8 w-64 bg-muted/50"></div>
			<div class="h-4 w-96 bg-muted/30"></div>
			<div class="space-y-2 pt-4">
				{#each Array(8) as _}
					<div class="flex items-center gap-6 py-2.5">
						<div class="h-3 w-44 bg-muted/40"></div>
						<div class="ml-auto h-3 w-72 bg-muted/30"></div>
						<div class="h-3 w-16 bg-muted/30"></div>
					</div>
				{/each}
			</div>
		</div>
	{:else if loadError}
		<section class="mt-6">
			<div class="border-b border-border pb-3">
				<h3 class="text-sm font-medium text-foreground">Couldn't load trace</h3>
			</div>
			<div class="flex flex-col items-start gap-3 pt-4 sm:flex-row sm:items-center sm:justify-between sm:gap-6">
				<p class="font-mono tabular-nums text-xs text-destructive">{loadError}</p>
				<Button variant="secondary" onclick={loadTrace}>
					<RefreshCw class="mr-1.5 h-3.5 w-3.5" />
					<span>Try again</span>
				</Button>
			</div>
		</section>
	{:else if spans.length === 0}
		<section class="mt-6">
			<div class="border-b border-border pb-3">
				<h3 class="text-sm font-medium text-foreground">Trace not found</h3>
			</div>
		</section>
	{:else}
		<!-- Header strip -->
		<header class="mt-6">
			<div class="flex items-center gap-2 font-mono tabular-nums text-xs text-muted-foreground">
				<span class="uppercase tracking-wider">Trace</span>
				<span class="text-muted-foreground/40">·</span>
				<span>{traceId}</span>
				<button
					onclick={copyTraceID}
					class="text-muted-foreground/60 transition-colors hover:text-foreground"
					aria-label="Copy trace_id"
				>
					{#if copiedTraceId}
						<Check class="h-3 w-3 text-success" />
					{:else}
						<Copy class="h-3 w-3" />
					{/if}
				</button>
			</div>
			<h1 class="mt-1.5 truncate font-mono tabular-nums text-xl font-medium text-foreground sm:text-2xl md:text-3xl">
				{rootSpan?.name ?? 'Trace'}
			</h1>
			<div class="mt-2 flex flex-wrap items-center gap-x-5 gap-y-1 font-mono tabular-nums text-xs">
				<span class="text-foreground">{formatDuration(totalDurationNs)}</span>
				<span class="text-muted-foreground"><span class="text-foreground">{spans.length}</span> spans</span>
				<span class="text-muted-foreground"><span class="text-foreground">{serviceCount}</span> service{serviceCount === 1 ? '' : 's'}</span>
				{#if errorCount > 0}
					<span class="flex items-center gap-1.5 text-destructive">
						<span class="inline-block h-1.5 w-1.5 rounded-full bg-destructive"></span>
						<span>{errorCount}</span> error{errorCount === 1 ? '' : 's'}
					</span>
				{/if}
				<span class="ml-auto text-muted-foreground/70">{formatTimeRange(traceStart, traceEnd)}</span>
			</div>
		</header>

		<div class="mt-8 grid grid-cols-1 gap-6 lg:grid-cols-[1fr_minmax(320px,420px)]">
			<div class="min-w-0">
				<Waterfall {spans} {selectedSpanId} onSelect={handleSpanSelect} />
			</div>

			{#if selectedSpan}
				<div class="self-start lg:sticky lg:top-4">
					<SpanDetailRail span={selectedSpan} {traceStart} onClose={closeRail} />
				</div>
			{/if}
		</div>
	{/if}
</div>
