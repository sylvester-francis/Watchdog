<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { GitBranch, RefreshCw, Search, AlertCircle, Copy, Check, ChevronDown } from 'lucide-svelte';
	import { Button, EmptyState, Input, Select, Skeleton, Tabs } from '@sylvester-francis/watchdog-ui';
	import { traces as tracesApi } from '$lib/api';
	import type { TraceSummary } from '$lib/types';

	type TimeRange = '1h' | '6h' | '24h';
	type AutoRefresh = 'off' | '15s' | '30s' | '60s';

	let summaries = $state<TraceSummary[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	// Pagination state — when "Load older" is clicked we append to summaries
	// instead of replacing. hasMore goes false once a page returns < pageSize
	// rows, signaling the end of the data set.
	const pageSize = 200;
	let loadingMore = $state(false);
	let hasMore = $state(true);

	let timeRange = $state<TimeRange>('1h');
	let serviceFilter = $state('');
	let errorsOnly = $state(false);
	let autoRefresh = $state<AutoRefresh>('off');

	let copiedTraceID = $state<string | null>(null);
	let copyResetTimer: ReturnType<typeof setTimeout> | null = null;
	let refreshTimer: ReturnType<typeof setInterval> | null = null;

	const timeTabs: { value: TimeRange; label: string; lookbackMs: number }[] = [
		{ value: '1h', label: 'Last 1h', lookbackMs: 60 * 60 * 1000 },
		{ value: '6h', label: 'Last 6h', lookbackMs: 6 * 60 * 60 * 1000 },
		{ value: '24h', label: 'Last 24h', lookbackMs: 24 * 60 * 60 * 1000 }
	];

	const refreshOptions: { value: AutoRefresh; label: string; intervalMs: number }[] = [
		{ value: 'off', label: 'Off', intervalMs: 0 },
		{ value: '15s', label: '15s', intervalMs: 15_000 },
		{ value: '30s', label: '30s', intervalMs: 30_000 },
		{ value: '60s', label: '60s', intervalMs: 60_000 }
	];

	let filtered = $derived.by(() => {
		if (!errorsOnly) return summaries;
		return summaries.filter((t) => t.has_error);
	});

	function lookbackForRange(r: TimeRange): number {
		return timeTabs.find((t) => t.value === r)?.lookbackMs ?? 60 * 60 * 1000;
	}

	function sinceForRange(r: TimeRange): string {
		return new Date(Date.now() - lookbackForRange(r)).toISOString();
	}

	async function loadData() {
		loadError = null;
		try {
			const resp = await tracesApi.listTraces({
				since: sinceForRange(timeRange),
				service: serviceFilter.trim() || undefined,
				limit: pageSize
			});
			summaries = resp.data ?? [];
			hasMore = (resp.data ?? []).length >= pageSize;
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load traces';
			summaries = [];
			hasMore = false;
		} finally {
			loading = false;
		}
	}

	async function loadMore() {
		if (loadingMore || !hasMore || summaries.length === 0) return;
		loadingMore = true;
		try {
			const oldest = summaries[summaries.length - 1].start_time;
			const resp = await tracesApi.listTraces({
				since: sinceForRange(timeRange),
				service: serviceFilter.trim() || undefined,
				before: oldest,
				limit: pageSize
			});
			const next = resp.data ?? [];
			summaries = [...summaries, ...next];
			hasMore = next.length >= pageSize;
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load more traces';
		} finally {
			loadingMore = false;
		}
	}

	function applyFilters() {
		loading = true;
		hasMore = true;
		void loadData();
	}

	function startAutoRefresh() {
		if (refreshTimer) {
			clearInterval(refreshTimer);
			refreshTimer = null;
		}
		const opt = refreshOptions.find((o) => o.value === autoRefresh);
		if (opt && opt.intervalMs > 0) {
			refreshTimer = setInterval(() => void loadData(), opt.intervalMs);
		}
	}

	$effect(() => {
		// Reload whenever the time range or service filter changes.
		void timeRange;
		void serviceFilter;
		applyFilters();
	});

	$effect(() => {
		void autoRefresh;
		startAutoRefresh();
	});

	function shortHex(hex: string): string {
		return hex.length > 8 ? hex.slice(0, 8) : hex;
	}

	async function copyTraceID(hex: string) {
		try {
			await navigator.clipboard.writeText(hex);
			copiedTraceID = hex;
			if (copyResetTimer) clearTimeout(copyResetTimer);
			copyResetTimer = setTimeout(() => {
				copiedTraceID = null;
			}, 1000);
		} catch {
			// Clipboard not available — silently noop. The user can still
			// drag-select the visible 8-char prefix.
		}
	}

	function formatDuration(ns: number): string {
		if (ns < 1_000) return `${ns}ns`;
		if (ns < 1_000_000) return `${(ns / 1_000).toFixed(1)}µs`;
		if (ns < 1_000_000_000) return `${(ns / 1_000_000).toFixed(1)}ms`;
		return `${(ns / 1_000_000_000).toFixed(2)}s`;
	}

	// Color buckets: ≤100ms muted, 100ms–1s normal, ≥1s warn, errors override.
	function durationClass(ns: number, hasError: boolean): string {
		if (hasError) return 'text-red-400';
		if (ns >= 1_000_000_000) return 'text-amber-400';
		if (ns >= 100_000_000) return 'text-foreground';
		return 'text-muted-foreground';
	}

	function relativeTime(iso: string): string {
		const then = new Date(iso).getTime();
		const diffMs = Date.now() - then;
		if (diffMs < 0) return 'in the future';
		const sec = Math.floor(diffMs / 1000);
		if (sec < 60) return `${sec}s ago`;
		const min = Math.floor(sec / 60);
		if (min < 60) return `${min}m ${sec % 60}s ago`;
		const hr = Math.floor(min / 60);
		return `${hr}h ${min % 60}m ago`;
	}

	onMount(() => {
		loadData();
	});

	onDestroy(() => {
		if (refreshTimer) clearInterval(refreshTimer);
		if (copyResetTimer) clearTimeout(copyResetTimer);
	});
</script>

<svelte:head>
	<title>Traces - WatchDog</title>
</svelte:head>

<div class="animate-fade-in-up">
	<!-- Page header -->
	<div class="flex items-center justify-between mb-5">
		<div>
			<h1 class="text-lg font-semibold text-foreground">Traces</h1>
			<p class="text-xs text-muted-foreground mt-0.5">Distributed traces ingested via OTLP</p>
		</div>
		<Button variant="secondary" onclick={() => applyFilters()} aria-label="Refresh">
			<RefreshCw class="w-3.5 h-3.5 {loading ? 'animate-spin' : ''} mr-1.5" />
			Refresh
		</Button>
	</div>

	<!-- Filter bar -->
	<div class="flex flex-col sm:flex-row sm:items-center gap-3 mb-4">
		<Tabs
			options={timeTabs as Array<{ value: string; label: string }>}
			value={timeRange}
			variant="pill"
			onchange={(v) => { timeRange = v as TimeRange; }}
		/>

		<div class="flex items-center gap-2 sm:ml-auto">
			<div class="w-44 sm:w-56">
				<Input bind:value={serviceFilter} placeholder="Filter by service...">
					{#snippet iconLeft()}
						<Search class="w-3.5 h-3.5" />
					{/snippet}
				</Input>
			</div>

			<label class="flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground cursor-pointer select-none">
				<input
					type="checkbox"
					bind:checked={errorsOnly}
					class="w-3.5 h-3.5 rounded border-border bg-card focus:ring-1 focus:ring-ring accent-red-400"
				/>
				<span>Errors only</span>
			</label>

			<div class="w-28">
				<Select bind:value={autoRefresh} size="sm" aria-label="Auto-refresh">
					{#each refreshOptions as r}
						<option value={r.value}>↻ {r.label}</option>
					{/each}
				</Select>
			</div>
		</div>
	</div>

	{#if loading}
		<!-- Skeleton -->
		<div class="bg-card border border-border rounded-lg overflow-hidden">
			{#each Array(5) as _}
				<div class="flex items-center px-4 py-3 border-b border-border/20">
					<Skeleton emphasis="secondary" width="5rem" height="0.75rem" />
					<div class="ml-6">
						<Skeleton emphasis="tertiary" width="6rem" height="0.75rem" />
					</div>
					<div class="ml-auto">
						<Skeleton emphasis="secondary" width="4rem" height="0.75rem" />
					</div>
					<div class="ml-4">
						<Skeleton emphasis="tertiary" width="5rem" height="0.75rem" />
					</div>
				</div>
			{/each}
		</div>
	{:else if loadError}
		<div class="bg-card border border-border rounded-lg">
			<EmptyState
				title="Couldn't load traces"
				description={loadError}
			>
				{#snippet icon()}
					<div class="w-12 h-12 bg-muted/50 rounded-lg flex items-center justify-center">
						<AlertCircle class="w-6 h-6 text-red-400/70" />
					</div>
				{/snippet}
				{#snippet cta()}
					<Button variant="secondary" onclick={() => applyFilters()}>
						<RefreshCw class="w-3.5 h-3.5" />
						<span>Try again</span>
					</Button>
				{/snippet}
			</EmptyState>
		</div>
	{:else if filtered.length === 0}
		{#if summaries.length === 0}
			<div class="bg-card border border-border rounded-lg">
				<EmptyState title="No traces ingested in this window">
					{#snippet icon()}
						<div class="w-12 h-12 bg-muted/50 rounded-lg flex items-center justify-center">
							<GitBranch class="w-6 h-6 text-muted-foreground/40" />
						</div>
					{/snippet}
					<p class="text-xs text-muted-foreground mb-1">
						Configure your OTLP exporter to point at this Hub.
					</p>
					<div class="inline-block text-left mt-3 bg-muted/30 border border-border rounded-md px-4 py-3 text-[11px] font-mono text-muted-foreground space-y-1">
						<div><span class="text-muted-foreground/60">endpoint</span> &nbsp; <span class="text-foreground">/v1/traces</span></div>
						<div><span class="text-muted-foreground/60">headers</span> &nbsp; <span class="text-foreground">Authorization: Bearer wd_…</span></div>
						<div><span class="text-muted-foreground/60">scope</span> &nbsp;&nbsp;&nbsp;<span class="text-foreground">telemetry_ingest</span></div>
					</div>
				</EmptyState>
			</div>
		{:else}
			<div class="bg-card border border-border rounded-lg">
				<EmptyState
					title="No matches"
					description="No traces match the current filter. Try widening the time range or clearing the service filter."
				>
					{#snippet icon()}
						<div class="w-12 h-12 bg-muted/50 rounded-lg flex items-center justify-center">
							<GitBranch class="w-6 h-6 text-muted-foreground/40" />
						</div>
					{/snippet}
				</EmptyState>
			</div>
		{/if}
	{:else}
		<div class="bg-card border border-border rounded-lg overflow-hidden">
			<!-- Column headers -->
			<div class="flex items-center px-4 py-2 border-b border-border/30 text-[9px] font-medium text-muted-foreground uppercase tracking-wider">
				<div class="flex-1 min-w-0">Operation</div>
				<div class="w-32 shrink-0 ml-3 hidden lg:block">Service</div>
				<div class="w-28 shrink-0 ml-3">Trace ID</div>
				<div class="w-16 shrink-0 text-right ml-3">Spans</div>
				<div class="w-24 shrink-0 text-right ml-3 hidden sm:block">Duration</div>
				<div class="w-28 shrink-0 text-right ml-3 hidden md:block">Started</div>
				<div class="w-6 shrink-0 ml-2"></div>
			</div>

			<div class="divide-y divide-border/20">
				{#each filtered as t (t.trace_id)}
					<a
						href="/traces/{t.trace_id}"
						class="flex items-center px-4 py-3 hover:bg-card-elevated transition-colors group"
					>
						<!-- operation (root span name) — primary cell, sans-serif -->
						<div class="flex-1 min-w-0 text-sm text-foreground truncate">
							{t.root_name || 'unknown'}
						</div>

						<!-- service -->
						<div class="w-32 shrink-0 ml-3 hidden lg:block text-xs text-muted-foreground truncate font-mono">
							{t.service_name || '—'}
						</div>

						<!-- trace_id chip + copy button -->
						<div class="w-28 shrink-0 ml-3 flex items-center gap-1.5">
							<span class="text-[11px] text-muted-foreground/70 tabular-nums font-mono">{shortHex(t.trace_id)}</span>
							<button
								onclick={(e) => { e.preventDefault(); e.stopPropagation(); void copyTraceID(t.trace_id); }}
								class="opacity-0 group-hover:opacity-100 text-muted-foreground/60 hover:text-foreground transition-all"
								aria-label="Copy trace_id"
								title="Copy trace_id"
							>
								{#if copiedTraceID === t.trace_id}
									<Check class="w-3 h-3 text-emerald-400" />
								{:else}
									<Copy class="w-3 h-3" />
								{/if}
							</button>
						</div>

						<!-- spans -->
						<div class="w-16 shrink-0 text-right ml-3 tabular-nums text-xs text-muted-foreground font-mono">
							{t.span_count}
						</div>

						<!-- duration -->
						<div class="w-24 shrink-0 text-right ml-3 hidden sm:block tabular-nums text-xs {durationClass(t.duration_ns, t.has_error)} font-mono">
							{formatDuration(t.duration_ns)}
						</div>

						<!-- started -->
						<div class="w-28 shrink-0 text-right ml-3 hidden md:block tabular-nums text-xs text-muted-foreground/80 font-mono">
							{relativeTime(t.start_time)}
						</div>

						<!-- error dot -->
						<div class="w-6 shrink-0 ml-2 flex justify-end">
							{#if t.has_error}
								<span
									class="w-1.5 h-1.5 rounded-full bg-red-400 shadow-[0_0_4px_rgba(239,68,68,0.6)]"
									aria-label="Has errors"
								></span>
							{/if}
						</div>
					</a>
				{/each}
			</div>

			<div class="px-4 py-2 border-t border-border/30 text-[10px] text-muted-foreground/70 font-mono flex items-center justify-between gap-3">
				<span>{filtered.length} trace{filtered.length === 1 ? '' : 's'}</span>
				{#if errorsOnly && summaries.length > filtered.length}
					<span>{summaries.length - filtered.length} hidden by errors-only filter</span>
				{/if}
			</div>
		</div>

		<!-- Pagination control: dedicated row below the table for visibility. -->
		<div class="mt-4 flex justify-center">
			{#if hasMore}
				<Button
					variant="outline"
					disabled={loadingMore}
					onclick={() => void loadMore()}
				>
					{#if loadingMore}
						<RefreshCw class="w-3.5 h-3.5 animate-spin" />
						<span>Loading older traces…</span>
					{:else}
						<ChevronDown class="w-3.5 h-3.5" />
						<span>Load older traces</span>
					{/if}
				</Button>
			{:else}
				<span class="text-[11px] text-muted-foreground/60 font-mono">— end of {timeRange} window —</span>
			{/if}
		</div>
	{/if}
</div>
