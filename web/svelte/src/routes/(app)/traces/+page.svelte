<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { RefreshCw, Search, Copy, Check, ChevronDown } from 'lucide-svelte';
	import { Button, Input, Select, Tabs } from '@sylvester-francis/watchdog-ui';
	import { traces as tracesApi } from '$lib/api';
	import type { TraceSummary } from '$lib/types';

	type TimeRange = '1h' | '6h' | '24h';
	type AutoRefresh = 'off' | '15s' | '30s' | '60s';

	let summaries = $state<TraceSummary[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

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
			// Clipboard unavailable — user can still drag-select the visible prefix.
		}
	}

	function formatDuration(ns: number): string {
		if (ns < 1_000) return `${ns}ns`;
		if (ns < 1_000_000) return `${(ns / 1_000).toFixed(1)}µs`;
		if (ns < 1_000_000_000) return `${(ns / 1_000_000).toFixed(1)}ms`;
		return `${(ns / 1_000_000_000).toFixed(2)}s`;
	}

	function durationClass(ns: number, hasError: boolean): string {
		if (hasError) return 'text-destructive';
		if (ns >= 1_000_000_000) return 'text-warning';
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

<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-8 sm:px-6 sm:py-10">
	<!-- Page header -->
	<header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between sm:gap-4">
		<div class="min-w-0">
			<div class="flex items-center gap-2 font-mono tabular-nums text-xs text-muted-foreground">
				<span class="uppercase tracking-wider">Telemetry · Traces</span>
			</div>
			<h1 class="mt-1.5 text-2xl font-medium text-foreground sm:text-3xl">Traces</h1>
			<p class="mt-1 text-sm text-muted-foreground">Distributed traces ingested via OTLP.</p>
		</div>
		<Button variant="secondary" onclick={() => applyFilters()} aria-label="Refresh">
			<RefreshCw class="mr-1.5 h-3.5 w-3.5 {loading ? 'animate-spin' : ''}" />
			Refresh
		</Button>
	</header>

	<!-- Filter bar -->
	<div class="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center">
		<Tabs
			options={timeTabs as Array<{ value: string; label: string }>}
			value={timeRange}
			variant="pill"
			onchange={(v) => { timeRange = v as TimeRange; }}
		/>

		<div class="flex items-center gap-3 sm:ml-auto">
			<div class="w-44 sm:w-56">
				<Input bind:value={serviceFilter} placeholder="Filter by service...">
					{#snippet iconLeft()}
						<Search class="h-3.5 w-3.5" />
					{/snippet}
				</Input>
			</div>

			<label class="flex cursor-pointer select-none items-center gap-1.5 text-xs text-muted-foreground transition-colors hover:text-foreground">
				<input
					type="checkbox"
					bind:checked={errorsOnly}
					class="h-3.5 w-3.5 border-border bg-background accent-destructive focus:ring-1 focus:ring-ring"
				/>
				<span>Errors only</span>
			</label>

			<div class="w-24">
				<Select bind:value={autoRefresh} size="sm" aria-label="Auto-refresh">
					{#each refreshOptions as r}
						<option value={r.value}>↻ {r.label}</option>
					{/each}
				</Select>
			</div>
		</div>
	</div>

	<div class="mt-6">
		{#if loading}
			<div class="space-y-2">
				{#each Array(5) as _}
					<div class="flex animate-pulse items-center gap-6 py-3">
						<div class="h-3 w-20 bg-muted/50"></div>
						<div class="h-3 w-24 bg-muted/30"></div>
						<div class="ml-auto h-3 w-16 bg-muted/50"></div>
						<div class="h-3 w-20 bg-muted/30"></div>
					</div>
				{/each}
			</div>
		{:else if loadError}
			<section>
				<div class="border-b border-border pb-3">
					<h3 class="text-sm font-medium text-foreground">Couldn't load traces</h3>
				</div>
				<div class="flex flex-col items-start gap-3 pt-4 sm:flex-row sm:items-center sm:justify-between sm:gap-6">
					<p class="font-mono tabular-nums text-xs text-destructive">{loadError}</p>
					<Button variant="secondary" onclick={() => applyFilters()}>
						<RefreshCw class="mr-1.5 h-3.5 w-3.5" />
						<span>Try again</span>
					</Button>
				</div>
			</section>
		{:else if filtered.length === 0}
			{#if summaries.length === 0}
				<section>
					<div class="border-b border-border pb-3">
						<h3 class="text-sm font-medium text-foreground">No traces ingested in this window</h3>
					</div>
					<div class="pt-4">
						<p class="text-xs text-muted-foreground">Configure your OTLP exporter to point at this Hub.</p>
						<div class="mt-3 inline-block border border-border bg-background px-4 py-3 font-mono tabular-nums text-[11px] text-muted-foreground">
							<div class="grid grid-cols-[6rem_1fr] gap-x-2 gap-y-1">
								<span class="text-muted-foreground/60">endpoint</span>
								<span class="text-foreground">/v1/traces</span>
								<span class="text-muted-foreground/60">headers</span>
								<span class="text-foreground">Authorization: Bearer wd_…</span>
								<span class="text-muted-foreground/60">scope</span>
								<span class="text-foreground">telemetry_ingest</span>
							</div>
						</div>
					</div>
				</section>
			{:else}
				<section>
					<div class="border-b border-border pb-3">
						<h3 class="text-sm font-medium text-foreground">No matches</h3>
					</div>
					<p class="pt-4 text-xs text-muted-foreground">
						No traces match the current filter. Try widening the time range or clearing the service filter.
					</p>
				</section>
			{/if}
		{:else}
			<section>
				<div class="flex items-baseline justify-between gap-2 border-b border-border pb-3">
					<div class="flex items-baseline gap-2">
						<h3 class="text-sm font-medium text-foreground">Traces</h3>
						<span class="font-mono tabular-nums text-[11px] text-muted-foreground">{filtered.length}</span>
					</div>
					{#if errorsOnly && summaries.length > filtered.length}
						<span class="font-mono tabular-nums text-[11px] text-muted-foreground">
							{summaries.length - filtered.length} hidden by errors-only filter
						</span>
					{/if}
				</div>

				<!-- Column headers -->
				<div class="hidden items-center pb-2 pt-3 text-[9px] font-medium uppercase tracking-wider text-muted-foreground sm:flex">
					<div class="min-w-0 flex-1">Operation</div>
					<div class="ml-3 hidden w-32 shrink-0 lg:block">Service</div>
					<div class="ml-3 w-28 shrink-0">Trace ID</div>
					<div class="ml-3 w-16 shrink-0 text-right">Spans</div>
					<div class="ml-3 w-24 shrink-0 text-right">Duration</div>
					<div class="ml-3 hidden w-28 shrink-0 text-right md:block">Started</div>
					<div class="ml-2 w-4 shrink-0"></div>
				</div>

				<div class="divide-y divide-border/40">
					{#each filtered as t (t.trace_id)}
						<a
							href="/traces/{t.trace_id}"
							class="group flex items-center py-3 transition-colors hover:bg-muted/30"
						>
							<div class="min-w-0 flex-1 truncate text-sm text-foreground transition-colors group-hover:text-accent">
								{t.root_name || 'unknown'}
							</div>

							<div class="ml-3 hidden w-32 shrink-0 truncate font-mono tabular-nums text-xs text-muted-foreground lg:block">
								{t.service_name || '—'}
							</div>

							<div class="ml-3 flex w-28 shrink-0 items-center gap-1.5">
								<span class="font-mono tabular-nums text-[11px] text-muted-foreground/70">{shortHex(t.trace_id)}</span>
								<button
									onclick={(e) => { e.preventDefault(); e.stopPropagation(); void copyTraceID(t.trace_id); }}
									class="text-muted-foreground/60 opacity-0 transition-all hover:text-foreground group-hover:opacity-100"
									aria-label="Copy trace_id"
									title="Copy trace_id"
								>
									{#if copiedTraceID === t.trace_id}
										<Check class="h-3 w-3 text-success" />
									{:else}
										<Copy class="h-3 w-3" />
									{/if}
								</button>
							</div>

							<div class="ml-3 w-16 shrink-0 text-right font-mono tabular-nums text-xs text-muted-foreground">
								{t.span_count}
							</div>

							<div class="ml-3 w-24 shrink-0 text-right font-mono tabular-nums text-xs {durationClass(t.duration_ns, t.has_error)}">
								{formatDuration(t.duration_ns)}
							</div>

							<div class="ml-3 hidden w-28 shrink-0 text-right font-mono tabular-nums text-xs text-muted-foreground/80 md:block">
								{relativeTime(t.start_time)}
							</div>

							<div class="ml-2 flex w-4 shrink-0 justify-end">
								{#if t.has_error}
									<span class="inline-block h-1.5 w-1.5 rounded-full bg-destructive" aria-label="Has errors"></span>
								{/if}
							</div>
						</a>
					{/each}
				</div>

				<!-- Pagination control -->
				<div class="mt-6 flex justify-center">
					{#if hasMore}
						<Button variant="outline" disabled={loadingMore} onclick={() => void loadMore()}>
							{#if loadingMore}
								<RefreshCw class="mr-1.5 h-3.5 w-3.5 animate-spin" />
								<span>Loading older traces…</span>
							{:else}
								<ChevronDown class="mr-1.5 h-3.5 w-3.5" />
								<span>Load older traces</span>
							{/if}
						</Button>
					{:else}
						<span class="font-mono tabular-nums text-[11px] text-muted-foreground/60">— end of {timeRange} window —</span>
					{/if}
				</div>
			</section>
		{/if}
	</div>
</div>
