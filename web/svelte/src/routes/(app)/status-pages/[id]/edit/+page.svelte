<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { ChevronRight, AlertCircle } from 'lucide-svelte';
	import { Alert, Button, FormField } from '@sylvester-francis/watchdog-ui';
	import { statusPages as statusPagesApi } from '$lib/api';
	import { getToasts } from '$lib/stores/toast.svelte';
	import type { StatusPage } from '$lib/types';

	const toast = getToasts();

	interface AvailableMonitor {
		id: string;
		name: string;
		type: string;
		target: string;
		status: string;
	}

	let statusPage = $state<StatusPage | null>(null);
	let availableMonitors = $state<AvailableMonitor[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');
	let loadError = $state('');

	let name = $state('');
	let description = $state('');
	let isPublic = $state(false);
	let selectedMonitorIds = $state<Set<string>>(new Set());

	let pageId = $derived(page.params.id ?? '');

	function statusPipClass(status: string): string {
		if (status === 'up') return 'bg-success';
		if (status === 'down') return 'bg-destructive';
		if (status === 'degraded') return 'bg-warning';
		return 'bg-muted-foreground/50';
	}

	function toggleMonitor(id: string) {
		const next = new Set(selectedMonitorIds);
		if (next.has(id)) {
			next.delete(id);
		} else {
			next.add(id);
		}
		selectedMonitorIds = next;
	}

	async function loadData() {
		loading = true;
		loadError = '';

		try {
			const res = await statusPagesApi.getStatusPage(pageId);
			statusPage = res.data;
			availableMonitors = res.available_monitors ?? [];

			name = statusPage.name;
			description = statusPage.description ?? '';
			isPublic = statusPage.is_public;
			selectedMonitorIds = new Set(statusPage.monitor_ids ?? []);
		} catch (err) {
			const msg = err instanceof Error ? err.message : 'Failed to load status page';
			loadError = msg;
			toast.error(msg);
		} finally {
			loading = false;
		}
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();

		if (!name.trim()) {
			error = 'Name is required';
			return;
		}

		saving = true;
		error = '';

		try {
			await statusPagesApi.updateStatusPage(pageId, {
				name: name.trim(),
				description: description.trim(),
				is_public: isPublic,
				monitor_ids: Array.from(selectedMonitorIds)
			});
			toast.success('Status page updated');
			goto(`/status-pages`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update status page';
		} finally {
			saving = false;
		}
	}

	onMount(() => {
		loadData();
	});

	const inputClass =
		'w-full border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground/50 focus:border-foreground/30 focus:outline-none focus:ring-0';
</script>

<svelte:head>
	<title>{statusPage?.name ?? 'Edit Status Page'} - WatchDog</title>
</svelte:head>

<div class="animate-fade-in-up mx-auto max-w-[1080px] px-4 py-8 sm:px-6 sm:py-10">
	{#if loading}
		<div class="space-y-4">
			<div class="h-4 w-56 animate-pulse bg-muted/50"></div>
			<div class="space-y-3 pt-4">
				<div class="h-4 w-20 animate-pulse bg-muted/50"></div>
				<div class="h-10 w-full animate-pulse bg-muted/30"></div>
				<div class="h-4 w-24 animate-pulse bg-muted/50"></div>
				<div class="h-20 w-full animate-pulse bg-muted/30"></div>
			</div>
		</div>
	{:else if loadError && !statusPage}
		<p class="text-sm font-medium text-foreground">Failed to load status page</p>
		<p class="mt-1 font-mono tabular-nums text-xs text-destructive">{loadError}</p>
		<a
			href="/status-pages"
			class="mt-4 inline-block text-sm text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline"
		>
			← Back to Status Pages
		</a>
	{:else if statusPage}
		<!-- Breadcrumb -->
		<nav class="flex items-center gap-1.5 text-xs">
			<a href="/status-pages" class="text-muted-foreground transition-colors hover:text-foreground">
				Status Pages
			</a>
			<ChevronRight class="h-3 w-3 text-muted-foreground/40" />
			<span class="truncate max-w-[200px] font-mono tabular-nums text-foreground">{statusPage.name}</span>
		</nav>

		<!-- Header -->
		<header class="mt-6">
			<div class="flex items-center gap-2 font-mono tabular-nums text-xs text-muted-foreground">
				<span class="uppercase tracking-wider">Edit · Status Page</span>
			</div>
			<h1 class="mt-1.5 truncate text-2xl font-medium text-foreground sm:text-3xl">{statusPage.name}</h1>
		</header>

		<form onsubmit={handleSubmit} class="mt-8 space-y-8">
			{#if error}
				<Alert tone="down">
					{#snippet icon()}<AlertCircle class="h-3.5 w-3.5" />{/snippet}
					{error}
				</Alert>
			{/if}

			<!-- Configuration -->
			<section>
				<div class="border-b border-border pb-3">
					<h3 class="text-sm font-medium text-foreground">Configuration</h3>
				</div>
				<div class="space-y-4 pt-4">
					<FormField label="Name" htmlFor="sp-name" required>
						<input
							id="sp-name"
							type="text"
							bind:value={name}
							required
							placeholder="My Status Page"
							class={inputClass}
						/>
					</FormField>

					<FormField label="Description" htmlFor="sp-description">
						<textarea
							id="sp-description"
							bind:value={description}
							placeholder="A brief description of what this status page covers..."
							rows="3"
							class={inputClass}
						></textarea>
					</FormField>

					<div class="flex items-center gap-3">
						<input
							id="sp-public"
							type="checkbox"
							bind:checked={isPublic}
							class="h-4 w-4 border-border bg-background accent-accent focus:ring-ring"
						/>
						<label for="sp-public" class="text-xs text-foreground">
							Make this page publicly accessible
						</label>
					</div>
				</div>
			</section>

			<!-- Monitors -->
			<section>
				<div class="flex items-baseline gap-2 border-b border-border pb-3">
					<h3 class="text-sm font-medium text-foreground">Monitors</h3>
					{#if selectedMonitorIds.size > 0}
						<span class="font-mono tabular-nums text-[11px] text-muted-foreground">
							{selectedMonitorIds.size} selected
						</span>
					{/if}
				</div>

				{#if availableMonitors.length === 0}
					<p class="pt-4 text-xs text-muted-foreground">
						No monitors available. Create monitors first, then assign them here.
					</p>
				{:else}
					<div class="max-h-64 divide-y divide-border/40 overflow-y-auto">
						{#each availableMonitors as monitor (monitor.id)}
							<button
								type="button"
								onclick={() => toggleMonitor(monitor.id)}
								class="flex w-full items-center gap-3 py-3 text-left transition-colors hover:bg-muted/30"
							>
								<input
									type="checkbox"
									checked={selectedMonitorIds.has(monitor.id)}
									tabindex={-1}
									class="pointer-events-none h-3.5 w-3.5 border-border bg-background accent-accent"
								/>
								<span class="inline-block h-1.5 w-1.5 shrink-0 rounded-full {statusPipClass(monitor.status)}"></span>
								<div class="min-w-0 flex-1">
									<div class="flex items-center gap-2">
										<span class="truncate text-sm text-foreground">{monitor.name}</span>
										<span class="font-mono tabular-nums text-[10px] uppercase tracking-wider text-muted-foreground">
											{monitor.type}
										</span>
									</div>
									<p class="mt-0.5 truncate font-mono tabular-nums text-[11px] text-muted-foreground">
										{monitor.target}
									</p>
								</div>
							</button>
						{/each}
					</div>
				{/if}
			</section>

			<!-- Footer -->
			<div class="flex items-center justify-end gap-4 border-t border-border pt-4 text-xs">
				<a
					href="/status-pages"
					class="text-foreground/70 underline-offset-4 transition-colors hover:text-foreground hover:underline"
				>
					Cancel
				</a>
				<Button variant="primary" size="sm" type="submit" disabled={saving}>
					{saving ? 'Saving…' : 'Save Changes'}
				</Button>
			</div>
		</form>
	{/if}
</div>
